(** JSON serialization/deserialization for the PicoClaw wire protocol.

    Translates between OCaml types (from picoclaw_core.ml) and JSON
    (via Yojson). This module handles the JSON-RPC boundary between
    the verified core and the Go gateway. *)

open Picoclaw_core

(** ─── JSON -> OCaml ─────────────────────────────────────────────── *)

let string_of_json = function
  | `String s -> s
  | _ -> ""

let int_of_json = function
  | `Int n -> n
  | _ -> 0

let bool_of_json = function
  | `Bool b -> b
  | _ -> false

let list_of_json f = function
  | `List l -> List.map f l
  | _ -> []

let option_of_json f = function
  | `Null -> None
  | j -> Some (f j)

let assoc_field key = function
  | `Assoc l -> (try List.assoc key l with Not_found -> `Null)
  | _ -> `Null

(** Parse a route_peer from JSON *)
let route_peer_of_json j : Types.route_peer =
  { rp_kind = string_of_json (assoc_field "kind" j);
    rp_id = string_of_json (assoc_field "id" j) }

(** Parse a route_input from JSON *)
let route_input_of_json j : Types.route_input =
  { ri_channel = string_of_json (assoc_field "channel" j);
    ri_account_id = string_of_json (assoc_field "account_id" j);
    ri_peer = option_of_json route_peer_of_json (assoc_field "peer" j);
    ri_parent_peer = option_of_json route_peer_of_json (assoc_field "parent_peer" j);
    ri_guild_id = string_of_json (assoc_field "guild_id" j);
    ri_team_id = string_of_json (assoc_field "team_id" j) }

(** Parse a binding_match from JSON *)
let binding_match_of_json j : Types.binding_match =
  { bm_channel = string_of_json (assoc_field "channel" j);
    bm_account_id = string_of_json (assoc_field "account_id" j);
    bm_peer = option_of_json route_peer_of_json (assoc_field "peer" j);
    bm_guild_id = string_of_json (assoc_field "guild_id" j);
    bm_team_id = string_of_json (assoc_field "team_id" j) }

(** Parse an agent_binding from JSON *)
let agent_binding_of_json j : Types.agent_binding =
  { ab_agent_id = string_of_json (assoc_field "agent_id" j);
    ab_match = binding_match_of_json (assoc_field "match" j) }

(** Parse a tool_definition from JSON *)
let tool_definition_of_json j : Types.tool_definition =
  { td_name = string_of_json (assoc_field "name" j);
    td_description = string_of_json (assoc_field "description" j);
    td_parameters = string_of_json (assoc_field "parameters" j) }

(** Parse a function_call from JSON *)
let function_call_of_json j : Types.function_call =
  { fc_name = string_of_json (assoc_field "name" j);
    fc_arguments = string_of_json (assoc_field "arguments" j) }

(** Parse a tool_call from JSON *)
let tool_call_of_json j : Types.tool_call =
  { tc_id = string_of_json (assoc_field "id" j);
    tc_type = string_of_json (assoc_field "type" j);
    tc_function = option_of_json function_call_of_json (assoc_field "function" j);
    tc_name = string_of_json (assoc_field "name" j) }

(** Parse a usage_info from JSON *)
let usage_info_of_json j : Types.usage_info =
  { ui_prompt_tokens = int_of_json (assoc_field "prompt_tokens" j);
    ui_completion_tokens = int_of_json (assoc_field "completion_tokens" j);
    ui_total_tokens = int_of_json (assoc_field "total_tokens" j) }

(** Parse a tool_result from JSON *)
let tool_result_of_json j : Types.tool_result =
  { tr_for_llm = string_of_json (assoc_field "for_llm" j);
    tr_for_user = string_of_json (assoc_field "for_user" j);
    tr_silent = bool_of_json (assoc_field "silent" j);
    tr_is_error = bool_of_json (assoc_field "is_error" j);
    tr_async = bool_of_json (assoc_field "async" j) }

(** Parse an llm_response from JSON *)
let llm_response_of_json j : Types.llm_response =
  { lr_content = string_of_json (assoc_field "content" j);
    lr_reasoning_content = string_of_json (assoc_field "reasoning_content" j);
    lr_tool_calls = list_of_json tool_call_of_json (assoc_field "tool_calls" j);
    lr_finish_reason = string_of_json (assoc_field "finish_reason" j);
    lr_usage = option_of_json usage_info_of_json (assoc_field "usage" j) }

(** Parse a message from JSON *)
let message_of_json j : Types.message =
  let role_str = string_of_json (assoc_field "role" j) in
  let role = match Types.string_to_role role_str with
    | Some r -> r
    | None -> Types.User
  in
  { msg_role = role;
    msg_content = string_of_json (assoc_field "content" j);
    msg_reasoning_content = string_of_json (assoc_field "reasoning_content" j);
    msg_tool_calls = list_of_json tool_call_of_json (assoc_field "tool_calls" j);
    msg_tool_call_id = string_of_json (assoc_field "tool_call_id" j) }

(** Parse a process_message request params *)
type process_message_params = {
  pm_route_input : Types.route_input;
  pm_content : string;
  pm_bindings : Types.agent_binding list;
  pm_default_agent : string;
  pm_dm_scope : Types.dm_scope;
  pm_tool_definitions : Types.tool_definition list;
  pm_max_iterations : int;
  pm_request_id : string;
}

let process_message_params_of_json j =
  { pm_route_input = route_input_of_json (assoc_field "route_input" j);
    pm_content = string_of_json (assoc_field "content" j);
    pm_bindings = list_of_json agent_binding_of_json (assoc_field "bindings" j);
    pm_default_agent = string_of_json (assoc_field "default_agent" j);
    pm_dm_scope = Types.string_to_dm_scope (string_of_json (assoc_field "dm_scope" j));
    pm_tool_definitions = list_of_json tool_definition_of_json (assoc_field "tool_definitions" j);
    pm_max_iterations = int_of_json (assoc_field "max_iterations" j);
    pm_request_id = string_of_json (assoc_field "request_id" j) }

(** ─── OCaml -> JSON ─────────────────────────────────────────────── *)

let json_of_match_reason r =
  `String (Types.match_reason_to_string r)

let json_of_resolved_route (r : Types.resolved_route) =
  `Assoc [
    "agent_id", `String r.rr_agent_id;
    "channel", `String r.rr_channel;
    "account_id", `String r.rr_account_id;
    "session_key", `String r.rr_session_key;
    "main_session_key", `String r.rr_main_session_key;
    "matched_by", json_of_match_reason r.rr_matched_by;
  ]

let json_of_audit_event = function
  | AuditLog.RouteResolved s -> `Assoc ["type", `String "route_resolved"; "detail", `String s]
  | AuditLog.ToolAuthorized s -> `Assoc ["type", `String "tool_authorized"; "detail", `String s]
  | AuditLog.ToolDenied s -> `Assoc ["type", `String "tool_denied"; "detail", `String s]
  | AuditLog.ToolExecuted s -> `Assoc ["type", `String "tool_executed"; "detail", `String s]
  | AuditLog.LLMCallStarted s -> `Assoc ["type", `String "llm_call_started"; "detail", `String s]
  | AuditLog.LLMCallCompleted s -> `Assoc ["type", `String "llm_call_completed"; "detail", `String s]
  | AuditLog.SessionCreated s -> `Assoc ["type", `String "session_created"; "detail", `String s]
  | AuditLog.MessageProcessed s -> `Assoc ["type", `String "message_processed"; "detail", `String s]
  | AuditLog.ApertureMetering s -> `Assoc ["type", `String "aperture_metering"; "detail", `String s]
  | AuditLog.CerbosDecision s -> `Assoc ["type", `String "cerbos_decision"; "detail", `String s]

let json_of_audit_entry (e : AuditLog.audit_entry) =
  `Assoc [
    "sequence", `Int e.ae_sequence;
    "timestamp", `Int e.ae_timestamp;
    "event", json_of_audit_event e.ae_event;
    "agent_id", `String e.ae_agent_id;
    "session_key", `String e.ae_session_key;
    "prev_hash", `String e.ae_prev_hash;
    "request_id", `String e.ae_request_id;
  ]

let json_of_audit_log log =
  `List (List.map json_of_audit_entry log)

let json_of_tool_call (tc : Types.tool_call) =
  `Assoc [
    "id", `String tc.tc_id;
    "type", `String tc.tc_type;
    "name", `String tc.tc_name;
    "function", (match tc.tc_function with
      | Some fc -> `Assoc [
          "name", `String fc.fc_name;
          "arguments", `String fc.fc_arguments;
        ]
      | None -> `Null);
  ]

let json_of_message (m : Types.message) =
  `Assoc [
    "role", `String (Types.role_to_string m.msg_role);
    "content", `String m.msg_content;
    "reasoning_content", `String m.msg_reasoning_content;
    "tool_calls", `List (List.map json_of_tool_call m.msg_tool_calls);
    "tool_call_id", `String m.msg_tool_call_id;
  ]

let json_of_tool_definition (td : Types.tool_definition) =
  `Assoc [
    "name", `String td.td_name;
    "description", `String td.td_description;
    "parameters", `String td.td_parameters;
  ]

(** Build a JSON-RPC 2.0 request *)
let json_rpc_request id method_ params =
  `Assoc [
    "jsonrpc", `String "2.0";
    "id", `Int id;
    "method", `String method_;
    "params", params;
  ]

(** Build a JSON-RPC 2.0 success response *)
let json_rpc_response id result =
  `Assoc [
    "jsonrpc", `String "2.0";
    "id", `Int id;
    "result", result;
  ]

(** Build a JSON-RPC 2.0 error response *)
let json_rpc_error id code message data =
  let err = [
    "code", `Int code;
    "message", `String message;
  ] @ (match data with None -> [] | Some d -> ["data", d])
  in
  `Assoc [
    "jsonrpc", `String "2.0";
    "id", `Int id;
    "error", `Assoc err;
  ]
