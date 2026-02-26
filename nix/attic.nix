# Attic binary cache configuration for PicoClaw.
#
# Pushes Nix build artifacts to an Attic cache server, following the
# gloriousflywheel watch-store pattern. This enables fast CI builds
# by sharing build outputs across machines.
#
# Usage:
#   nix build .#picoclaw
#   attic push picoclaw ./result
#
# Or via justfile:
#   just attic-push
{ pkgs ? import <nixpkgs> {} }:

let
  # Attic cache configuration
  cacheConfig = {
    server = "https://cache.tinyland.dev";
    cache = "picoclaw";
    # Auth token should be provided via ATTIC_TOKEN env var
  };

  # Script to push build results to Attic
  pushScript = pkgs.writeShellScriptBin "picoclaw-attic-push" ''
    set -euo pipefail

    ATTIC_SERVER="''${ATTIC_SERVER:-${cacheConfig.server}}"
    ATTIC_CACHE="''${ATTIC_CACHE:-${cacheConfig.cache}}"

    if [ -z "''${ATTIC_TOKEN:-}" ]; then
      echo "Error: ATTIC_TOKEN environment variable not set" >&2
      exit 1
    fi

    # Configure attic
    ${pkgs.attic-client}/bin/attic login picoclaw "$ATTIC_SERVER" "$ATTIC_TOKEN"

    # Push all flake outputs
    for output in picoclaw dhall-config picoclaw-bundle; do
      echo "Pushing $output..."
      nix build ".#$output" --no-link --print-out-paths | \
        xargs ${pkgs.attic-client}/bin/attic push "$ATTIC_CACHE" 2>/dev/null || \
        echo "  Warning: failed to push $output (may not exist)"
    done

    echo "Push complete"
  '';

  # Script to verify fixed-point build
  fixedPointCheck = pkgs.writeShellScriptBin "picoclaw-fixed-point-check" ''
    set -euo pipefail

    echo "Verifying fixed-point build..."

    # Build once
    FIRST=$(nix build .#picoclaw --no-link --print-out-paths)
    FIRST_HASH=$(nix hash path "$FIRST")

    # Build again (should be identical)
    SECOND=$(nix build .#picoclaw --no-link --print-out-paths --rebuild)
    SECOND_HASH=$(nix hash path "$SECOND")

    if [ "$FIRST_HASH" = "$SECOND_HASH" ]; then
      echo "Fixed-point verified: $FIRST_HASH"
      exit 0
    else
      echo "Fixed-point FAILED"
      echo "  First build:  $FIRST_HASH"
      echo "  Second build: $SECOND_HASH"
      exit 1
    fi
  '';

in {
  inherit pushScript fixedPointCheck cacheConfig;

  # Shell hook for development
  shellHook = ''
    export ATTIC_SERVER="${cacheConfig.server}"
    export ATTIC_CACHE="${cacheConfig.cache}"
  '';
}
