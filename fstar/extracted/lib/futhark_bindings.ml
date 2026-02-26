(** OCaml FFI bindings to Futhark C backend.

    These bindings wrap the Futhark-generated C library for use
    from the F*-extracted OCaml core. The C library is produced by
    `futhark c` and linked via Ctypes.

    For now, this module provides pure OCaml fallback implementations
    that match the Futhark kernel semantics. The Futhark C backend
    will be linked in when available. *)

(** Cosine similarity between two float arrays. *)
let cosine_similarity (a : float array) (b : float array) : float =
  let n = min (Array.length a) (Array.length b) in
  if n = 0 then 0.0
  else
    let dot = ref 0.0 in
    let na = ref 0.0 in
    let nb = ref 0.0 in
    for i = 0 to n - 1 do
      dot := !dot +. a.(i) *. b.(i);
      na := !na +. a.(i) *. a.(i);
      nb := !nb +. b.(i) *. b.(i)
    done;
    let na_sqrt = sqrt !na in
    let nb_sqrt = sqrt !nb in
    if na_sqrt = 0.0 || nb_sqrt = 0.0 then 0.0
    else !dot /. (na_sqrt *. nb_sqrt)

(** Batch cosine similarity: query vs each candidate row. *)
let batch_cosine_similarity (query : float array) (candidates : float array array) : float array =
  Array.map (cosine_similarity query) candidates

(** Top-k most similar candidates. Returns indices sorted by descending similarity. *)
let top_k_similar (query : float array) (candidates : float array array) (k : int) : int array =
  let scores = batch_cosine_similarity query candidates in
  let indexed = Array.mapi (fun i s -> (s, i)) scores in
  Array.sort (fun (a, _) (b, _) -> compare b a) indexed;
  let k' = min k (Array.length indexed) in
  Array.init k' (fun i -> snd indexed.(i))

(** Estimate token count for a string using the 2.5 chars/token heuristic. *)
let estimate_tokens (text : string) : int =
  let n = String.length text in
  (* Simple heuristic matching Go's totalChars * 2 / 5 *)
  n * 2 / 5

(** Batch token estimation for multiple strings. *)
let batch_estimate_tokens (texts : string array) : int array =
  Array.map estimate_tokens texts

(** Check if total tokens exceed a threshold percentage of max. *)
let exceeds_context_threshold (texts : string array) (max_tokens : int) (threshold_pct : int) : bool =
  let total = Array.fold_left (fun acc t -> acc + estimate_tokens t) 0 texts in
  total * 100 > max_tokens * threshold_pct
