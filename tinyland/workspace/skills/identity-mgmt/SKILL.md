---
name: identity-mgmt
description: RemoteJuggler identity query and validation tools
version: "1.0"
tags: [identity, ssh, credentials]
---

# Identity Management

Query identity state via rj-gateway tools (proxied by adapter).

## Tools
- `juggler_status()` -- current identity context
- `juggler_list_identities(provider="all")` -- all identities
- `juggler_validate(identity="rj-agent-bot")` -- test connectivity
- `juggler_token_verify()` -- verify token validity

## Bot Identity
- Name: rj-agent-bot[bot]
- App ID: 2945224
- SSH: /home/agent/.ssh/id_ed25519
