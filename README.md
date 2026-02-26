<div align="center">
  <img src="assets/logo.jpg" alt="PicoClaw" width="512">

  <h1>PicoClaw</h1>

  <h3>Verified AI Agent Framework</h3>

  <p>
    <img src="https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go&logoColor=white" alt="Go">
    <img src="https://img.shields.io/badge/F*-verified-blueviolet?style=flat" alt="F*">
    <img src="https://img.shields.io/badge/Dhall-config-yellow?style=flat" alt="Dhall">
    <img src="https://img.shields.io/badge/Futhark-kernels-orange?style=flat" alt="Futhark">
    <img src="https://img.shields.io/badge/Nix-reproducible-5277C3?style=flat&logo=nixos&logoColor=white" alt="Nix">
    <br>
    <img src="https://img.shields.io/badge/Arch-x86__64%2C%20ARM64%2C%20RISC--V%2C%20LoongArch-blue" alt="Hardware">
    <img src="https://img.shields.io/badge/license-MIT-green" alt="License">
  </p>
</div>

---

PicoClaw is an ultra-lightweight AI agent framework with a formally verified core. It runs on $10 hardware with <10MB RAM, supports 7 LLM providers and 12 chat integrations, and produces a tamper-evident audit log for every decision.

The verified core is written in [F*](https://www.fstar-lang.org/) (dependently typed language) with proofs of safety properties, [Futhark](https://futhark-lang.org/) parallel compute kernels, [Dhall](https://dhall-lang.org/) typed configuration, and [Nix](https://nixos.org/) reproducible builds.

## Architecture

```
                    ┌───────────────────────────────┐
                    │     Go Gateway (retained)      │
                    │  12 chat adapters  │  Cobra CLI │
                    │  Tailscale tsnet   │  Aperture  │
                    │  Health/cron/heartbeat          │
                    └───────────┬───┬───────────────-─┘
                                │   │ STDIO JSON-RPC
                    ┌───────────▼───▼────────────────┐
                    │   F*-extracted Core (OCaml)     │
                    │  7-level routing cascade        │
                    │  Tool authorization proofs      │
                    │  Hash-chained audit log         │
                    │  Fuel-bounded agent loop        │
                    └───────────┬───┬────────────────┘
                                │   │ OCaml FFI
                    ┌───────────▼───▼────────────────┐
                    │    Futhark Compute Kernels      │
                    │  Batch similarity search        │
                    │  Parallel audit log analysis    │
                    │  Token estimation               │
                    └────────────────────────────────-┘

    Dhall configs ──dhall-to-json──▸ JSON ──stdin──▸ F* Core
```

## Verified Safety Properties

The F* core proves six safety properties ([`PicoClaw.Proof`](fstar/src/PicoClaw.Proof.fst)):

| Property | Guarantee |
|----------|-----------|
| **No unauthorized tool execution** | AlwaysDenied tools never authorize; RequiresGrant needs matching grant |
| **Tamper-evident audit log** | Hash chain grows monotonically; each append adds exactly 1 entry |
| **Budget constraints enforced** | Spending monotonic; kill switch immediate; iterations monotonic |
| **Deterministic routing** | Same inputs always produce same outputs; 7-level cascade always resolves |
| **Loop termination** | Iteration counter advances by exactly 1 per step; bounded by fuel |
| **Session monotonicity** | Message count never decreases after summarization |

## Quick Start

### Prerequisites

- Go 1.21+ (gateway)
- [just](https://github.com/casey/just) (task runner)
- Optional: [Nix](https://nixos.org/) (reproducible dev shell with all tools)

### Install from source

```bash
git clone https://github.com/tinyland-inc/picoclaw.git
cd picoclaw

# Using just (recommended)
just build
just install
```

### Using Nix

```bash
# Enter dev shell with all tools (Go, Dhall, OCaml, Futhark, just)
nix develop

# Build all packages
nix build

# Build verified bundle (gateway + core + config)
nix build .#picoclaw-verified-bundle

# Build Docker image
nix build .#picoclaw-verified-docker
```

### Configure

```bash
# Interactive setup
picoclaw onboard

# Or edit directly
$EDITOR ~/.picoclaw/config.json
```

Minimal config:

```json
{
  "model_list": [
    {
      "model_name": "claude",
      "model": "anthropic/claude-sonnet-4-6",
      "api_key": "sk-ant-..."
    }
  ],
  "agents": {
    "defaults": {
      "model": "claude"
    }
  }
}
```

### Run

```bash
# One-shot
picoclaw agent -m "What is 2+2?"

# Interactive
picoclaw agent

# Gateway (connects chat platforms)
picoclaw gateway
```

## Configuration

PicoClaw supports two configuration formats:

- **JSON** (`~/.picoclaw/config.json`) -- legacy, works out of the box
- **Dhall** (`~/.picoclaw/config.dhall`) -- typed, composable, recommended

Convert existing config: `just migrate-to-dhall`

### Dhall Configuration

```dhall
let Config = ./dhall/types/Config.dhall
let defaults = ./dhall/constants.dhall

in defaults // {
  agents = defaults.agents // {
    defaults = defaults.agents.defaults // {
      model_name = "claude"
    }
  }
}
```

Type-check: `just dhall-check`
Render to JSON: `just dhall-render`
Diff against active config: `just dhall-diff`

### Providers

PicoClaw routes by protocol family (OpenAI-compatible, Anthropic-native, custom):

| Vendor | Prefix | Protocol |
|--------|--------|----------|
| OpenAI | `openai/` | OpenAI |
| Anthropic | `anthropic/` | Anthropic |
| Zhipu | `zhipu/` | OpenAI |
| DeepSeek | `deepseek/` | OpenAI |
| Gemini | `gemini/` | OpenAI |
| Groq | `groq/` | OpenAI |
| Ollama | `ollama/` | OpenAI |
| OpenRouter | `openrouter/` | OpenAI |
| Cerebras | `cerebras/` | OpenAI |
| Qwen | `qwen/` | OpenAI |

Use `vendor/model` in the `model` field (e.g., `anthropic/claude-sonnet-4-6`).

### Chat Platforms

| Channel | Setup |
|---------|-------|
| Telegram | Token from @BotFather |
| Discord | Bot token + MESSAGE CONTENT INTENT |
| Slack | App token + event subscriptions |
| QQ | AppID + AppSecret |
| DingTalk | Client ID + Client Secret |
| LINE | Channel Secret + Access Token + webhook |
| WeCom Bot | Webhook URL |
| WeCom App | CorpID + Secret + AgentId |
| Feishu | App ID + App Secret |
| OneBot | WebSocket/HTTP endpoint |
| WhatsApp | Business API credentials |
| MaixCAM | Local serial connection |

Enable in config under `channels.<name>.enabled = true`.

## Build System

PicoClaw uses `just` as the primary task runner, organized by subsystem:

```bash
just --list              # show all targets

# Core
just build               # dhall-render + go-build
just test                # go-test + e2e-test
just check               # dhall-check + go-lint + go-test
just verified-build      # full verified pipeline

# Dhall
just dhall-check         # type-check all .dhall files
just dhall-render        # render to JSON
just dhall-diff TARGET   # diff rendered vs active config

# Go
just go-build            # build gateway binary
just go-test             # run unit tests
just go-lint             # golangci-lint

# F* (requires F* toolchain)
just fstar-check         # verify all F* modules
just fstar-proof         # verify security theorem
just fstar-build         # extract to OCaml and compile
just fstar-build-ocaml   # build OCaml core (no extraction)

# Futhark
just futhark-check       # type-check Futhark programs
just futhark-build       # compile to C backend
just futhark-test        # run Futhark tests

# Nix
just nix-build           # build all Nix packages
just nix-check           # run flake checks
just nix-docker          # build Docker image

# Operations
just drift-check         # compare Dhall source vs active JSON
just attic-push          # push to Attic binary cache
just fixed-point-check   # verify reproducible build
just campaign-test       # run campaign orchestration tests
just aperture-test       # run Aperture/Tailscale tests

# Installation
just install             # install to ~/.local/bin
just uninstall           # remove binary
```

## Project Structure

```
picoclaw/
├── cmd/picoclaw/           # Go CLI entry point (Cobra)
│   └── internal/           # Gateway, agent, migrate, onboard commands
├── pkg/                    # Go packages
│   ├── agent/              # Agent loop (legacy Go mode)
│   ├── aperture/           # Tailscale Aperture proxy + Cerbos
│   ├── campaign/           # Campaign orchestration + adapters
│   ├── channels/           # 12 chat platform integrations
│   ├── config/             # Config loader (JSON + Dhall)
│   ├── core/               # CoreProxy (JSON-RPC to F* binary)
│   ├── migrate/            # JSON-to-Dhall migration
│   ├── providers/          # LLM provider factory (7 vendors)
│   ├── routing/            # Go routing (legacy mode)
│   ├── tailscale/          # tsnet + Setec integration
│   └── tools/              # Tool registry + sandbox
├── dhall/                  # Typed configuration
│   ├── types/              # Config, Agent, Channel, Campaign, etc.
│   ├── policy/             # Aperture + Cerbos policies
│   ├── lib/tinyland/       # Shared ACL/policy types
│   └── examples/           # Example configs
├── fstar/                  # Formally verified core
│   ├── src/                # 11 F* modules with proofs
│   ├── extracted/          # OCaml extraction + Futhark bindings
│   │   ├── bin/main.ml     # JSON-RPC dispatch
│   │   ├── lib/            # Core logic + JSON codec
│   │   └── c/              # KaRaMeL FFI header
│   └── karamel/            # C extraction config
├── futhark/                # Parallel compute kernels
│   ├── src/                # Similarity, audit, token estimation
│   └── lib/                # Linear algebra primitives
├── nix/                    # Nix modules (Attic cache)
├── scripts/                # Drift detection, utilities
├── tests/e2e/              # Integration tests
├── flake.nix               # Nix flake (packages, dev shell, checks)
└── justfile                # Task runner (primary build system)
```

## Docker

```bash
# Build and start gateway
docker compose --profile gateway up -d

# One-shot agent
docker compose run --rm picoclaw-agent -m "Hello"

# Check logs
docker compose logs -f picoclaw-gateway

# Stop
docker compose --profile gateway down
```

## Security

PicoClaw runs in a sandboxed environment by default (`restrict_to_workspace: true`). File/command access is limited to the configured workspace. Dangerous commands (`rm -rf`, `format`, `dd`, fork bombs) are blocked even with restrictions disabled.

The verified core adds formal guarantees: no tool executes without authorization, and every decision is recorded in a tamper-evident hash-chained audit log.

## CLI Reference

| Command | Description |
|---------|-------------|
| `picoclaw onboard` | Initialize config and workspace |
| `picoclaw agent -m "..."` | One-shot chat |
| `picoclaw agent` | Interactive chat |
| `picoclaw gateway` | Start gateway (chat platforms) |
| `picoclaw gateway --legacy` | Use Go agent loop instead of verified core |
| `picoclaw status` | Show status |
| `picoclaw migrate to-dhall` | Convert JSON config to Dhall |
| `picoclaw cron list` | List scheduled jobs |
| `picoclaw cron add ...` | Add a scheduled job |

## Contributing

PRs welcome. The codebase is organized by subsystem -- pick what interests you:

- **Go gateway** (`pkg/`, `cmd/`) -- chat integrations, providers, tools
- **Dhall config** (`dhall/`) -- type definitions, policy fragments
- **F* core** (`fstar/src/`) -- verified logic, proofs
- **Futhark kernels** (`futhark/src/`) -- parallel algorithms
- **Nix build** (`flake.nix`, `nix/`) -- packaging, CI

## License

MIT
