-- Unified policy type aggregating all policy definitions.
-- These are enforcement rules consumed by the F* verified core.

let ToolAuth = ../policy/tool-auth.dhall
let Routing = ../policy/routing.dhall
let AperturePolicy = ../policy/aperture.dhall

let Policy =
      { tool_auth : ToolAuth.Policy
      , routing : List Routing.CascadeRule
      , aperture_rules : List AperturePolicy.ApertureRule
      }

in  { Policy, ToolAuth, Routing, AperturePolicy }
