-- Example campaign: automated code review of recent commits.

let Campaign = ../../types/Campaign.dhall

in  { id = "code-review-daily"
    , name = "Daily Code Review"
    , description = "Review recent commits for security issues, code quality, and documentation gaps"
    , targets =
      [ { agent_id = "code-reviewer"
        , backend = "picoclaw"
        , config_override = None Text
        }
      ]
    , steps =
      [ { name = "fetch-changes"
        , prompt = "List all commits from the last 24 hours with their diffs"
        , tools = [ "exec_command", "read_file" ]
        , timeout_minutes = 5
        }
      , { name = "security-review"
        , prompt = "Review each diff for security vulnerabilities: credential exposure, injection risks, unsafe deserialization, SSRF"
        , tools = [ "web_search", "read_file" ]
        , timeout_minutes = 15
        }
      , { name = "quality-review"
        , prompt = "Check for code quality issues: error handling, resource leaks, race conditions, missing tests"
        , tools = [ "read_file" ]
        , timeout_minutes = 15
        }
      , { name = "report"
        , prompt = "Summarize findings as a structured report with severity ratings"
        , tools = [] : List Text
        , timeout_minutes = 5
        }
      ]
    , guardrails = Campaign.readOnlyGuardrails
    , feedback = Campaign.FeedbackPolicy.CreateGitHubIssue
    , tags = [ "security", "quality", "daily" ]
    } : Campaign.CampaignDefinition
