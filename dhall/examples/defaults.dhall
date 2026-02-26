-- Default configuration: renders to the same JSON as Go's DefaultConfig()
-- Usage: dhall-to-json --file dhall/examples/defaults.dhall

let constants = ../constants.dhall

in  constants
