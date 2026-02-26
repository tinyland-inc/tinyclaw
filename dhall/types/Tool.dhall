-- Tool configuration types mirroring pkg/config/config.go tool structs

let Brave =
      { enabled : Bool
      , api_key{- -} : Text
      , max_results : Natural
      }

let Tavily =
      { enabled : Bool
      , api_key{- -} : Text
      , base_url : Text
      , max_results : Natural
      }

let DuckDuckGo =
      { enabled : Bool
      , max_results : Natural
      }

let Perplexity =
      { enabled : Bool
      , api_key{- -} : Text
      , max_results : Natural
      }

let Web =
      { brave : Brave
      , tavily : Tavily
      , duckduckgo : DuckDuckGo
      , perplexity : Perplexity
      , proxy : Text
      }

let Cron =
      { exec_timeout_minutes : Natural }

let Exec =
      { enable_deny_patterns : Bool
      , custom_deny_patterns : List Text
      }

let SearchCache =
      { max_size : Natural
      , ttl_seconds : Natural
      }

let ClawHub =
      { enabled : Bool
      , base_url : Text
      , auth_token{- -} : Text
      , search_path : Text
      , skills_path : Text
      , download_path : Text
      , timeout : Natural
      , max_zip_size : Natural
      , max_response_size : Natural
      }

let Registries =
      { clawhub : ClawHub }

let Skills =
      { registries : Registries
      , max_concurrent_searches : Natural
      , search_cache : SearchCache
      }

let Tools =
      { web : Web
      , cron : Cron
      , exec : Exec
      , skills : Skills
      }

in  { Brave
    , Tavily
    , DuckDuckGo
    , Perplexity
    , Web
    , Cron
    , Exec
    , SearchCache
    , ClawHub
    , Registries
    , Skills
    , Tools
    }
