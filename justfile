# PicoClaw Verified Core - Build System
# Subsystems: Go gateway, Dhall config, F* core, Futhark kernels, Nix packaging

# ─── Variables ──────────────────────────────────────────────────────────────────

binary_name := "picoclaw"
build_dir := "build"
cmd_dir := "cmd/" + binary_name
internal := "github.com/sipeed/picoclaw/cmd/picoclaw/internal"

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

# Run all tests
test: go-test

# Verified build: dhall check + go build (+ fstar-check + futhark-test in future)
verified-build: dhall-check build

# ─── Dhall Config ───────────────────────────────────────────────────────────────

# Type-check all Dhall files
dhall-check:
    @echo "Checking Dhall types..."
    find dhall -name '*.dhall' -exec dhall type --file {} \; > /dev/null
    @echo "Dhall type-check passed"

# Render Dhall config to JSON
dhall-render target="defaults":
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

# ─── F* Verified Core (Sprint 2+) ──────────────────────────────────────────────

# Type-check and verify all F* modules
fstar-check:
    @echo "F* verification not yet implemented (Sprint 2)"
    @exit 1

# Extract F* to OCaml
fstar-extract:
    @echo "F* extraction not yet implemented (Sprint 2)"
    @exit 1

# Build extracted OCaml binary
fstar-build: fstar-extract
    @echo "F* build not yet implemented (Sprint 2)"
    @exit 1

# ─── Futhark Compute Kernels (Sprint 3+) ───────────────────────────────────────

# Type-check Futhark programs
futhark-check:
    @echo "Futhark check not yet implemented (Sprint 3)"
    @exit 1

# Build Futhark C backend
futhark-build:
    @echo "Futhark build not yet implemented (Sprint 3)"
    @exit 1

# Run Futhark tests
futhark-test:
    @echo "Futhark tests not yet implemented (Sprint 3)"
    @exit 1

# ─── Nix ────────────────────────────────────────────────────────────────────────

# Build all Nix packages
nix-build:
    nix build

# Run all Nix flake checks
nix-check:
    nix flake check

# Build Docker image via nix2container (Sprint 4+)
nix-docker:
    @echo "Nix Docker build not yet implemented (Sprint 4)"
    @exit 1

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
