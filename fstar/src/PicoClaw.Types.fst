(**
  PicoClaw.Types -- Wire types mirroring protocoltypes/types.go and config structs.

  These types are the verified boundary between the Go gateway and the F* core.
  All data crossing the JSON-RPC boundary is expressed in these types.
*)
module PicoClaw.Types

open FStar.String
open FStar.List.Tot

(** Tool call function details *)
type function_call = {
  fc_name: string;
  fc_arguments: string;  (* JSON-encoded arguments *)
}

(** Tool invocation from LLM *)
type tool_call = {
  tc_id: string;
  tc_type: string;       (* "function" *)
  tc_function: option function_call;
  tc_name: string;       (* Parsed name for internal use *)
}

(** Token usage information *)
type usage_info = {
  ui_prompt_tokens: nat;
  ui_completion_tokens: nat;
  ui_total_tokens: nat;
}

(** Message roles *)
type role =
  | User
  | Assistant
  | System
  | Tool

let role_to_string (r: role) : string =
  match r with
  | User -> "user"
  | Assistant -> "assistant"
  | System -> "system"
  | Tool -> "tool"

let string_to_role (s: string) : option role =
  if s = "user" then Some User
  else if s = "assistant" then Some Assistant
  else if s = "system" then Some System
  else if s = "tool" then Some Tool
  else None

(** Unified message format *)
type message = {
  msg_role: role;
  msg_content: string;
  msg_reasoning_content: string;
  msg_tool_calls: list tool_call;
  msg_tool_call_id: string;
}

(** LLM response from provider *)
type llm_response = {
  lr_content: string;
  lr_reasoning_content: string;
  lr_tool_calls: list tool_call;
  lr_finish_reason: string;  (* "stop", "tool_calls", "length" *)
  lr_usage: option usage_info;
}

(** Tool result semantics *)
type tool_result = {
  tr_for_llm: string;    (* Content for LLM context *)
  tr_for_user: string;   (* Content for user display *)
  tr_silent: bool;        (* Suppress user message *)
  tr_is_error: bool;      (* Execution failed *)
  tr_async: bool;         (* Will complete later *)
}

(** Tool definition for LLM *)
type tool_definition = {
  td_name: string;
  td_description: string;
  td_parameters: string;  (* JSON Schema as string *)
}

(** Model configuration *)
type model_config = {
  mc_model_name: string;
  mc_model: string;       (* protocol/model-identifier *)
  mc_api_base: string;
}

(** DM scope levels for session isolation *)
type dm_scope =
  | DMScopeMain                  (* Single main session *)
  | DMScopePerPeer               (* One session per peer globally *)
  | DMScopePerChannelPeer        (* One session per (channel, peer) *)
  | DMScopePerAccountChannelPeer (* One session per (account, channel, peer) *)

let string_to_dm_scope (s: string) : dm_scope =
  if s = "per-peer" then DMScopePerPeer
  else if s = "per-channel-peer" then DMScopePerChannelPeer
  else if s = "per-account-channel-peer" then DMScopePerAccountChannelPeer
  else DMScopeMain

(** Peer identity for routing *)
type route_peer = {
  rp_kind: string;  (* "direct", "group", etc. *)
  rp_id: string;
}

(** Routing input from gateway *)
type route_input = {
  ri_channel: string;
  ri_account_id: string;
  ri_peer: option route_peer;
  ri_parent_peer: option route_peer;
  ri_guild_id: string;
  ri_team_id: string;
}

(** Binding match configuration *)
type binding_match = {
  bm_channel: string;
  bm_account_id: string;
  bm_peer: option route_peer;
  bm_guild_id: string;
  bm_team_id: string;
}

(** Agent binding: maps match criteria to agent ID *)
type agent_binding = {
  ab_agent_id: string;
  ab_match: binding_match;
}

(** How a route was resolved *)
type match_reason =
  | MatchedByPeer
  | MatchedByParentPeer
  | MatchedByGuild
  | MatchedByTeam
  | MatchedByAccount
  | MatchedByChannelWildcard
  | MatchedByDefault

let match_reason_to_string (r: match_reason) : string =
  match r with
  | MatchedByPeer -> "peer"
  | MatchedByParentPeer -> "parent_peer"
  | MatchedByGuild -> "guild"
  | MatchedByTeam -> "team"
  | MatchedByAccount -> "account"
  | MatchedByChannelWildcard -> "channel_wildcard"
  | MatchedByDefault -> "default"

(** Resolved route output *)
type resolved_route = {
  rr_agent_id: string;
  rr_channel: string;
  rr_account_id: string;
  rr_session_key: string;
  rr_main_session_key: string;
  rr_matched_by: match_reason;
}
