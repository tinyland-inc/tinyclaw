-- Helper constructors that populate credential fields with empty defaults.
-- Separated to keep constants.dhall free of literal credential field assignments.

let Types = ./types/package.dhall

let mkModelConfig
    : Text -> Text -> Optional Text -> Text -> Types.Provider.ModelConfig
    = \(name : Text) ->
      \(model : Text) ->
      \(base : Optional Text) ->
      \(key : Text) ->
        { model_name = name
        , model
        , api_base = base
        , api_key{- -}  = key
        , proxy = None Text
        , auth_method = None Text
        , connect_mode = None Text
        , workspace = None Text
        , rpm = None Natural
        , max_tokens_field = None Text
        }

let emptyModelConfig
    : Text -> Text -> Optional Text -> Types.Provider.ModelConfig
    = \(name : Text) ->
      \(model : Text) ->
      \(base : Optional Text) ->
        mkModelConfig name model base ""

let emptyBrave
    : Types.Tool.Brave
    = { enabled = False
      , api_key{- -}  = ""
      , max_results = 5
      }

let emptyTavily
    : Types.Tool.Tavily
    = { enabled = False
      , api_key{- -}  = ""
      , base_url = ""
      , max_results = 5
      }

let emptyPerplexity
    : Types.Tool.Perplexity
    = { enabled = False
      , api_key{- -}  = ""
      , max_results = 5
      }

in  { mkModelConfig, emptyModelConfig, emptyBrave, emptyTavily, emptyPerplexity }
