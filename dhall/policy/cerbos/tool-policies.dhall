-- Cerbos policy definitions for tool-level authorization.
--
-- These policies define which agents can execute which tools, and under
-- what conditions. They are rendered to JSON and loaded by the Cerbos PDP.

let ToolPolicy =
      { tool_name : Text
      , actions : List Text
      , effect : Text          -- "EFFECT_ALLOW" or "EFFECT_DENY"
      , condition : Optional Text  -- CEL expression for conditional access
      }

let ResourcePolicy =
      { resource : Text
      , version : Text
      , rules : List ToolPolicy
      }

-- Default policy: allow all standard tools for any agent
let defaultToolPolicy
    : ResourcePolicy
    = { resource = "tool"
      , version = "default"
      , rules =
        [ { tool_name = "web_search"
          , actions = [ "execute" ]
          , effect = "EFFECT_ALLOW"
          , condition = None Text
          }
        , { tool_name = "read_file"
          , actions = [ "execute" ]
          , effect = "EFFECT_ALLOW"
          , condition = None Text
          }
        , { tool_name = "write_file"
          , actions = [ "execute" ]
          , effect = "EFFECT_ALLOW"
          , condition = Some "request.resource.attr.path.startsWith(P.attr.workspace)"
          }
        , { tool_name = "exec_command"
          , actions = [ "execute" ]
          , effect = "EFFECT_ALLOW"
          , condition = Some "request.resource.attr.restrict_to_workspace == false || request.resource.attr.path.startsWith(P.attr.workspace)"
          }
        ]
      }

-- Restricted policy: deny dangerous tools
let restrictedToolPolicy
    : ResourcePolicy
    = { resource = "tool"
      , version = "restricted"
      , rules =
        [ { tool_name = "exec_command"
          , actions = [ "execute" ]
          , effect = "EFFECT_DENY"
          , condition = None Text
          }
        , { tool_name = "write_file"
          , actions = [ "execute" ]
          , effect = "EFFECT_DENY"
          , condition = None Text
          }
        ]
      }

in  { ToolPolicy
    , ResourcePolicy
    , defaultToolPolicy
    , restrictedToolPolicy
    }
