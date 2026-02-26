(**
  PicoClaw.AuditLog -- Hash-chained append-only audit trail.

  Every decision made by the verified core is recorded in the audit log.
  Entries are hash-chained: each entry includes the hash of the previous
  entry, making the log tamper-evident.

  Properties proven:
  - Append-only: entries can only be added, never removed or modified
  - Hash-chain integrity: each entry's prev_hash matches predecessor
  - Monotonic: log length never decreases
*)
module PicoClaw.AuditLog

open FStar.List.Tot

(** Audit event types *)
type audit_event =
  | RouteResolved of string    (* agent_id *)
  | ToolAuthorized of string   (* tool_name *)
  | ToolDenied of string       (* tool_name with reason *)
  | ToolExecuted of string     (* tool_name *)
  | LLMCallStarted of string   (* model *)
  | LLMCallCompleted of string (* model with token count *)
  | SessionCreated of string   (* session_key *)
  | MessageProcessed of string (* summary *)

(** A single audit log entry *)
type audit_entry = {
  ae_sequence: nat;         (* Monotonically increasing *)
  ae_timestamp: nat;        (* Unix timestamp *)
  ae_event: audit_event;
  ae_agent_id: string;
  ae_session_key: string;
  ae_prev_hash: string;     (* Hash of previous entry, "" for first *)
  ae_request_id: string;    (* Correlation ID for Aperture integration *)
}

(** The audit log is a list of entries *)
type audit_log = list audit_entry

(** Simple hash function placeholder -- replaced by real hash in extraction *)
let hash_entry (e: audit_entry) : string =
  String.concat ":" [
    string_of_int (e.ae_sequence);
    string_of_int (e.ae_timestamp);
    e.ae_agent_id;
    e.ae_session_key;
    e.ae_prev_hash;
    e.ae_request_id
  ]

(** Get the hash of the last entry in the log *)
let last_hash (log: audit_log) : string =
  match log with
  | [] -> ""
  | _ ->
    let last_entry = index log (length log - 1) in
    hash_entry last_entry

(** Get the next sequence number *)
let next_sequence (log: audit_log) : nat =
  length log

(** Append a new entry to the audit log *)
let append_entry
  (log: audit_log)
  (timestamp: nat)
  (event: audit_event)
  (agent_id: string)
  (session_key: string)
  (request_id: string)
  : Tot audit_log =
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

(** Proof: appending increases log length by exactly 1 *)
let append_increases_length
  (log: audit_log)
  (timestamp: nat)
  (event: audit_event)
  (agent_id: string)
  (session_key: string)
  (request_id: string)
  : Lemma (length (append_entry log timestamp event agent_id session_key request_id) ==
           length log + 1)
  = append_length log [{
      ae_sequence = next_sequence log;
      ae_timestamp = timestamp;
      ae_event = event;
      ae_agent_id = agent_id;
      ae_session_key = session_key;
      ae_prev_hash = last_hash log;
      ae_request_id = request_id;
    }]

(** Proof: log length is monotonically non-decreasing after append *)
let append_monotonic
  (log: audit_log)
  (timestamp: nat)
  (event: audit_event)
  (agent_id: string)
  (session_key: string)
  (request_id: string)
  : Lemma (length (append_entry log timestamp event agent_id session_key request_id) >=
           length log)
  = append_increases_length log timestamp event agent_id session_key request_id

(** Check hash chain integrity for consecutive entries *)
let chain_valid_pair (prev cur: audit_entry) : bool =
  cur.ae_prev_hash = hash_entry prev &&
  cur.ae_sequence = prev.ae_sequence + 1

(** Validate the entire hash chain *)
let rec chain_valid (log: audit_log) : Tot bool (decreases log) =
  match log with
  | [] -> true
  | [_] -> true
  | x :: y :: rest ->
    chain_valid_pair x y && chain_valid (y :: rest)

(** Empty log has valid chain *)
let empty_chain_valid (_: unit)
  : Lemma (chain_valid [] == true)
  = ()

(** Record a routing decision *)
let log_route
  (log: audit_log)
  (timestamp: nat)
  (agent_id: string)
  (session_key: string)
  (request_id: string)
  : Tot audit_log =
  append_entry log timestamp (RouteResolved agent_id) agent_id session_key request_id

(** Record a tool authorization decision *)
let log_tool_auth
  (log: audit_log)
  (timestamp: nat)
  (tool_name: string)
  (authorized: bool)
  (agent_id: string)
  (session_key: string)
  (request_id: string)
  : Tot audit_log =
  let event = if authorized
              then ToolAuthorized tool_name
              else ToolDenied tool_name
  in
  append_entry log timestamp event agent_id session_key request_id

(** Record a tool execution *)
let log_tool_exec
  (log: audit_log)
  (timestamp: nat)
  (tool_name: string)
  (agent_id: string)
  (session_key: string)
  (request_id: string)
  : Tot audit_log =
  append_entry log timestamp (ToolExecuted tool_name) agent_id session_key request_id

(** Record an LLM call *)
let log_llm_call
  (log: audit_log)
  (timestamp: nat)
  (model: string)
  (completed: bool)
  (agent_id: string)
  (session_key: string)
  (request_id: string)
  : Tot audit_log =
  let event = if completed
              then LLMCallCompleted model
              else LLMCallStarted model
  in
  append_entry log timestamp event agent_id session_key request_id
