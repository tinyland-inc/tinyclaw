-- Session configuration type mirroring pkg/config/config.go SessionConfig

let Session =
      { dm_scope : Text
      , identity_links : List { mapKey : Text, mapValue : List Text }
      }

in  { Session }
