-- Routing policy definitions
-- Defines the cascade order for agent resolution.
--
-- The 7-level routing cascade (verified in fstar/src/PicoClaw.Routing.fst):
--   1. Explicit binding match (channel + account_id + peer)
--   2. Channel + guild/team match
--   3. Channel + account_id match
--   4. Channel-only match
--   5. Agent marked as default
--   6. First agent in list
--   7. Built-in fallback agent
--
-- This file defines the policy; enforcement is in Go (legacy) or F* (verified).

let CascadeLevel =
      < ExplicitBinding
      | ChannelGuild
      | ChannelAccount
      | ChannelOnly
      | DefaultAgent
      | FirstAgent
      | BuiltinFallback
      >

let CascadeRule =
      { level : CascadeLevel
      , description : Text
      , priority : Natural
      }

let cascade
    : List CascadeRule
    = [ { level = CascadeLevel.ExplicitBinding
        , description = "Exact match: channel + account_id + peer"
        , priority = 1
        }
      , { level = CascadeLevel.ChannelGuild
        , description = "Channel + guild_id/team_id match"
        , priority = 2
        }
      , { level = CascadeLevel.ChannelAccount
        , description = "Channel + account_id match"
        , priority = 3
        }
      , { level = CascadeLevel.ChannelOnly
        , description = "Channel type match only"
        , priority = 4
        }
      , { level = CascadeLevel.DefaultAgent
        , description = "Agent with default=true"
        , priority = 5
        }
      , { level = CascadeLevel.FirstAgent
        , description = "First agent in list"
        , priority = 6
        }
      , { level = CascadeLevel.BuiltinFallback
        , description = "Built-in fallback with default model"
        , priority = 7
        }
      ]

in  { CascadeLevel, CascadeRule, cascade }
