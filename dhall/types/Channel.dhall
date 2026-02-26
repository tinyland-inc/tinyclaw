-- Channel configuration types mirroring pkg/config/config.go channel structs

let WhatsApp =
      { enabled : Bool
      , bridge_url : Text
      , allow_from : List Text
      }

let Telegram =
      { enabled : Bool
      , token : Text
      , proxy : Text
      , allow_from : List Text
      }

let Feishu =
      { enabled : Bool
      , app_id : Text
      , app_secret{- -} : Text
      , encrypt_key : Text
      , verification_token : Text
      , allow_from : List Text
      }

let Discord =
      { enabled : Bool
      , token : Text
      , allow_from : List Text
      , mention_only : Bool
      }

let MaixCam =
      { enabled : Bool
      , host : Text
      , port : Natural
      , allow_from : List Text
      }

let QQ =
      { enabled : Bool
      , app_id : Text
      , app_secret{- -} : Text
      , allow_from : List Text
      }

let DingTalk =
      { enabled : Bool
      , client_id : Text
      , client_secret{- -} : Text
      , allow_from : List Text
      }

let Slack =
      { enabled : Bool
      , bot_token : Text
      , app_token : Text
      , allow_from : List Text
      }

let LINE =
      { enabled : Bool
      , channel_secret{- -} : Text
      , channel_access_token{- -} : Text
      , webhook_host : Text
      , webhook_port : Natural
      , webhook_path : Text
      , allow_from : List Text
      }

let OneBot =
      { enabled : Bool
      , ws_url : Text
      , access_token{- -} : Text
      , reconnect_interval : Natural
      , group_trigger_prefix : List Text
      , allow_from : List Text
      }

let WeCom =
      { enabled : Bool
      , token : Text
      , encoding_aes_key : Text
      , webhook_url : Text
      , webhook_host : Text
      , webhook_port : Natural
      , webhook_path : Text
      , allow_from : List Text
      , reply_timeout : Natural
      }

let WeComApp =
      { enabled : Bool
      , corp_id : Text
      , corp_secret{- -} : Text
      , agent_id : Integer
      , token : Text
      , encoding_aes_key : Text
      , webhook_host : Text
      , webhook_port : Natural
      , webhook_path : Text
      , allow_from : List Text
      , reply_timeout : Natural
      }

let Channels =
      { whatsapp : WhatsApp
      , telegram : Telegram
      , feishu : Feishu
      , discord : Discord
      , maixcam : MaixCam
      , qq : QQ
      , dingtalk : DingTalk
      , slack : Slack
      , line : LINE
      , onebot : OneBot
      , wecom : WeCom
      , wecom_app : WeComApp
      }

in  { WhatsApp
    , Telegram
    , Feishu
    , Discord
    , MaixCam
    , QQ
    , DingTalk
    , Slack
    , LINE
    , OneBot
    , WeCom
    , WeComApp
    , Channels
    }
