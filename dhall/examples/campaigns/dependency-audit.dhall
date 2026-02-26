-- Example campaign: audit project dependencies for vulnerabilities.

let Campaign = ../../types/Campaign.dhall

in  { id = "dependency-audit"
    , name = "Dependency Security Audit"
    , description = "Scan project dependencies for known vulnerabilities and license compliance"
    , targets =
      [ { agent_id = "security-auditor"
        , backend = "picoclaw"
        , config_override = None Text
        }
      ]
    , steps =
      [ { name = "scan-go-deps"
        , prompt = "Run go list -m all and check each dependency against known vulnerability databases"
        , tools = [ "exec_command", "web_search" ]
        , timeout_minutes = 10
        }
      , { name = "check-licenses"
        , prompt = "Verify all dependencies use compatible licenses (MIT, Apache-2.0, BSD)"
        , tools = [ "exec_command", "read_file" ]
        , timeout_minutes = 10
        }
      , { name = "report"
        , prompt = "Create a summary of findings with remediation recommendations"
        , tools = [] : List Text
        , timeout_minutes = 5
        }
      ]
    , guardrails = Campaign.readOnlyGuardrails
    , feedback = Campaign.FeedbackPolicy.CreateGitHubIssue
    , tags = [ "security", "dependencies", "weekly" ]
    } : Campaign.CampaignDefinition
