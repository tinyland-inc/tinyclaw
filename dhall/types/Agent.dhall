-- Agent configuration types mirroring pkg/config/config.go agent structs

let AgentModelConfig =
      { primary : Text
      , fallbacks : List Text
      }

let SubagentsConfig =
      { allow_agents : List Text
      , model : Optional AgentModelConfig
      }

let AgentDefaults =
      { workspace : Text
      , restrict_to_workspace : Bool
      , provider : Text
      , model_name : Optional Text
      , model : Optional Text
      , model_fallbacks : List Text
      , image_model : Optional Text
      , image_model_fallbacks : List Text
      , max_tokens : Natural
      , temperature : Optional Double
      , max_tool_iterations : Natural
      }

let AgentConfig =
      { id : Text
      , default : Bool
      , name : Optional Text
      , workspace : Optional Text
      , model : Optional AgentModelConfig
      , skills : List Text
      , subagents : Optional SubagentsConfig
      }

let Agents =
      { defaults : AgentDefaults
      , list : List AgentConfig
      }

in  { AgentModelConfig
    , SubagentsConfig
    , AgentDefaults
    , AgentConfig
    , Agents
    }
