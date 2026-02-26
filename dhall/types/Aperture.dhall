-- Tailscale Aperture proxy integration configuration
-- Mirrors pkg/config/config.go ApertureConfig struct

let Aperture =
      { enabled : Bool
      , proxy_url : Text
      , webhook_url : Text
      , webhook_key{- -} : Text
      , cerbos_url : Text
      }

in  { Aperture }
