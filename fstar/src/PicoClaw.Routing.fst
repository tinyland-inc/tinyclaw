(**
  PicoClaw.Routing -- 7-level routing cascade, proven total and deterministic.

  The routing cascade resolves an inbound message to a specific agent and
  session key. The cascade is total (always produces a result) and
  deterministic (same input always produces the same output).
*)
module PicoClaw.Routing

open FStar.List.Tot
open PicoClaw.Types

(** Default agent ID used when no binding matches *)
let default_agent_id : string = "main"

(** Default account ID for normalization *)
let default_account_id : string = "default"

(** Build main session key for an agent *)
let build_main_session_key (agent_id: string) : string =
  String.concat ":" ["agent"; agent_id; "main"]

(** Build peer session key based on DM scope *)
let build_peer_session_key
  (scope: dm_scope)
  (agent_id: string)
  (channel: string)
  (account_id: string)
  (peer: route_peer)
  : string =
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

(** Build group session key *)
let build_group_session_key
  (agent_id: string)
  (channel: string)
  (group_id: string)
  : string =
  String.concat ":" ["agent"; agent_id; channel; "group"; group_id]

(** Check if a binding matches by peer *)
let matches_peer (b: agent_binding) (input: route_input) : bool =
  match b.ab_match.bm_peer, input.ri_peer with
  | Some bp, Some ip ->
    bp.rp_kind = ip.rp_kind && bp.rp_id = ip.rp_id &&
    b.ab_match.bm_channel = input.ri_channel
  | _, _ -> false

(** Check if a binding matches by parent peer *)
let matches_parent_peer (b: agent_binding) (input: route_input) : bool =
  match b.ab_match.bm_peer, input.ri_parent_peer with
  | Some bp, Some pp ->
    bp.rp_kind = pp.rp_kind && bp.rp_id = pp.rp_id &&
    b.ab_match.bm_channel = input.ri_channel
  | _, _ -> false

(** Check if a binding matches by guild *)
let matches_guild (b: agent_binding) (input: route_input) : bool =
  b.ab_match.bm_guild_id <> "" &&
  b.ab_match.bm_guild_id = input.ri_guild_id &&
  b.ab_match.bm_channel = input.ri_channel

(** Check if a binding matches by team *)
let matches_team (b: agent_binding) (input: route_input) : bool =
  b.ab_match.bm_team_id <> "" &&
  b.ab_match.bm_team_id = input.ri_team_id &&
  b.ab_match.bm_channel = input.ri_channel

(** Check if a binding matches by account *)
let matches_account (b: agent_binding) (input: route_input) : bool =
  b.ab_match.bm_account_id <> "" &&
  b.ab_match.bm_account_id <> "*" &&
  b.ab_match.bm_account_id = input.ri_account_id &&
  b.ab_match.bm_channel = input.ri_channel &&
  (match b.ab_match.bm_peer with None -> true | _ -> false)

(** Check if a binding matches by channel wildcard *)
let matches_channel_wildcard (b: agent_binding) (input: route_input) : bool =
  b.ab_match.bm_channel = input.ri_channel &&
  (b.ab_match.bm_account_id = "*" || b.ab_match.bm_account_id = "") &&
  (match b.ab_match.bm_peer with None -> true | _ -> false) &&
  b.ab_match.bm_guild_id = "" &&
  b.ab_match.bm_team_id = ""

(** Find first binding matching a predicate *)
let find_binding (pred: agent_binding -> bool) (bindings: list agent_binding)
  : option agent_binding =
  find pred bindings

(**
  Resolve route through the 7-level cascade.

  This function is:
  - Total: always produces a resolved_route (level 7 is unconditional default)
  - Deterministic: same inputs always produce the same output
  - Pure: no side effects

  Cascade levels:
    1. Peer binding (direct match on sender peer)
    2. Parent peer binding (match on reply/thread parent)
    3. Guild binding (Slack workspace, Discord server)
    4. Team binding (team/group context)
    5. Account binding (account-level, no peer qualifier)
    6. Channel wildcard (any account on channel)
    7. Default agent (unconditional fallback)
*)
let resolve_route
  (bindings: list agent_binding)
  (default_id: string)
  (scope: dm_scope)
  (input: route_input)
  : Tot resolved_route =
  let make_route (agent_id: string) (reason: match_reason) : resolved_route =
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
    {
      rr_agent_id = agent_id;
      rr_channel = input.ri_channel;
      rr_account_id = input.ri_account_id;
      rr_session_key = session_key;
      rr_main_session_key = build_main_session_key agent_id;
      rr_matched_by = reason;
    }
  in
  (* Level 1: Peer binding *)
  match find_binding (fun b -> matches_peer b input) bindings with
  | Some b -> make_route b.ab_agent_id MatchedByPeer
  | None ->
  (* Level 2: Parent peer binding *)
  match find_binding (fun b -> matches_parent_peer b input) bindings with
  | Some b -> make_route b.ab_agent_id MatchedByParentPeer
  | None ->
  (* Level 3: Guild binding *)
  match find_binding (fun b -> matches_guild b input) bindings with
  | Some b -> make_route b.ab_agent_id MatchedByGuild
  | None ->
  (* Level 4: Team binding *)
  match find_binding (fun b -> matches_team b input) bindings with
  | Some b -> make_route b.ab_agent_id MatchedByTeam
  | None ->
  (* Level 5: Account binding *)
  match find_binding (fun b -> matches_account b input) bindings with
  | Some b -> make_route b.ab_agent_id MatchedByAccount
  | None ->
  (* Level 6: Channel wildcard *)
  match find_binding (fun b -> matches_channel_wildcard b input) bindings with
  | Some b -> make_route b.ab_agent_id MatchedByChannelWildcard
  | None ->
  (* Level 7: Default (always succeeds, making this function total) *)
  make_route default_id MatchedByDefault

(** Proof: resolve_route is total -- F* verifies this automatically since
    the return type is Tot resolved_route and all branches terminate. *)

(**
  Determinism lemma: resolve_route produces identical output for identical input.
  This is trivially true since resolve_route is a pure function with no
  mutable state, but we state it explicitly for documentation.
*)
let resolve_route_deterministic
  (bindings: list agent_binding)
  (default_id: string)
  (scope: dm_scope)
  (input: route_input)
  : Lemma (resolve_route bindings default_id scope input ==
           resolve_route bindings default_id scope input) = ()
