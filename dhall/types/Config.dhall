-- Top-level Config type mirroring pkg/config/config.go Config struct

let Agent = ./Agent.dhall
let Aperture = ./Aperture.dhall
let Binding = ./Binding.dhall
let Session = ./Session.dhall
let Channel = ./Channel.dhall
let Provider = ./Provider.dhall
let Gateway = ./Gateway.dhall
let Tailscale = ./Tailscale.dhall
let Tool = ./Tool.dhall
let Heartbeat = ./Heartbeat.dhall
let Device = ./Device.dhall

let Config =
      { agents : Agent.Agents
      , bindings : List Binding.AgentBinding
      , session : Session.Session
      , channels : Channel.Channels
      , model_list : List Provider.ModelConfig
      , gateway : Gateway.Gateway
      , tools : Tool.Tools
      , heartbeat : Heartbeat.Heartbeat
      , devices : Device.Devices
      , tailscale : Tailscale.Tailscale
      , aperture : Aperture.Aperture
      }

in  Config
