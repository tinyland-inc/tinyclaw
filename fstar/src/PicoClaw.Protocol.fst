(**
  PicoClaw.Protocol -- JSON-RPC wire protocol between Go gateway and F* core.

  The F* core communicates with the Go gateway via JSON-RPC over STDIO.
  The gateway sends requests, the core processes them and may issue
  callbacks (llm_call, execute_tool) before returning the final response.

  Methods:
  - process_message: Gateway -> Core (process inbound message)
  - llm_call: Core -> Gateway (request LLM inference)
  - execute_tool: Core -> Gateway (request tool execution with auth proof)
*)
module PicoClaw.Protocol

open FStar.List.Tot
open PicoClaw.Types
open PicoClaw.AuditLog

(** JSON-RPC request ID *)
type request_id = nat

(** Process message request from gateway *)
type process_message_request = {
  pmr_id: request_id;
  pmr_route_input: route_input;
  pmr_content: string;
  pmr_media: list string;
  pmr_bindings: list agent_binding;
  pmr_default_agent: string;
  pmr_dm_scope: dm_scope;
  pmr_tool_definitions: list tool_definition;
  pmr_max_iterations: nat;
  pmr_request_id: string;  (* Aperture correlation ID *)
}

(** LLM call request from core to gateway *)
type llm_call_request = {
  lcr_id: request_id;
  lcr_model: string;
  lcr_messages: list message;
  lcr_tools: list tool_definition;
  lcr_max_tokens: nat;
  lcr_request_id: string;
}

(** LLM call response from gateway *)
type llm_call_response = {
  lcres_id: request_id;
  lcres_response: llm_response;
}

(** Tool execution request from core to gateway *)
type execute_tool_request = {
  etr_id: request_id;
  etr_tool_name: string;
  etr_arguments: string;  (* JSON *)
  etr_agent_id: string;
  etr_grant_proof: string;  (* Serialized grant *)
  etr_request_id: string;
}

(** Tool execution response from gateway *)
type execute_tool_response = {
  etres_id: request_id;
  etres_result: tool_result;
}

(** Process message response from core to gateway *)
type process_message_response = {
  pmres_id: request_id;
  pmres_content: string;
  pmres_agent_id: string;
  pmres_session_key: string;
  pmres_audit_entries: audit_log;
}

(** All possible RPC messages *)
type rpc_message =
  | RPCProcessMessage of process_message_request
  | RPCLLMCall of llm_call_request
  | RPCLLMCallResult of llm_call_response
  | RPCExecuteTool of execute_tool_request
  | RPCExecuteToolResult of execute_tool_response
  | RPCProcessMessageResult of process_message_response
  | RPCError of request_id & string

(** Check if an RPC message is a request (gateway -> core) *)
let is_request (msg: rpc_message) : bool =
  match msg with
  | RPCProcessMessage _ -> true
  | RPCLLMCallResult _ -> true
  | RPCExecuteToolResult _ -> true
  | _ -> false

(** Check if an RPC message is a response (core -> gateway) *)
let is_response (msg: rpc_message) : bool =
  match msg with
  | RPCLLMCall _ -> true
  | RPCExecuteTool _ -> true
  | RPCProcessMessageResult _ -> true
  | RPCError _ -> true
  | _ -> false

(** Extract request ID from any RPC message *)
let get_request_id (msg: rpc_message) : request_id =
  match msg with
  | RPCProcessMessage r -> r.pmr_id
  | RPCLLMCall r -> r.lcr_id
  | RPCLLMCallResult r -> r.lcres_id
  | RPCExecuteTool r -> r.etr_id
  | RPCExecuteToolResult r -> r.etres_id
  | RPCProcessMessageResult r -> r.pmres_id
  | RPCError (id, _) -> id
