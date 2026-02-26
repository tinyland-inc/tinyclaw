-- Default configuration values matching pkg/config/defaults.go DefaultConfig()

let Types = ./types/package.dhall
let Agent = Types.Agent
let Channel = Types.Channel
let Tool = Types.Tool
let H = ./helpers.dhall

let emptyStrings = [] : List Text

let defaults
    : ./types/Config.dhall
    = { agents =
        { defaults =
          { workspace = "~/.picoclaw/workspace"
          , restrict_to_workspace = True
          , provider = ""
          , model_name = None Text
          , model = Some "glm-4.7"
          , model_fallbacks = emptyStrings
          , image_model = None Text
          , image_model_fallbacks = emptyStrings
          , max_tokens = 8192
          , temperature = None Double
          , max_tool_iterations = 20
          }
        , list = [] : List Agent.AgentConfig
        }
      , bindings = [] : List Types.Binding.AgentBinding
      , session =
        { dm_scope = "main"
        , identity_links = [] : List { mapKey : Text, mapValue : List Text }
        }
      , channels =
        { whatsapp =
          { enabled = False
          , bridge_url = "ws://localhost:3001"
          , allow_from = emptyStrings
          }
        , telegram =
          { enabled = False
          , token = ""
          , proxy = ""
          , allow_from = emptyStrings
          }
        , feishu =
          { enabled = False
          , app_id = ""
          , app_secret{- -} = ""
          , encrypt_key = ""
          , verification_token = ""
          , allow_from = emptyStrings
          }
        , discord =
          { enabled = False
          , token = ""
          , allow_from = emptyStrings
          , mention_only = False
          }
        , maixcam =
          { enabled = False
          , host = "0.0.0.0"
          , port = 18790
          , allow_from = emptyStrings
          }
        , qq =
          { enabled = False
          , app_id = ""
          , app_secret{- -} = ""
          , allow_from = emptyStrings
          }
        , dingtalk =
          { enabled = False
          , client_id = ""
          , client_secret{- -} = ""
          , allow_from = emptyStrings
          }
        , slack =
          { enabled = False
          , bot_token = ""
          , app_token = ""
          , allow_from = emptyStrings
          }
        , line =
          { enabled = False
          , channel_secret{- -} = ""
          , channel_access_token{- -} = ""
          , webhook_host = "0.0.0.0"
          , webhook_port = 18791
          , webhook_path = "/webhook/line"
          , allow_from = emptyStrings
          }
        , onebot =
          { enabled = False
          , ws_url = "ws://127.0.0.1:3001"
          , access_token{- -} = ""
          , reconnect_interval = 5
          , group_trigger_prefix = emptyStrings
          , allow_from = emptyStrings
          }
        , wecom =
          { enabled = False
          , token = ""
          , encoding_aes_key = ""
          , webhook_url = ""
          , webhook_host = "0.0.0.0"
          , webhook_port = 18793
          , webhook_path = "/webhook/wecom"
          , allow_from = emptyStrings
          , reply_timeout = 5
          }
        , wecom_app =
          { enabled = False
          , corp_id = ""
          , corp_secret{- -} = ""
          , agent_id = +0
          , token = ""
          , encoding_aes_key = ""
          , webhook_host = "0.0.0.0"
          , webhook_port = 18792
          , webhook_path = "/webhook/wecom-app"
          , allow_from = emptyStrings
          , reply_timeout = 5
          }
        }
      , model_list =
          [ H.emptyModelConfig
              "glm-4.7"
              "zhipu/glm-4.7"
              (Some "https://open.bigmodel.cn/api/paas/v4")
          ]
      , gateway = { host = "127.0.0.1", port = 18790 }
      , tools =
        { web =
          { brave = H.emptyBrave
          , tavily = H.emptyTavily
          , duckduckgo = { enabled = True, max_results = 5 }
          , perplexity = H.emptyPerplexity
          , proxy = ""
          }
        , cron = { exec_timeout_minutes = 5 }
        , exec =
          { enable_deny_patterns = True
          , custom_deny_patterns = emptyStrings
          }
        , skills =
          { registries =
            { clawhub =
              { enabled = True
              , base_url = "https://clawhub.ai"
              , auth_token{- -} = ""
              , search_path = ""
              , skills_path = ""
              , download_path = ""
              , timeout = 0
              , max_zip_size = 0
              , max_response_size = 0
              }
            }
          , max_concurrent_searches = 2
          , search_cache = { max_size = 50, ttl_seconds = 300 }
          }
        }
      , heartbeat = { enabled = True, interval = 30 }
      , devices = { enabled = False, monitor_usb = True }
      , tailscale =
        { enabled = False
        , hostname = "picoclaw"
        , state_dir = ""
        , auth_key{- -} = ""
        }
      , aperture =
        { enabled = False
        , proxy_url = ""
        , webhook_url = ""
        , webhook_key{- -} = ""
        , cerbos_url = ""
        }
      }

in  defaults
