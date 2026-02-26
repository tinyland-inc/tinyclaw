-- Tinyland deployment configuration
-- Produces config matching tinyland/config.json (recognized fields only)
-- Usage: dhall-to-json --file dhall/examples/tinyland.dhall

let constants = ../constants.dhall
let H = ../helpers.dhall

-- Placeholder substituted by entrypoint.sh at container runtime
let anthropicKey = "__ANTHROPIC_API_KEY__"
let anthropicBase = Some "__ANTHROPIC_BASE_URL__"

let tinyland =
      constants
      // { agents =
            constants.agents
            // { defaults =
                  constants.agents.defaults
                  // { workspace = "/workspace"
                     , restrict_to_workspace = True
                     , model_name = Some "claude-haiku"
                     , model = None Text
                     , max_tokens = 8192
                     , temperature = Some 0.3
                     , max_tool_iterations = 30
                     }
               }
         , model_list =
           [ H.mkModelConfig
               "claude-haiku"
               "anthropic/claude-haiku-4-5-20251001"
               anthropicBase
               anthropicKey
           , H.mkModelConfig
               "claude-sonnet"
               "anthropic/claude-sonnet-4-20250514"
               anthropicBase
               anthropicKey
           ]
         , gateway = { host = "0.0.0.0", port = 18790 }
         , heartbeat = { enabled = True, interval = 120 }
         }

in  tinyland
