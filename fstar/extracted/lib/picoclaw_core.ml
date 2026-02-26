(** PicoClaw Verified Core -- OCaml implementation matching F* specifications.

    This module provides the core logic extracted from the F* verified
    specifications. Each submodule mirrors its F* counterpart and preserves
    the verified properties through faithful translation.

    In the final architecture, this file is replaced by machine-extracted
    OCaml from `fstar --codegen OCaml`. The hand-written implementation
    serves as the reference until extraction tooling is fully wired. *)

module Types = struct
  type role = User | Assistant | System | Tool

  let role_to_string = function
    | User -> "user"
    | Assistant -> "assistant"
    | System -> "system"
    | Tool -> "tool"

  let string_to_role = function
    | "user" -> Some User
    | "assistant" -> Some Assistant
    | "system" -> Some System
    | "tool" -> Some Tool
    | _ -> None

  type function_call = {
    fc_name : string;
    fc_arguments : string;
  }

  type tool_call = {
    tc_id : string;
    tc_type : string;
    tc_function : function_call option;
    tc_name : string;
  }

  type usage_info = {
    ui_prompt_tokens : int;
    ui_completion_tokens : int;
    ui_total_tokens : int;
  }

  type message = {
    msg_role : role;
    msg_content : string;
    msg_reasoning_content : string;
    msg_tool_calls : tool_call list;
    msg_tool_call_id : string;
  }

  type llm_response = {
    lr_content : string;
    lr_reasoning_content : string;
    lr_tool_calls : tool_call list;
    lr_finish_reason : string;
    lr_usage : usage_info option;
  }

  type tool_result = {
    tr_for_llm : string;
    tr_for_user : string;
    tr_silent : bool;
    tr_is_error : bool;
    tr_async : bool;
  }

  type tool_definition = {
    td_name : string;
    td_description : string;
    td_parameters : string;
  }

  type dm_scope =
    | DMScopeMain
    | DMScopePerPeer
    | DMScopePerChannelPeer
    | DMScopePerAccountChannelPeer

  let string_to_dm_scope = function
    | "per-peer" -> DMScopePerPeer
    | "per-channel-peer" -> DMScopePerChannelPeer
    | "per-account-channel-peer" -> DMScopePerAccountChannelPeer
    | _ -> DMScopeMain

  type route_peer = {
    rp_kind : string;
    rp_id : string;
  }

  type route_input = {
    ri_channel : string;
    ri_account_id : string;
    ri_peer : route_peer option;
    ri_parent_peer : route_peer option;
    ri_guild_id : string;
    ri_team_id : string;
  }

  type binding_match = {
    bm_channel : string;
    bm_account_id : string;
    bm_peer : route_peer option;
    bm_guild_id : string;
    bm_team_id : string;
  }

  type agent_binding = {
    ab_agent_id : string;
    ab_match : binding_match;
  }

  type match_reason =
    | MatchedByPeer
    | MatchedByParentPeer
    | MatchedByGuild
    | MatchedByTeam
    | MatchedByAccount
    | MatchedByChannelWildcard
    | MatchedByDefault

  let match_reason_to_string = function
    | MatchedByPeer -> "peer"
    | MatchedByParentPeer -> "parent_peer"
    | MatchedByGuild -> "guild"
    | MatchedByTeam -> "team"
    | MatchedByAccount -> "account"
    | MatchedByChannelWildcard -> "channel_wildcard"
    | MatchedByDefault -> "default"

  type resolved_route = {
    rr_agent_id : string;
    rr_channel : string;
    rr_account_id : string;
    rr_session_key : string;
    rr_main_session_key : string;
    rr_matched_by : match_reason;
  }
end

module Routing = struct
  open Types

  let default_agent_id = "main"

  let build_main_session_key agent_id =
    String.concat ":" ["agent"; agent_id; "main"]

  let build_peer_session_key scope agent_id channel account_id (peer : route_peer) =
    let prefix = String.concat ":" ["agent"; agent_id] in
    match scope with
    | DMScopeMain ->
      String.concat ":" [prefix; "main"]
    | DMScopePerPeer ->
      String.concat ":" [prefix; "direct"; peer.rp_id]
    | DMScopePerChannelPeer ->
      String.concat ":" [prefix; channel; "direct"; peer.rp_id]
    | DMScopePerAccountChannelPeer ->
      String.concat ":" [prefix; account_id; channel; "direct"; peer.rp_id]

  let build_group_session_key agent_id channel group_id =
    String.concat ":" ["agent"; agent_id; channel; "group"; group_id]

  let matches_peer (b : agent_binding) (input : route_input) =
    match b.ab_match.bm_peer, input.ri_peer with
    | Some bp, Some ip ->
      bp.rp_kind = ip.rp_kind && bp.rp_id = ip.rp_id &&
      b.ab_match.bm_channel = input.ri_channel
    | _, _ -> false

  let matches_parent_peer (b : agent_binding) (input : route_input) =
    match b.ab_match.bm_peer, input.ri_parent_peer with
    | Some bp, Some pp ->
      bp.rp_kind = pp.rp_kind && bp.rp_id = pp.rp_id &&
      b.ab_match.bm_channel = input.ri_channel
    | _, _ -> false

  let matches_guild (b : agent_binding) (input : route_input) =
    b.ab_match.bm_guild_id <> "" &&
    b.ab_match.bm_guild_id = input.ri_guild_id &&
    b.ab_match.bm_channel = input.ri_channel

  let matches_team (b : agent_binding) (input : route_input) =
    b.ab_match.bm_team_id <> "" &&
    b.ab_match.bm_team_id = input.ri_team_id &&
    b.ab_match.bm_channel = input.ri_channel

  let matches_account (b : agent_binding) (input : route_input) =
    b.ab_match.bm_account_id <> "" &&
    b.ab_match.bm_account_id <> "*" &&
    b.ab_match.bm_account_id = input.ri_account_id &&
    b.ab_match.bm_channel = input.ri_channel &&
    (match b.ab_match.bm_peer with None -> true | _ -> false)

  let matches_channel_wildcard (b : agent_binding) (input : route_input) =
    b.ab_match.bm_channel = input.ri_channel &&
    (b.ab_match.bm_account_id = "*" || b.ab_match.bm_account_id = "") &&
    (match b.ab_match.bm_peer with None -> true | _ -> false) &&
    b.ab_match.bm_guild_id = "" &&
    b.ab_match.bm_team_id = ""

  let find_binding pred bindings =
    List.find_opt pred bindings

  let resolve_route bindings default_id scope (input : route_input) =
    let make_route agent_id reason =
      let session_key =
        match input.ri_peer with
        | Some peer ->
          if peer.rp_kind = "direct" then
            build_peer_session_key scope agent_id input.ri_channel input.ri_account_id peer
          else
            build_group_session_key agent_id input.ri_channel peer.rp_id
        | None ->
          build_main_session_key agent_id
      in
      { rr_agent_id = agent_id;
        rr_channel = input.ri_channel;
        rr_account_id = input.ri_account_id;
        rr_session_key = session_key;
        rr_main_session_key = build_main_session_key agent_id;
        rr_matched_by = reason }
    in
    (* 7-level cascade *)
    match find_binding (fun b -> matches_peer b input) bindings with
    | Some b -> make_route b.ab_agent_id MatchedByPeer
    | None ->
    match find_binding (fun b -> matches_parent_peer b input) bindings with
    | Some b -> make_route b.ab_agent_id MatchedByParentPeer
    | None ->
    match find_binding (fun b -> matches_guild b input) bindings with
    | Some b -> make_route b.ab_agent_id MatchedByGuild
    | None ->
    match find_binding (fun b -> matches_team b input) bindings with
    | Some b -> make_route b.ab_agent_id MatchedByTeam
    | None ->
    match find_binding (fun b -> matches_account b input) bindings with
    | Some b -> make_route b.ab_agent_id MatchedByAccount
    | None ->
    match find_binding (fun b -> matches_channel_wildcard b input) bindings with
    | Some b -> make_route b.ab_agent_id MatchedByChannelWildcard
    | None ->
    (* Level 7: default -- always succeeds *)
    make_route default_id MatchedByDefault
end

module AuditLog = struct
  type audit_event =
    | RouteResolved of string
    | ToolAuthorized of string
    | ToolDenied of string
    | ToolExecuted of string
    | LLMCallStarted of string
    | LLMCallCompleted of string
    | SessionCreated of string
    | MessageProcessed of string
    | ApertureMetering of string
    | CerbosDecision of string

  type audit_entry = {
    ae_sequence : int;
    ae_timestamp : int;
    ae_event : audit_event;
    ae_agent_id : string;
    ae_session_key : string;
    ae_prev_hash : string;
    ae_request_id : string;
  }

  type audit_log = audit_entry list

  let hash_entry (e : audit_entry) : string =
    (* SHA256 would be used in production; placeholder hash for now *)
    String.concat ":" [
      string_of_int e.ae_sequence;
      string_of_int e.ae_timestamp;
      e.ae_agent_id;
      e.ae_session_key;
      e.ae_prev_hash;
      e.ae_request_id;
    ]

  let last_hash (log : audit_log) : string =
    match log with
    | [] -> ""
    | _ ->
      let last = List.nth log (List.length log - 1) in
      hash_entry last

  let next_sequence (log : audit_log) : int =
    List.length log

  let append_entry log timestamp event agent_id session_key request_id =
    let entry = {
      ae_sequence = next_sequence log;
      ae_timestamp = timestamp;
      ae_event = event;
      ae_agent_id = agent_id;
      ae_session_key = session_key;
      ae_prev_hash = last_hash log;
      ae_request_id = request_id;
    } in
    log @ [entry]

  let chain_valid_pair prev cur =
    cur.ae_prev_hash = hash_entry prev &&
    cur.ae_sequence = prev.ae_sequence + 1

  let rec chain_valid = function
    | [] | [_] -> true
    | x :: (y :: _ as rest) ->
      chain_valid_pair x y && chain_valid rest

  let log_route log timestamp agent_id session_key request_id =
    append_entry log timestamp (RouteResolved agent_id) agent_id session_key request_id

  let log_tool_auth log timestamp tool_name authorized agent_id session_key request_id =
    let event = if authorized then ToolAuthorized tool_name
                else ToolDenied tool_name in
    append_entry log timestamp event agent_id session_key request_id

  let log_tool_exec log timestamp tool_name agent_id session_key request_id =
    append_entry log timestamp (ToolExecuted tool_name) agent_id session_key request_id

  let log_llm_call log timestamp model completed agent_id session_key request_id =
    let event = if completed then LLMCallCompleted model
                else LLMCallStarted model in
    append_entry log timestamp event agent_id session_key request_id

  let event_to_string = function
    | RouteResolved s -> "route_resolved:" ^ s
    | ToolAuthorized s -> "tool_authorized:" ^ s
    | ToolDenied s -> "tool_denied:" ^ s
    | ToolExecuted s -> "tool_executed:" ^ s
    | LLMCallStarted s -> "llm_call_started:" ^ s
    | LLMCallCompleted s -> "llm_call_completed:" ^ s
    | SessionCreated s -> "session_created:" ^ s
    | MessageProcessed s -> "message_processed:" ^ s
    | ApertureMetering s -> "aperture_metering:" ^ s
    | CerbosDecision s -> "cerbos_decision:" ^ s
end

module ToolAuth = struct
  type auth_level = AlwaysAllowed | RequiresGrant | AlwaysDenied

  type policy_entry = {
    pe_tool_name : string;
    pe_level : auth_level;
  }

  type grant = {
    g_tool_name : string;
    g_agent_id : string;
    g_issued_at : int;
  }

  type auth_decision =
    | Authorized of grant
    | Denied of string

  let lookup_policy tool_name policy =
    match List.find_opt (fun e -> e.pe_tool_name = tool_name) policy with
    | Some entry -> entry.pe_level
    | None -> RequiresGrant

  let has_grant tool_name agent_id grants =
    List.find_opt (fun g ->
      g.g_tool_name = tool_name && g.g_agent_id = agent_id
    ) grants

  let authorize_tool tool_name agent_id policy grants timestamp =
    match lookup_policy tool_name policy with
    | AlwaysDenied ->
      Denied ("tool '" ^ tool_name ^ "' is always denied")
    | AlwaysAllowed ->
      Authorized { g_tool_name = tool_name; g_agent_id = agent_id; g_issued_at = timestamp }
    | RequiresGrant ->
      match has_grant tool_name agent_id grants with
      | Some g -> Authorized g
      | None ->
        Denied ("no grant for tool '" ^ tool_name ^ "' agent '" ^ agent_id ^ "'")

  let authorize_tools tool_names agent_id policy grants timestamp =
    List.map (fun name -> authorize_tool name agent_id policy grants timestamp) tool_names
end

module Session = struct
  open Types

  type session = {
    s_key : string;
    s_messages : message list;
    s_summary : string;
    s_message_count : int;
  }

  let empty_session key =
    { s_key = key; s_messages = []; s_summary = ""; s_message_count = 0 }

  let add_message s msg =
    { s_key = s.s_key;
      s_messages = s.s_messages @ [msg];
      s_summary = s.s_summary;
      s_message_count = s.s_message_count + 1 }

  let get_history s = s.s_messages
  let get_summary s = s.s_summary
  let get_total_count s = s.s_message_count

  let summarize s new_summary keep_last =
    let msgs = s.s_messages in
    let len = List.length msgs in
    let kept =
      if keep_last >= len then msgs
      else
        let start = len - keep_last in
        let rec drop n = function
          | [] -> []
          | _ :: rest when n > 0 -> drop (n - 1) rest
          | l -> l
        in
        drop start msgs
    in
    let combined =
      if s.s_summary = "" then new_summary
      else s.s_summary ^ "\n\n" ^ new_summary
    in
    { s_key = s.s_key;
      s_messages = kept;
      s_summary = combined;
      s_message_count = s.s_message_count }

  let needs_summarization s max_messages =
    List.length s.s_messages > max_messages

  let build_context s =
    if s.s_summary = "" then s.s_messages
    else
      let summary_msg = {
        msg_role = System;
        msg_content = "Previous conversation summary:\n" ^ s.s_summary;
        msg_reasoning_content = "";
        msg_tool_calls = [];
        msg_tool_call_id = "";
      } in
      summary_msg :: s.s_messages
end

module AgentLoop = struct
  open Types
  open Session

  type loop_state = {
    ls_session : session;
    ls_audit_log : AuditLog.audit_log;
    ls_messages : message list;
    ls_iteration : int;
    ls_agent_id : string;
    ls_request_id : string;
  }

  type iteration_outcome =
    | FinalResponse of string * loop_state
    | NeedsToolCalls of tool_call list * loop_state
    | FuelExhausted of loop_state
    | IterationError of string * loop_state

  let classify_response response state fuel =
    if fuel = 0 then FuelExhausted state
    else if List.length response.lr_tool_calls > 0 then
      NeedsToolCalls (response.lr_tool_calls, state)
    else
      FinalResponse (response.lr_content, state)

  let step_iteration state response fuel timestamp =
    let log1 = AuditLog.log_llm_call
      state.ls_audit_log timestamp "model"
      true state.ls_agent_id
      state.ls_session.s_key state.ls_request_id
    in
    let assistant_msg = {
      msg_role = Assistant;
      msg_content = response.lr_content;
      msg_reasoning_content = response.lr_reasoning_content;
      msg_tool_calls = response.lr_tool_calls;
      msg_tool_call_id = "";
    } in
    let new_messages = state.ls_messages @ [assistant_msg] in
    let new_state = {
      ls_session = add_message state.ls_session assistant_msg;
      ls_audit_log = log1;
      ls_messages = new_messages;
      ls_iteration = state.ls_iteration + 1;
      ls_agent_id = state.ls_agent_id;
      ls_request_id = state.ls_request_id;
    } in
    let outcome = classify_response response new_state fuel in
    (outcome, log1)

  let init_loop (route : Types.resolved_route) sess user_content request_id timestamp =
    let user_msg = {
      msg_role = User;
      msg_content = user_content;
      msg_reasoning_content = "";
      msg_tool_calls = [];
      msg_tool_call_id = "";
    } in
    let sess_with_msg = add_message sess user_msg in
    let context = build_context sess_with_msg in
    let initial_log = AuditLog.log_route [] timestamp route.rr_agent_id
                        route.rr_session_key request_id in
    { ls_session = sess_with_msg;
      ls_audit_log = initial_log;
      ls_messages = context;
      ls_iteration = 0;
      ls_agent_id = route.rr_agent_id;
      ls_request_id = request_id }

  let authorize_tool_calls calls agent_id policy grants timestamp =
    List.map (fun (tc : tool_call) ->
      let decision = ToolAuth.authorize_tool tc.tc_name agent_id policy grants timestamp in
      (tc, decision)
    ) calls

  let filter_authorized decisions =
    List.filter_map (fun (tc, d) ->
      match d with ToolAuth.Authorized _ -> Some tc | _ -> None
    ) decisions

  let filter_denied decisions =
    List.filter_map (fun (tc, d) ->
      match d with
      | ToolAuth.Denied r -> Some (tc, r)
      | _ -> None
    ) decisions
end
