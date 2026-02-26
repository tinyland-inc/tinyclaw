# PicoClaw Tool Reference

## Platform Tools via Adapter Proxy (51 total: 15 gateway + 36 Chapel)

Tools are provided by the adapter sidecar's tool proxy, which bridges
rj-gateway MCP tools into PicoClaw's native ToolRegistry format.

Gateway endpoint: `http://rj-gateway.fuzzy-dev.svc.cluster.local:8080/mcp`

### Gateway Tools (7)

| Tool | Parameters | Description |
|------|-----------|-------------|
| `juggler_resolve_composite` | `query`, `sources` (opt) | Multi-source credential resolution (env/SOPS/KDBX/Setec) |
| `juggler_setec_list` | (none) | List all Setec secrets |
| `juggler_setec_get` | `name` | Get secret value (bare name, no prefix) |
| `juggler_setec_put` | `name`, `value` | Store secret value (bare name) |
| `juggler_audit_log` | `count` (opt, default 20) | Query credential access audit trail |
| `juggler_campaign_status` | `campaign_id` (opt) | Campaign runner status / results |
| `juggler_aperture_usage` | `campaign_id` (opt), `agent` (opt) | Aperture token metering data |

### GitHub Tools (8)

| Tool | Parameters | Description |
|------|-----------|-------------|
| `github_fetch` | `owner`, `repo`, `path`, `ref` (opt) | Fetch file contents |
| `github_list_alerts` | `owner`, `repo`, `state` (opt), `severity` (opt) | List CodeQL alerts |
| `github_get_alert` | `owner`, `repo`, `alert_number` | Get specific CodeQL alert |
| `github_create_branch` | `owner`, `repo`, `branch_name`, `base` (opt) | Create branch from base ref |
| `github_update_file` | `owner`, `repo`, `path`, `content`, `message`, `branch` | Create/update file |
| `github_create_pr` | `owner`, `repo`, `title`, `head`, `body` (opt), `base` (opt) | Create pull request |
| `github_create_issue` | `owner`, `repo`, `title`, `body` (opt), `labels` (opt) | Create issue |
| `juggler_request_secret` | `name`, `reason`, `urgency` (opt) | Request secret provisioning (creates labeled issue) |

### Identity Management (7)

| Tool | Parameters | Description |
|------|-----------|-------------|
| `juggler_list_identities` | `provider` (opt) | List all configured git identities |
| `juggler_detect_identity` | `path` (opt) | Auto-detect identity for a repository |
| `juggler_switch` | `identity` | Switch to a different git identity context |
| `juggler_status` | (none) | Current identity context and auth status |
| `juggler_validate` | `identity` | Test SSH and credential connectivity |
| `juggler_token_verify` | (none) | Verify token validity and scopes via API |
| `juggler_sync_config` | (none) | Synchronize SSH/gitconfig blocks |

### GPG, Signing & Trusted Workstation (6)

| Tool | Parameters | Description |
|------|-----------|-------------|
| `juggler_gpg_status` | (none) | Check GPG/SSH signing readiness |
| `juggler_security_mode` | `mode` (opt) | Get/set GPG signing security mode |
| `juggler_pin_store` | (none) | Store YubiKey PIN in HSM |
| `juggler_pin_clear` | (none) | Remove stored PIN from HSM |
| `juggler_tws_status` | (none) | Check Trusted Workstation mode status |
| `juggler_tws_enable` | `identity` | Enable Trusted Workstation mode |

### Token Management (3)

| Tool | Parameters | Description |
|------|-----------|-------------|
| `juggler_store_token` | `token` | Store token in system keychain |
| `juggler_token_get` | (none) | Retrieve stored token (masked) |
| `juggler_token_clear` | (none) | Remove stored token from credential store |

### Key Store / KeePassXC (13)

| Tool | Parameters | Description |
|------|-----------|-------------|
| `juggler_keys_status` | (none) | Check KeePassXC availability and lock state |
| `juggler_keys_search` | `query` | Fuzzy search across key store entries |
| `juggler_keys_get` | `path` | Retrieve secret by entry path |
| `juggler_keys_store` | `path`, `value` | Store/update secret in key store |
| `juggler_keys_list` | `group` (opt) | List entries in a key store group |
| `juggler_keys_resolve` | `query` | Search and retrieve secret in one call |
| `juggler_keys_delete` | `path` | Delete entry from key store |
| `juggler_keys_init` | (none) | Bootstrap new KeePassXC database |
| `juggler_keys_ingest_env` | `path` | Ingest .env file into key store |
| `juggler_keys_crawl_env` | `path` (opt) | Find and ingest all .env files recursively |
| `juggler_keys_discover` | (none) | Auto-discover credentials from env/SSH |
| `juggler_keys_export` | `group`, `format` (opt) | Export key store group as .env or JSON |
| `juggler_keys_sops_status` | (none) | Check SOPS+age integration status |

### SOPS Integration (3)

| Tool | Parameters | Description |
|------|-----------|-------------|
| `juggler_keys_sops_ingest` | `path` | Ingest SOPS-encrypted file into key store |
| `juggler_keys_sops_sync` | `path` | Sync SOPS file with stored entries |
| `juggler_keys_sops_export` | (none) | Export age public key for SOPS config |

### Setup & Debug (3)

| Tool | Parameters | Description |
|------|-----------|-------------|
| `juggler_setup` | (none) | Run first-time setup wizard |
| `juggler_config_show` | (none) | Show RemoteJuggler configuration |
| `juggler_debug_ssh` | `identity` (opt) | Debug SSH configuration and connectivity |

## Known Gotchas

- **Setec prefix**: Gateway client adds `remotejuggler/` prefix. Use bare names like `campaigns/oc-dep-audit`
- **GitHub token**: Automatically resolved by gateway. No need to pass tokens
- **Tool proxy**: All tools come through the adapter's tool proxy, not direct MCP
- **Rate limits**: GitHub API has rate limits. Space out bulk fetches
- **Tool timeouts**: MCP tools have a 30s default timeout. Long operations may need retry
- **KeePassXC**: Keys tools require auto-unlock via HSM. May not be available in-cluster
- **SOPS tools**: Require `sops` and `age` binaries. Check with `juggler_keys_sops_status()` first

## Preferred Patterns

```
# Credential resolution (multi-source)
juggler_resolve_composite(query="github-token")

# Fetch a file from GitHub
github_fetch(owner="tinyland-inc", repo="picoclaw", path="go.mod")

# Check identity status
juggler_status()

# Request a missing secret
juggler_request_secret(name="brave-api-key", reason="Web search", urgency="low")
```
