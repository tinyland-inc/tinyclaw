{
  description = "PicoClaw - Verified agent framework";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};

        version =
          if self ? rev
          then builtins.substring 0 8 self.rev
          else "dev";

        ldflags = [
          "-X github.com/sipeed/picoclaw/cmd/picoclaw/internal.version=${version}"
          "-X github.com/sipeed/picoclaw/cmd/picoclaw/internal.gitCommit=${version}"
          "-s" "-w"
        ];
      in
      {
        packages = {
          # Go gateway binary
          picoclaw = pkgs.buildGoModule {
            pname = "picoclaw";
            inherit version;
            src = ./.;
            vendorHash = null; # Set after first build or use goModules
            CGO_ENABLED = 0;
            inherit ldflags;
            subPackages = [ "cmd/picoclaw" ];

            # Skip tests that require network
            doCheck = false;

            meta = {
              description = "Ultra-lightweight personal AI agent";
              license = pkgs.lib.licenses.mit;
            };
          };

          # Dhall config package - renders all configs to JSON
          dhall-config = pkgs.stdenv.mkDerivation {
            pname = "picoclaw-dhall-config";
            inherit version;
            src = ./dhall;

            nativeBuildInputs = with pkgs; [ dhall dhall-json ];

            buildPhase = ''
              # Type-check all Dhall files
              find . -name '*.dhall' -exec dhall type --file {} \; > /dev/null

              # Render examples
              mkdir -p rendered
              for example in examples/*.dhall; do
                name=$(basename "$example" .dhall)
                dhall-to-json --file "$example" --output "rendered/$name.json"
              done
            '';

            installPhase = ''
              mkdir -p $out/share/picoclaw
              cp -r rendered/* $out/share/picoclaw/
              cp -r types $out/share/picoclaw/types
            '';
          };

          default = self.packages.${system}.picoclaw;
        };

        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            # Go gateway
            go
            golangci-lint
            goreleaser

            # Dhall config
            dhall
            dhall-json
            dhall-lsp-server

            # Build system
            just
            jq

            # Nix tools
            direnv
            nix-direnv
          ];

          shellHook = ''
            echo "picoclaw dev shell"
            echo "  just --list    # available targets"
          '';
        };

        # Flake checks
        checks = {
          dhall-typecheck = pkgs.stdenv.mkDerivation {
            pname = "picoclaw-dhall-check";
            inherit version;
            src = ./dhall;
            nativeBuildInputs = with pkgs; [ dhall dhall-json ];
            buildPhase = ''
              find . -name '*.dhall' -exec dhall type --file {} \; > /dev/null
            '';
            installPhase = "mkdir -p $out && touch $out/ok";
          };
        };
      }
    );
}
