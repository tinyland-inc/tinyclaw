(** PicoClaw Verified Core -- STDIO JSON-RPC entry point.

    This binary is spawned by the Go gateway as a subprocess.
    Communication is via JSON-RPC 2.0 over STDIO (stdin/stdout).

    The core processes messages from the gateway, making callbacks
    for LLM inference and tool execution as needed, then returns
    the final response with an audit trail.

    Protocol:
    - Gateway -> Core: process_message (route + content + bindings)
    - Core -> Gateway: llm_call (request LLM inference)
    - Gateway -> Core: llm_call result (LLM response)
    - Core -> Gateway: execute_tool (request tool execution)
    - Gateway -> Core: execute_tool result (tool output)
    - Core -> Gateway: process_message result (final response + audit log) *)

open Picoclaw_lib
open Picoclaw_core
open AgentLoop

(** ─── Session Store ─────────────────────────────────────────────── *)

(** In-memory session cache: session_key -> session *)
let sessions : (string, Session.session) Hashtbl.t = Hashtbl.create 64

let get_or_create_session key =
  match Hashtbl.find_opt sessions key with
  | Some s -> s
  | None ->
    let s = Session.empty_session key in
    Hashtbl.replace sessions key s;
    s

let save_session (s : Session.session) =
  Hashtbl.replace sessions s.s_key s

(** ─── I/O Primitives ────────────────────────────────────────────── *)

let ic = stdin
let oc = stdout

(** Read a JSON-RPC message from stdin (Content-Length framing) *)
let read_message () : Yojson.Basic.t =
  let header = input_line ic in
  let content_length =
    if String.length header > 16 then
      int_of_string (String.trim (String.sub header 16 (String.length header - 16)))
    else
      failwith "Invalid Content-Length header"
  in
  (* Skip blank line separator *)
  let _ = input_line ic in
  let buf = Bytes.create content_length in
  really_input ic buf 0 content_length;
  Yojson.Basic.from_string (Bytes.to_string buf)

(** Write a JSON-RPC message to stdout (Content-Length framing) *)
let write_message (json : Yojson.Basic.t) : unit =
  let content = Yojson.Basic.to_string json in
  let len = String.length content in
  Printf.fprintf oc "Content-Length: %d\r\n\r\n%s" len content;
  flush oc

(** ─── Callback Management ───────────────────────────────────────── *)

(** Global callback ID counter *)
let next_callback_id = ref 1

let fresh_id () =
  let id = !next_callback_id in
  incr next_callback_id;
  id

(** Send an LLM call request to the gateway and wait for response *)
let call_llm messages tools agent_id request_id : Types.llm_response =
  let id = fresh_id () in
  let params = `Assoc [
    "messages", `List (List.map Json_codec.json_of_message messages);
    "tools", `List (List.map Json_codec.json_of_tool_definition tools);
    "agent_id", `String agent_id;
    "request_id", `String request_id;
  ] in
  write_message (Json_codec.json_rpc_request id "llm_call" params);
  (* Read response from gateway *)
  let resp = read_message () in
  let result = Json_codec.assoc_field "result" resp in
  Json_codec.llm_response_of_json result

(** Send a tool execution request to the gateway and wait for response *)
let execute_tool (tc : Types.tool_call) agent_id grant_proof request_id : Types.tool_result =
  let id = fresh_id () in
  let arguments = match tc.tc_function with
    | Some fc -> fc.fc_arguments
    | None -> "{}"
  in
  let params = `Assoc [
    "tool_name", `String tc.tc_name;
    "arguments", `String arguments;
    "agent_id", `String agent_id;
    "grant_proof", `String grant_proof;
    "request_id", `String request_id;
  ] in
  write_message (Json_codec.json_rpc_request id "execute_tool" params);
  let resp = read_message () in
  let result = Json_codec.assoc_field "result" resp in
  Json_codec.tool_result_of_json result

(** ─── Message Processing ────────────────────────────────────────── *)

(** Get current timestamp as int *)
let now () = int_of_float (Unix.gettimeofday ())

(** Process a single message through the verified agent loop *)
let process_message (params : Json_codec.process_message_params) : Yojson.Basic.t =
  (* Step 1: Route the message *)
  let route = Routing.resolve_route
    params.pm_bindings
    params.pm_default_agent
    params.pm_dm_scope
    params.pm_route_input
  in

  (* Step 2: Get or create session *)
  let sess = get_or_create_session route.rr_session_key in

  (* Step 3: Initialize the agent loop *)
  let state = AgentLoop.init_loop route sess params.pm_content
    params.pm_request_id (now ()) in

  (* Step 4: Run the iterative loop *)
  let rec loop state fuel =
    if fuel <= 0 then
      (* Fuel exhausted: return what we have *)
      let log = AuditLog.append_entry state.ls_audit_log (now ())
        (AuditLog.MessageProcessed "fuel_exhausted")
        state.ls_agent_id state.ls_session.Session.s_key state.ls_request_id
      in
      ("I've reached the maximum number of iterations.", state.ls_agent_id,
       state.ls_session.Session.s_key, log)
    else
      (* Call LLM via gateway callback *)
      let response = call_llm state.ls_messages params.pm_tool_definitions
        state.ls_agent_id state.ls_request_id in
      let (outcome, log) = AgentLoop.step_iteration state response fuel (now ()) in
      match outcome with
      | AgentLoop.FinalResponse (content, final_state) ->
        let log = AuditLog.append_entry log (now ())
          (AuditLog.MessageProcessed "final_response")
          final_state.ls_agent_id final_state.ls_session.Session.s_key
          final_state.ls_request_id
        in
        save_session final_state.ls_session;
        (content, final_state.ls_agent_id,
         final_state.ls_session.Session.s_key, log)

      | AgentLoop.NeedsToolCalls (tool_calls, tool_state) ->
        (* Authorize tool calls *)
        let decisions = AgentLoop.authorize_tool_calls tool_calls
          tool_state.ls_agent_id [] [] (now ()) in
        let authorized = AgentLoop.filter_authorized decisions in
        let denied = AgentLoop.filter_denied decisions in
        (* Log denied calls *)
        let log_with_denied = List.fold_left (fun acc (tc, _reason) ->
          AuditLog.log_tool_auth acc (now ()) tc.Types.tc_name false
            tool_state.ls_agent_id tool_state.ls_session.Session.s_key
            tool_state.ls_request_id
        ) log denied in
        (* Execute authorized tools *)
        let (log_final, tool_messages) = List.fold_left (fun (log_acc, msgs) tc ->
          let log_auth = AuditLog.log_tool_auth log_acc (now ()) tc.Types.tc_name true
            tool_state.ls_agent_id tool_state.ls_session.Session.s_key
            tool_state.ls_request_id in
          let result = execute_tool tc tool_state.ls_agent_id "" tool_state.ls_request_id in
          let log_exec = AuditLog.log_tool_exec log_auth (now ()) tc.Types.tc_name
            tool_state.ls_agent_id tool_state.ls_session.Session.s_key
            tool_state.ls_request_id in
          let tool_msg = {
            Types.msg_role = Types.Tool;
            msg_content = result.Types.tr_for_llm;
            msg_reasoning_content = "";
            msg_tool_calls = [];
            msg_tool_call_id = tc.Types.tc_id;
          } in
          (log_exec, msgs @ [tool_msg])
        ) (log_with_denied, []) authorized in
        (* Add tool results to messages and continue *)
        let updated_state = {
          tool_state with
          ls_messages = tool_state.ls_messages @ tool_messages;
          ls_session = List.fold_left Session.add_message
            tool_state.ls_session tool_messages;
          ls_audit_log = log_final;
        } in
        loop updated_state (fuel - 1)

      | AgentLoop.FuelExhausted final_state ->
        let log = AuditLog.append_entry log (now ())
          (AuditLog.MessageProcessed "fuel_exhausted")
          final_state.ls_agent_id final_state.ls_session.Session.s_key
          final_state.ls_request_id
        in
        save_session final_state.ls_session;
        ("I've reached the maximum number of iterations.", final_state.ls_agent_id,
         final_state.ls_session.Session.s_key, log)

      | AgentLoop.IterationError (msg, final_state) ->
        let log = AuditLog.append_entry log (now ())
          (AuditLog.MessageProcessed ("error:" ^ msg))
          final_state.ls_agent_id final_state.ls_session.Session.s_key
          final_state.ls_request_id
        in
        save_session final_state.ls_session;
        (msg, final_state.ls_agent_id,
         final_state.ls_session.Session.s_key, log)
  in
  let max_iter = if params.pm_max_iterations > 0 then params.pm_max_iterations else 10 in
  let (content, agent_id, session_key, audit_log) = loop state max_iter in
  `Assoc [
    "content", `String content;
    "agent_id", `String agent_id;
    "session_key", `String session_key;
    "audit_log", Json_codec.json_of_audit_log audit_log;
  ]

(** ─── JSON-RPC Dispatch ─────────────────────────────────────────── *)

let handle_request (json : Yojson.Basic.t) : unit =
  let id = Json_codec.int_of_json (Json_codec.assoc_field "id" json) in
  let method_ = Json_codec.string_of_json (Json_codec.assoc_field "method" json) in
  let params = Json_codec.assoc_field "params" json in
  match method_ with
  | "process_message" ->
    (try
      let pm_params = Json_codec.process_message_params_of_json params in
      let result = process_message pm_params in
      write_message (Json_codec.json_rpc_response id result)
    with exn ->
      write_message (Json_codec.json_rpc_error id (-32603)
        ("Internal error: " ^ Printexc.to_string exn) None))
  | "ping" ->
    write_message (Json_codec.json_rpc_response id (`Assoc [
      "status", `String "ok";
      "version", `String "0.8.0-verified";
    ]))
  | _ ->
    write_message (Json_codec.json_rpc_error id (-32601)
      ("Method not found: " ^ method_) None)

(** ─── Main Loop ─────────────────────────────────────────────────── *)

let () =
  set_binary_mode_out oc true;
  try
    while true do
      let msg = read_message () in
      handle_request msg
    done
  with
  | End_of_file -> () (* Gateway closed connection *)
  | exn ->
    Printf.eprintf "picoclaw-core: fatal error: %s\n" (Printexc.to_string exn);
    exit 1
