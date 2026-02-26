-- Campaign type definitions for campaign-driven agent orchestration.
--
-- A campaign defines a structured task dispatched to one or more agents
-- with guardrails (budget, duration, killswitch) and feedback policies.

let GuardrailsConfig =
      { max_duration_minutes : Natural
      , read_only : Bool
      , ai_api_budget_cents : Natural
      , kill_switch : Bool
      , max_tool_calls : Natural
      , max_iterations : Natural
      }

let FeedbackPolicy =
      < CreateGitHubIssue
      | CreateGitHubPR
      | PostToChannel
      | StoreInSetec
      | NoFeedback
      >

let CampaignTarget =
      { agent_id : Text
      , backend : Text       -- "picoclaw" | "ironclaw" | "hexstrike"
      , config_override : Optional Text  -- JSON config overlay
      }

let ProcessStep =
      { name : Text
      , prompt : Text
      , tools : List Text
      , timeout_minutes : Natural
      }

let CampaignStatus =
      < Pending | Running | Completed | Failed | Cancelled >

let CampaignDefinition =
      { id : Text
      , name : Text
      , description : Text
      , targets : List CampaignTarget
      , steps : List ProcessStep
      , guardrails : GuardrailsConfig
      , feedback : FeedbackPolicy
      , tags : List Text
      }

let defaultGuardrails
    : GuardrailsConfig
    = { max_duration_minutes = 60
      , read_only = False
      , ai_api_budget_cents = 1000
      , kill_switch = True
      , max_tool_calls = 100
      , max_iterations = 50
      }

let readOnlyGuardrails
    : GuardrailsConfig
    = { max_duration_minutes = 30
      , read_only = True
      , ai_api_budget_cents = 500
      , kill_switch = True
      , max_tool_calls = 50
      , max_iterations = 25
      }

in  { GuardrailsConfig
    , FeedbackPolicy
    , CampaignTarget
    , ProcessStep
    , CampaignStatus
    , CampaignDefinition
    , defaultGuardrails
    , readOnlyGuardrails
    }
