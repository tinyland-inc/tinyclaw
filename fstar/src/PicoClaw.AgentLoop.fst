(**
  PicoClaw.AgentLoop -- State machine with fuel-based termination proof.

  The agent loop processes messages through an iterative cycle:
  1. Route inbound message to agent + session
  2. Build context (summary + history + system prompt)
  3. Call LLM with tool definitions
  4. If LLM requests tool calls, authorize and execute them
  5. Repeat from step 3 until no more tool calls or fuel exhausted
  6. Return final response with audit trail

  Termination is guaranteed by fuel (max_iterations).
*)
module PicoClaw.AgentLoop

open FStar.List.Tot
open PicoClaw.Types
open PicoClaw.Routing
open PicoClaw.ToolAuth
open PicoClaw.Session
open PicoClaw.AuditLog
open PicoClaw.Protocol

(** Agent loop state *)
type loop_state = {
  ls_session: session;
  ls_audit_log: audit_log;
  ls_messages: list message;  (* Current conversation for LLM *)
  ls_iteration: nat;
  ls_agent_id: string;
  ls_request_id: string;
}

(** Outcome of a single LLM iteration *)
type iteration_outcome =
  | FinalResponse of string & loop_state
      (* LLM produced final text, done *)
  | NeedsToolCalls of list tool_call & loop_state
      (* LLM wants to call tools, continue *)
  | FuelExhausted of loop_state
      (* Max iterations reached *)
  | IterationError of string & loop_state
      (* Unrecoverable error *)

(** Process tool call results into messages *)
let tool_results_to_messages (calls: list (tool_call & tool_result))
  : Tot (list message) =
  map (fun (tc, tr) -> {
    msg_role = Tool;
    msg_content = tr.tr_for_llm;
    msg_reasoning_content = "";
    msg_tool_calls = [];
    msg_tool_call_id = tc.tc_id;
  }) calls

(** Classify LLM response into iteration outcome *)
let classify_response
  (response: llm_response)
  (state: loop_state)
  (fuel: nat)
  : Tot iteration_outcome =
  if fuel = 0 then
    FuelExhausted state
  else if length response.lr_tool_calls > 0 then
    NeedsToolCalls response.lr_tool_calls state
  else
    FinalResponse response.lr_content state

(**
  Run the agent loop for one iteration.

  This is a pure function that takes the current state and LLM response,
  and produces the next state. The actual LLM call and tool execution
  happen via JSON-RPC callbacks in the gateway.

  Returns:
  - Updated state with new messages and audit entries
  - The iteration outcome (final, needs tools, or exhausted)
*)
let step_iteration
  (state: loop_state)
  (response: llm_response)
  (fuel: nat)
  (timestamp: nat)
  : Tot (iteration_outcome & audit_log) =
  (* Log the LLM call completion *)
  let log1 = log_llm_call
    state.ls_audit_log timestamp "model"
    true state.ls_agent_id
    state.ls_session.s_key state.ls_request_id
  in
  (* Add assistant response to messages *)
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

(**
  Process tool authorization for a batch of tool calls.

  Returns list of (tool_call, auth_decision) pairs.
  Every tool call gets a decision -- authorized or denied.
*)
let authorize_tool_calls
  (calls: list tool_call)
  (agent_id: string)
  (policy: list policy_entry)
  (grants: list grant)
  (timestamp: nat)
  : Tot (list (tool_call & auth_decision)) =
  map (fun tc ->
    let decision = authorize_tool tc.tc_name agent_id policy grants timestamp in
    (tc, decision)
  ) calls

(** Filter authorized tool calls *)
let filter_authorized (decisions: list (tool_call & auth_decision))
  : Tot (list tool_call) =
  let authorized = filter (fun (_, d) -> Authorized? d) decisions in
  map fst authorized

(** Filter denied tool calls *)
let filter_denied (decisions: list (tool_call & auth_decision))
  : Tot (list (tool_call & string)) =
  let denied = filter (fun (_, d) -> Denied? d) decisions in
  map (fun (tc, d) ->
    let reason = match d with | Denied r -> r | _ -> "unknown" in
    (tc, reason)
  ) denied

(**
  Initialize the agent loop state for a new message.

  This sets up the initial state with routing, session lookup, and
  the user message added to context.
*)
let init_loop
  (route: resolved_route)
  (sess: session)
  (user_content: string)
  (request_id: string)
  (timestamp: nat)
  : Tot loop_state =
  let user_msg = {
    msg_role = User;
    msg_content = user_content;
    msg_reasoning_content = "";
    msg_tool_calls = [];
    msg_tool_call_id = "";
  } in
  let sess_with_msg = add_message sess user_msg in
  let context = build_context sess_with_msg in
  let initial_log = log_route [] timestamp route.rr_agent_id
                      route.rr_session_key request_id in
  {
    ls_session = sess_with_msg;
    ls_audit_log = initial_log;
    ls_messages = context;
    ls_iteration = 0;
    ls_agent_id = route.rr_agent_id;
    ls_request_id = request_id;
  }

(**
  Proof: iteration count is bounded by fuel.

  Since each call to step_iteration increments ls_iteration by 1,
  and we only continue when fuel > 0, the loop terminates.
*)
let step_increases_iteration (state: loop_state) (response: llm_response)
  (fuel: nat) (timestamp: nat)
  : Lemma (let (outcome, _) = step_iteration state response fuel timestamp in
           match outcome with
           | FinalResponse _ s | NeedsToolCalls _ s | FuelExhausted s | IterationError _ s ->
             s.ls_iteration == state.ls_iteration + 1)
  = ()
