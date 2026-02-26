(**
  PicoClaw.Session -- Append-only session history with monotonic summarization.

  Session history is append-only: messages are added but never removed
  from the logical history. Summarization compresses older messages into
  a summary but the summary monotonically grows (never loses information).
*)
module PicoClaw.Session

open FStar.List.Tot
open PicoClaw.Types

(** Session state *)
type session = {
  s_key: string;
  s_messages: list message;
  s_summary: string;
  s_message_count: nat;  (* Total messages ever added, including summarized *)
}

(** Create a new empty session *)
let empty_session (key: string) : session = {
  s_key = key;
  s_messages = [];
  s_summary = "";
  s_message_count = 0;
}

(** Add a message to the session history *)
let add_message (s: session) (msg: message) : Tot session = {
  s_key = s.s_key;
  s_messages = s.s_messages @ [msg];
  s_summary = s.s_summary;
  s_message_count = s.s_message_count + 1;
}

(** Proof: add_message increases message list length by 1 *)
let add_message_length (s: session) (msg: message)
  : Lemma (length (add_message s msg).s_messages == length s.s_messages + 1)
  = append_length s.s_messages [msg]

(** Proof: add_message increases total count *)
let add_message_count (s: session) (msg: message)
  : Lemma ((add_message s msg).s_message_count == s.s_message_count + 1)
  = ()

(** Get current history *)
let get_history (s: session) : list message = s.s_messages

(** Get current summary *)
let get_summary (s: session) : string = s.s_summary

(** Get total message count (including summarized) *)
let get_total_count (s: session) : nat = s.s_message_count

(**
  Summarize: replace older messages with a summary, keeping recent ones.

  This operation:
  - Replaces messages with a new summary + recent messages
  - Total count never decreases (monotonic)
  - Summary string can only grow (new summary appended to old)
*)
let summarize
  (s: session)
  (new_summary: string)
  (keep_last: nat)
  : Tot session =
  let msgs = s.s_messages in
  let kept =
    if keep_last >= length msgs then msgs
    else
      let start = length msgs - keep_last in
      (* Keep last keep_last messages *)
      let rec take_from (l: list message) (n: nat) : Tot (list message) (decreases l) =
        match l with
        | [] -> []
        | x :: rest ->
          if n > 0 then take_from rest (n - 1)
          else l
      in
      take_from msgs start
  in
  let combined_summary =
    if s.s_summary = "" then new_summary
    else String.concat "\n\n" [s.s_summary; new_summary]
  in
  {
    s_key = s.s_key;
    s_messages = kept;
    s_summary = combined_summary;
    s_message_count = s.s_message_count;  (* Never decreases *)
  }

(** Proof: summarization preserves total count (monotonic) *)
let summarize_preserves_count
  (s: session)
  (new_summary: string)
  (keep_last: nat)
  : Lemma ((summarize s new_summary keep_last).s_message_count == s.s_message_count)
  = ()

(** Check if session needs summarization based on thresholds *)
let needs_summarization
  (s: session)
  (max_messages: nat)
  : Tot bool =
  length s.s_messages > max_messages

(** Build session context for LLM: summary + current messages *)
let build_context (s: session) : Tot (list message) =
  if s.s_summary = "" then
    s.s_messages
  else
    let summary_msg = {
      msg_role = System;
      msg_content = String.concat "" ["Previous conversation summary:\n"; s.s_summary];
      msg_reasoning_content = "";
      msg_tool_calls = [];
      msg_tool_call_id = "";
    } in
    summary_msg :: s.s_messages
