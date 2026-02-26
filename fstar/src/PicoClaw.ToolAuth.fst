(**
  PicoClaw.ToolAuth -- Effect-based tool authorization.

  Tools require grants to execute. This module defines the authorization
  model: each tool call must carry a proof that the caller has permission.
  The proof is a grant token checked against the policy.
*)
module PicoClaw.ToolAuth

open FStar.List.Tot
open PicoClaw.Types

(** Authorization policy level *)
type auth_level =
  | AlwaysAllowed   (* No grant needed *)
  | RequiresGrant   (* Must have explicit grant *)
  | AlwaysDenied    (* Never executable *)

(** A policy entry mapping tool name to authorization level *)
type policy_entry = {
  pe_tool_name: string;
  pe_level: auth_level;
}

(** A grant token proving authorization for a specific tool *)
type grant = {
  g_tool_name: string;
  g_agent_id: string;
  g_issued_at: nat;  (* Timestamp *)
}

(** Authorization decision *)
type auth_decision =
  | Authorized of grant     (* Tool may execute, with proof *)
  | Denied of string        (* Tool may not execute, with reason *)

(** Look up the policy for a tool *)
let lookup_policy (tool_name: string) (policy: list policy_entry)
  : auth_level =
  match find (fun e -> e.pe_tool_name = tool_name) policy with
  | Some entry -> entry.pe_level
  | None -> RequiresGrant  (* Default: require grant for unknown tools *)

(** Check if a grant exists for the given tool and agent *)
let has_grant (tool_name: string) (agent_id: string) (grants: list grant)
  : option grant =
  find (fun g -> g.g_tool_name = tool_name && g.g_agent_id = agent_id) grants

(**
  Authorize a tool call.

  Returns Authorized with the grant proof if allowed, or Denied with reason.
  This function is total and deterministic.
*)
let authorize_tool
  (tool_name: string)
  (agent_id: string)
  (policy: list policy_entry)
  (grants: list grant)
  (timestamp: nat)
  : Tot auth_decision =
  match lookup_policy tool_name policy with
  | AlwaysDenied ->
    Denied (String.concat "" ["tool '"; tool_name; "' is always denied"])
  | AlwaysAllowed ->
    Authorized { g_tool_name = tool_name; g_agent_id = agent_id; g_issued_at = timestamp }
  | RequiresGrant ->
    match has_grant tool_name agent_id grants with
    | Some g -> Authorized g
    | None ->
      Denied (String.concat "" ["no grant for tool '"; tool_name;
                                 "' agent '"; agent_id; "'"])

(** Proof: AlwaysDenied tools can never be authorized *)
let always_denied_never_authorized
  (tool_name: string)
  (agent_id: string)
  (policy: list policy_entry)
  (grants: list grant)
  (timestamp: nat)
  : Lemma
    (requires lookup_policy tool_name policy == AlwaysDenied)
    (ensures Denied? (authorize_tool tool_name agent_id policy grants timestamp))
  = ()

(** Proof: AlwaysAllowed tools are always authorized *)
let always_allowed_always_authorized
  (tool_name: string)
  (agent_id: string)
  (policy: list policy_entry)
  (grants: list grant)
  (timestamp: nat)
  : Lemma
    (requires lookup_policy tool_name policy == AlwaysAllowed)
    (ensures Authorized? (authorize_tool tool_name agent_id policy grants timestamp))
  = ()

(**
  Batch authorization: authorize multiple tool calls.
  Returns list of decisions in the same order as input.
*)
let authorize_tools
  (tool_names: list string)
  (agent_id: string)
  (policy: list policy_entry)
  (grants: list grant)
  (timestamp: nat)
  : Tot (list auth_decision) =
  map (fun name -> authorize_tool name agent_id policy grants timestamp) tool_names

(** Proof: batch authorization preserves length *)
let authorize_tools_length
  (tool_names: list string)
  (agent_id: string)
  (policy: list policy_entry)
  (grants: list grant)
  (timestamp: nat)
  : Lemma (length (authorize_tools tool_names agent_id policy grants timestamp) ==
           length tool_names)
  = map_length (fun name -> authorize_tool name agent_id policy grants timestamp) tool_names
