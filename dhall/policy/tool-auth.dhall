-- Tool authorization policies
-- Defines which tools require explicit grants, which are always allowed,
-- and which are denied by default.
--
-- This is the Dhall source of truth for tool authorization.
-- F*-verified enforcement lives in fstar/src/PicoClaw.ToolAuth.fst.

let ToolGrant =
      { tool_name : Text
      , description : Text
      }

let Policy =
      { always_allowed : List ToolGrant
      , requires_grant : List ToolGrant
      , always_denied : List ToolGrant
      }

let defaultPolicy
    : Policy
    = { always_allowed =
        [ { tool_name = "web_search"
          , description = "Search the web for information"
          }
        , { tool_name = "read_file"
          , description = "Read file contents within workspace"
          }
        , { tool_name = "list_files"
          , description = "List files in workspace directory"
          }
        ]
      , requires_grant =
        [ { tool_name = "execute_command"
          , description = "Execute shell commands"
          }
        , { tool_name = "write_file"
          , description = "Write or modify files"
          }
        , { tool_name = "cron_manage"
          , description = "Create or modify scheduled jobs"
          }
        , { tool_name = "subagent"
          , description = "Spawn sub-agent for delegated tasks"
          }
        ]
      , always_denied =
        [ { tool_name = "i2c_raw"
          , description = "Raw I2C bus access"
          }
        , { tool_name = "spi_raw"
          , description = "Raw SPI bus access"
          }
        ]
      }

in  { ToolGrant, Policy, defaultPolicy }
