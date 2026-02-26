-- Shared ACL rule types used across picoclaw, tailnet-acl, and remote-juggler.
--
-- These fragments follow the tailnet-acl composition pattern: each rule
-- is a self-contained record that can be composed into larger policy sets.

let ACLAction =
      < Accept | Deny | Log >

let ACLRule =
      { src : List Text
      , dst : List Text
      , action : ACLAction
      , description : Text
      }

let CapabilityGrant =
      { principal : Text
      , capabilities : List Text
      , expires : Optional Text   -- ISO 8601 duration or "never"
      }

let PolicySet =
      { name : Text
      , version : Text
      , rules : List ACLRule
      , grants : List CapabilityGrant
      }

let allowRule
    : List Text -> List Text -> Text -> ACLRule
    = \(src : List Text) ->
      \(dst : List Text) ->
      \(desc : Text) ->
        { src, dst, action = ACLAction.Accept, description = desc }

let denyRule
    : List Text -> List Text -> Text -> ACLRule
    = \(src : List Text) ->
      \(dst : List Text) ->
      \(desc : Text) ->
        { src, dst, action = ACLAction.Deny, description = desc }

let grant
    : Text -> List Text -> CapabilityGrant
    = \(principal : Text) ->
      \(caps : List Text) ->
        { principal, capabilities = caps, expires = None Text }

in  { ACLAction
    , ACLRule
    , CapabilityGrant
    , PolicySet
    , allowRule
    , denyRule
    , grant
    }
