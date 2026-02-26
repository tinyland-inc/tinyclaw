-- Agent binding types mirroring pkg/config/config.go binding structs

let PeerMatch =
      { kind : Text
      , id : Text
      }

let BindingMatch =
      { channel : Text
      , account_id : Optional Text
      , peer : Optional PeerMatch
      , guild_id : Optional Text
      , team_id : Optional Text
      }

let AgentBinding =
      { agent_id : Text
      , match : BindingMatch
      }

in  { PeerMatch, BindingMatch, AgentBinding }
