---
name: rj-platform
description: RemoteJuggler platform tools for credential resolution, secrets, and audit
version: "1.0"
tags: [credentials, secrets, audit]
---

# RemoteJuggler Platform

Tools accessed via the adapter's tool proxy (bridges rj-gateway MCP to PicoClaw's native format).

## Key Tools
- `juggler_resolve_composite(query="<name>")` -- resolve secrets from multiple backends
- `juggler_setec_list()` -- list available secrets
- `juggler_setec_get(name="<bare-name>")` -- get a secret (bare name, no prefix)
- `juggler_setec_put(name="<bare-name>", value="<value>")` -- store a secret
- `juggler_audit_log(count=20)` -- recent audit entries
- `juggler_campaign_status()` -- campaign results

## Gotchas
- `juggler_resolve_composite` uses `query` not `name`
- Setec tools use bare names (client adds `remotejuggler/` prefix)
