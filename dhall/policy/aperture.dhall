-- Aperture access rules for Tailscale-proxied LLM API routing.
--
-- These rules define which agents can access which LLM providers through
-- Aperture, with rate limits and budget constraints. This follows the
-- tailnet-acl fragment composition pattern.

let ApertureRule =
      { agent_id : Text
      , provider : Text
      , model_pattern : Text
      , max_rpm : Natural
      , max_tokens_per_day : Natural
      , allowed : Bool
      }

let defaultRule
    : Text -> Text -> ApertureRule
    = \(agent : Text) ->
      \(provider : Text) ->
        { agent_id = agent
        , provider
        , model_pattern = "*"
        , max_rpm = 60
        , max_tokens_per_day = 1000000
        , allowed = True
        }

let denyRule
    : Text -> Text -> ApertureRule
    = \(agent : Text) ->
      \(provider : Text) ->
        { agent_id = agent
        , provider
        , model_pattern = "*"
        , max_rpm = 0
        , max_tokens_per_day = 0
        , allowed = False
        }

let ApertureConfig =
      { enabled : Bool
      , proxy_url : Text
      , webhook_url : Text
      , rules : List ApertureRule
      }

let defaultConfig
    : ApertureConfig
    = { enabled = False
      , proxy_url = ""
      , webhook_url = ""
      , rules = [] : List ApertureRule
      }

in  { ApertureRule
    , ApertureConfig
    , defaultRule
    , denyRule
    , defaultConfig
    }
