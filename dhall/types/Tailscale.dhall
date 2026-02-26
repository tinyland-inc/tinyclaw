-- Tailscale tsnet integration configuration
-- Mirrors pkg/config/config.go TailscaleConfig struct

let Tailscale =
      { enabled : Bool
      , hostname : Text
      , state_dir : Text
      , auth_key{- -} : Text
      }

in  { Tailscale }
