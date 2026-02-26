-- Shared Aperture policy types for the Tinyland ecosystem.
--
-- Used by picoclaw, remote-juggler, and tailnet-acl to define
-- consistent Aperture access rules.

let ACL = ./acl.dhall

let ApertureEndpoint =
      { name : Text
      , upstream_url : Text
      , rate_limit_rpm : Natural
      , token_budget_daily : Natural
      }

let AperturePolicy =
      { endpoints : List ApertureEndpoint
      , access_rules : List ACL.ACLRule
      , metering_enabled : Bool
      , webhook_url : Text
      }

let defaultEndpoint
    : Text -> Text -> ApertureEndpoint
    = \(name : Text) ->
      \(url : Text) ->
        { name
        , upstream_url = url
        , rate_limit_rpm = 60
        , token_budget_daily = 1000000
        }

let defaultPolicy
    : AperturePolicy
    = { endpoints = [] : List ApertureEndpoint
      , access_rules = [] : List ACL.ACLRule
      , metering_enabled = True
      , webhook_url = ""
      }

in  { ApertureEndpoint
    , AperturePolicy
    , defaultEndpoint
    , defaultPolicy
    }
