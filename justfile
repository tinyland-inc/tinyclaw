# PicoClaw Verified Core - Build System
# Subsystems: Go gateway, Dhall config, F* core, Futhark kernels, Nix packaging

# ─── Variables ──────────────────────────────────────────────────────────────────

binary_name := "picoclaw"
build_dir := "build"
cmd_dir := "cmd/" + binary_name
internal := "github.com/tinyland-inc/picoclaw/cmd/picoclaw/internal"

install_prefix := env("INSTALL_PREFIX", env("HOME", "") + "/.local")
install_bin := install_prefix + "/bin"

picoclaw_home := env("PICOCLAW_HOME", env("HOME", "") + "/.picoclaw")
workspace_dir := env("WORKSPACE_DIR", picoclaw_home + "/workspace")

# Version info from git
version := `git describe --tags --always --dirty 2>/dev/null || echo "dev"`
git_commit := `git rev-parse --short=8 HEAD 2>/dev/null || echo "dev"`
build_time := `date +%FT%T%z`
go_version := `go version | awk '{print $3}'`

ldflags := "-X " + internal + ".version=" + version + " -X " + internal + ".gitCommit=" + git_commit + " -X " + internal + ".buildTime=" + build_time + " -X " + internal + ".goVersion=" + go_version + " -s -w"

go := env("GO", "CGO_ENABLED=0 go")
goflags := env("GOFLAGS", "-v -tags stdjson")

# Platform detection
platform := if os() == "macos" { "darwin" } else { os() }
arch := if arch() == "aarch64" { "arm64" } else if arch() == "x86_64" { "amd64" } else { arch() }

# ─── Default ────────────────────────────────────────────────────────────────────

default: build

# ─── Composite Targets ─────────────────────────────────────────────────────────

# Run all checks (dhall + go lint + go test)
check: dhall-check go-lint go-test

# Build everything (dhall render + go build)
build: dhall-render go-build

# Run all tests (unit + e2e)
test: go-test e2e-test

# Verified build: dhall check + go build + fstar check + futhark test
verified-build: dhall-check build fstar-check fstar-build

# ─── Dhall Config ───────────────────────────────────────────────────────────────

# Type-check all Dhall files
dhall-check:
    @echo "Checking Dhall types..."
    find dhall -name '*.dhall' -exec dhall type --file {} \; > /dev/null
    @echo "Dhall type-check passed"

# Render Dhall config to JSON
dhall-render target="tinyland":
    @mkdir -p dhall/rendered
    dhall-to-json --file dhall/examples/{{target}}.dhall --output dhall/rendered/{{target}}.json
    @echo "Rendered dhall/examples/{{target}}.dhall -> dhall/rendered/{{target}}.json"

# Diff Dhall-rendered JSON against an existing config
dhall-diff target:
    @mkdir -p dhall/rendered
    dhall-to-json --file dhall/examples/{{target}}.dhall --output dhall/rendered/{{target}}.json
    diff <(jq -S . dhall/rendered/{{target}}.json) <(jq -S . {{target}}/config.json) || true

# ─── Go Gateway ─────────────────────────────────────────────────────────────────

# Run go generate
go-generate:
    @echo "Running go generate..."
    @rm -rf ./{{cmd_dir}}/workspace 2>/dev/null || true
    {{go}} generate ./...

# Build the Go gateway binary for current platform
go-build: go-generate
    @echo "Building {{binary_name}} for {{platform}}/{{arch}}..."
    @mkdir -p {{build_dir}}
    {{go}} build {{goflags}} -ldflags "{{ldflags}}" -o {{build_dir}}/{{binary_name}}-{{platform}}-{{arch}} ./{{cmd_dir}}
    @ln -sf {{binary_name}}-{{platform}}-{{arch}} {{build_dir}}/{{binary_name}}
    @echo "Build complete: {{build_dir}}/{{binary_name}}-{{platform}}-{{arch}}"

# Build for all platforms
go-build-all: go-generate
    @echo "Building for multiple platforms..."
    @mkdir -p {{build_dir}}
    GOOS=linux GOARCH=amd64 {{go}} build -ldflags "{{ldflags}}" -o {{build_dir}}/{{binary_name}}-linux-amd64 ./{{cmd_dir}}
    GOOS=linux GOARCH=arm64 {{go}} build -ldflags "{{ldflags}}" -o {{build_dir}}/{{binary_name}}-linux-arm64 ./{{cmd_dir}}
    GOOS=linux GOARCH=loong64 {{go}} build -ldflags "{{ldflags}}" -o {{build_dir}}/{{binary_name}}-linux-loong64 ./{{cmd_dir}}
    GOOS=linux GOARCH=riscv64 {{go}} build -ldflags "{{ldflags}}" -o {{build_dir}}/{{binary_name}}-linux-riscv64 ./{{cmd_dir}}
    GOOS=darwin GOARCH=arm64 {{go}} build -ldflags "{{ldflags}}" -o {{build_dir}}/{{binary_name}}-darwin-arm64 ./{{cmd_dir}}
    GOOS=windows GOARCH=amd64 {{go}} build -ldflags "{{ldflags}}" -o {{build_dir}}/{{binary_name}}-windows-amd64.exe ./{{cmd_dir}}
    @echo "All builds complete"

# Run Go tests
go-test:
    {{go}} test ./...

# Run Go linter
go-lint:
    golangci-lint run

# Fix Go lint issues
go-fix:
    golangci-lint run --fix

# Format Go code
go-fmt:
    golangci-lint fmt

# Go vet
go-vet:
    {{go}} vet ./...

# Download and verify Go dependencies
go-deps:
    {{go}} mod download
    {{go}} mod verify

# Update Go dependencies
go-update-deps:
    {{go}} get -u ./...
    {{go}} mod tidy

# ─── E2E Integration Tests ─────────────────────────────────────────────────────

# Run E2E integration tests
e2e-test:
    {{go}} test ./tests/e2e/... -v -count=1

# ─── Migration ─────────────────────────────────────────────────────────────────

# Convert JSON config to Dhall
migrate-to-dhall config="":
    @if [ -n "{{config}}" ]; then \
        {{build_dir}}/{{binary_name}} migrate to-dhall --config "{{config}}"; \
    else \
        {{build_dir}}/{{binary_name}} migrate to-dhall; \
    fi

# Preview Dhall migration (dry-run)
migrate-to-dhall-preview config="":
    @if [ -n "{{config}}" ]; then \
        {{build_dir}}/{{binary_name}} migrate to-dhall --dry-run --config "{{config}}"; \
    else \
        {{build_dir}}/{{binary_name}} migrate to-dhall --dry-run; \
    fi

# ─── F* Verified Core ─────────────────────────────────────────────────────────

# Type-check and verify all F* modules
fstar-check:
    @echo "Checking F* modules..."
    @for f in fstar/src/*.fst; do \
        echo "  Verifying $$f..."; \
        fstar.exe --include fstar/src "$$f" || exit 1; \
    done
    @echo "F* verification passed"

# Extract F* to OCaml
fstar-extract:
    @echo "Extracting F* to OCaml..."
    @mkdir -p fstar/extracted/lib
    @for f in fstar/src/*.fst; do \
        fstar.exe --include fstar/src --codegen OCaml \
            --extract_module "$$(basename "$$f" .fst)" \
            --odir fstar/extracted/lib "$$f" || exit 1; \
    done
    @echo "F* extraction complete"

# Build extracted OCaml binary
fstar-build: fstar-extract
    cd fstar/extracted && dune build
    @echo "OCaml core binary built: fstar/extracted/_build/default/bin/main.exe"

# Build OCaml binary without F* extraction (uses hand-written OCaml)
fstar-build-ocaml:
    cd fstar/extracted && dune build
    @echo "OCaml core binary built: fstar/extracted/_build/default/bin/main.exe"

# Extract F* to C via KaRaMeL
fstar-extract-c:
    @echo "Extracting F* to C via KaRaMeL..."
    @mkdir -p fstar/extracted/c
    krml --skip_compilation \
        -bundle PicoClaw.* \
        -add-include '"picoclaw_ffi.h"' \
        -tmpdir fstar/extracted/c \
        fstar/src/PicoClaw.Types.fst \
        fstar/src/PicoClaw.Routing.fst \
        fstar/src/PicoClaw.ToolAuth.fst \
        fstar/src/PicoClaw.Session.fst \
        fstar/src/PicoClaw.AuditLog.fst \
        fstar/src/PicoClaw.AgentLoop.fst \
        fstar/src/PicoClaw.Protocol.fst \
        fstar/src/PicoClaw.Campaign.fst \
        fstar/src/PicoClaw.Network.fst \
        fstar/src/PicoClaw.Tailscale.fst
    @echo "C extraction complete: fstar/extracted/c/"

# Build verified C binary (KaRaMeL extraction + Futhark C kernels)
fstar-build-verified: fstar-extract-c futhark-build
    @echo "Building verified C binary..."
    @mkdir -p {{build_dir}}
    cc -O2 -Wall -std=c11 \
        -I fstar/extracted/c \
        -I futhark/build \
        fstar/extracted/c/*.c \
        futhark/build/*.c \
        -lm -lpthread \
        -o {{build_dir}}/picoclaw-verified
    @echo "Verified binary: {{build_dir}}/picoclaw-verified"

# Verify F* security proofs (PicoClaw.Proof module)
fstar-proof:
    @echo "Verifying security proofs..."
    fstar.exe --include fstar/src fstar/src/PicoClaw.Proof.fst
    @echo "All security proofs verified"

# ─── Futhark Compute Kernels ──────────────────────────────────────────────────

# Type-check Futhark programs
futhark-check:
    @echo "Checking Futhark programs..."
    @for f in futhark/src/*.fut; do \
        echo "  Checking $$f..."; \
        futhark check "$$f" || exit 1; \
    done
    @echo "Futhark type-check passed"

# Build Futhark C backend (portable CPU)
futhark-build backend="c":
    @echo "Building Futhark kernels ({{backend}} backend)..."
    @mkdir -p futhark/build
    @for f in futhark/src/*.fut; do \
        name=$$(basename "$$f" .fut); \
        echo "  Compiling $$name..."; \
        futhark {{backend}} --library -o "futhark/build/$$name" "$$f" || exit 1; \
    done
    @echo "Futhark build complete: futhark/build/"

# Run Futhark tests
futhark-test:
    @echo "Running Futhark tests..."
    futhark test futhark/src/*.fut
    @echo "Futhark tests passed"

# ─── Nix ────────────────────────────────────────────────────────────────────────

# Build all Nix packages
nix-build:
    nix build

# Run all Nix flake checks
nix-check:
    nix flake check

# Build Docker image via Nix dockerTools
nix-docker:
    nix build .#picoclaw-docker
    @echo "Docker image built: result (load with docker load < result)"

# Build full bundle (gateway + dhall config)
nix-bundle:
    nix build .#picoclaw-bundle
    @echo "Bundle built: result/"

# ─── Campaign ──────────────────────────────────────────────────────────────────

# Run campaign tests
campaign-test:
    {{go}} test ./pkg/campaign/... -v

# ─── Aperture / Tailscale ──────────────────────────────────────────────────────

# Run Aperture integration tests
aperture-test:
    {{go}} test ./pkg/aperture/... ./pkg/tailscale/... -v

# ─── Drift Detection ──────────────────────────────────────────────────────────

# Check for config drift between Dhall source and active JSON config
drift-check config="" dhall="":
    @if [ -n "{{config}}" ] && [ -n "{{dhall}}" ]; then \
        python3 scripts/drift-check.py --config "{{config}}" --dhall "{{dhall}}"; \
    elif [ -n "{{config}}" ]; then \
        python3 scripts/drift-check.py --config "{{config}}"; \
    else \
        python3 scripts/drift-check.py; \
    fi

# ─── Attic Binary Cache ──────────────────────────────────────────────────────

# Push Nix build artifacts to Attic cache
attic-push:
    @echo "Pushing to Attic cache..."
    nix build .#picoclaw --no-link --print-out-paths | xargs attic push picoclaw
    nix build .#dhall-config --no-link --print-out-paths | xargs attic push picoclaw
    @echo "Push complete"

# Verify fixed-point build (rebuild produces identical output)
fixed-point-check:
    @echo "Verifying fixed-point build..."
    @FIRST=$$(nix build .#picoclaw --no-link --print-out-paths) && \
    FIRST_HASH=$$(nix hash path "$$FIRST") && \
    SECOND=$$(nix build .#picoclaw --no-link --print-out-paths --rebuild) && \
    SECOND_HASH=$$(nix hash path "$$SECOND") && \
    if [ "$$FIRST_HASH" = "$$SECOND_HASH" ]; then \
        echo "Fixed-point verified: $$FIRST_HASH"; \
    else \
        echo "Fixed-point FAILED: $$FIRST_HASH != $$SECOND_HASH"; \
        exit 1; \
    fi

# ─── Installation ──────────────────────────────────────────────────────────────

# Install binary to INSTALL_PREFIX/bin
install: go-build
    @echo "Installing {{binary_name}}..."
    @mkdir -p {{install_bin}}
    @cp {{build_dir}}/{{binary_name}} {{install_bin}}/{{binary_name}}.new
    @chmod +x {{install_bin}}/{{binary_name}}.new
    @mv -f {{install_bin}}/{{binary_name}}.new {{install_bin}}/{{binary_name}}
    @echo "Installed to {{install_bin}}/{{binary_name}}"

# Remove binary
uninstall:
    @rm -f {{install_bin}}/{{binary_name}}
    @echo "Removed {{install_bin}}/{{binary_name}}"

# Remove binary and all data
uninstall-all: uninstall
    @rm -rf {{picoclaw_home}}
    @echo "Removed {{picoclaw_home}}"

# ─── Utilities ──────────────────────────────────────────────────────────────────

# Remove build artifacts
clean:
    @rm -rf {{build_dir}}
    @rm -rf dhall/rendered
    @echo "Clean complete"

# Build and run with arguments
run *args: go-build
    {{build_dir}}/{{binary_name}} {{args}}
