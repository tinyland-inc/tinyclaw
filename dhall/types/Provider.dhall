-- Provider and Model configuration types mirroring pkg/config/config.go

let ModelConfig =
      { model_name : Text
      , model : Text
      , api_base : Optional Text
      , api_key{- -} : Text
      , proxy : Optional Text
      , auth_method : Optional Text
      , connect_mode : Optional Text
      , workspace : Optional Text
      , rpm : Optional Natural
      , max_tokens_field : Optional Text
      }

in  { ModelConfig }
