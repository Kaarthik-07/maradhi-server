{
  description = "Maradhi API — Go backend for the Maradhi personal productivity app.";

  inputs = {
    nixpkgs.url     = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};

        # ── Pick Go version ──────────────────────────────────────────────
        # nixpkgs-unstable only keeps the two latest Go minor versions.
        # `pkgs.go` always points to the current stable (1.23 / 1.24 / etc.)
        # Our go.mod says "go 1.22" which means "minimum 1.22" — any newer
        # Go is forward-compatible, so pkgs.go is always correct here.
        # If you ever need an exact older version, pin nixpkgs to a specific
        # commit where that version still exists.
        goVersion = pkgs.go;

        appName    = "maradhi-api";
        appVersion = "0.1.0";

        # ── Binary build ─────────────────────────────────────────────────
        maradhi-api = pkgs.buildGoModule {
          pname   = appName;
          version = appVersion;
          src     = pkgs.lib.cleanSourceWith {
            src    = ./.;
            filter = path: type:
              let base = baseNameOf path;
              in !(pkgs.lib.hasPrefix "." base)
              && base != "flake.nix"
              && base != "flake.lock"
              && base != "nix"
              && base != "result";
          };

          # After running `nix build` for the first time it will print the
          # correct hash — replace lib.fakeHash with that value.
          vendorHash = pkgs.lib.fakeHash;

          subPackages = [ "cmd/api" ];

          ldflags = [ "-s" "-w" "-X main.version=${appVersion}" ];

          # Pure Go binary — no C deps, cross-compilation friendly
          CGO_ENABLED = "0";

          # Use the same Go version we pin in the dev shell
          go = goVersion;

          meta = {
            description = "Maradhi productivity app API";
            license     = pkgs.lib.licenses.mit;
            mainProgram = appName;
          };
        };

        # ── Docker image ─────────────────────────────────────────────────
        docker-image = pkgs.dockerTools.buildLayeredImage {
          name     = appName;
          tag      = appVersion;
          contents = with pkgs; [ maradhi-api cacert tzdata ];
          config   = {
            Entrypoint   = [ "${maradhi-api}/bin/api" ];
            ExposedPorts = { "8080/tcp" = {}; };
            Env          = [ "PORT=8080" "ENV=production" ];
          };
        };

      in {
        # nix build / nix build .#docker
        packages = {
          default      = maradhi-api;
          "${appName}" = maradhi-api;
          docker       = docker-image;
        };

        # nix run
        apps.default = flake-utils.lib.mkApp {
          drv  = maradhi-api;
          name = "api";
        };

        # ── nix develop  (full dev shell) ─────────────────────────────────
        devShells.default = pkgs.mkShell {
          packages = with pkgs; [
            # Go toolchain — same version used to build the binary
            goVersion
            gopls             # language server
            gotools           # goimports, godoc, etc.
            delve             # debugger

            # Code quality
            golangci-lint
            govulncheck

            # Live reload — `make watch` or `air`
            air

            # DB CLI — for running migrations with psql
            postgresql_16

            # Utilities
            gnumake
            git
            jq
            curl
          ];

          shellHook = ''
            echo ""
            echo "  MARADHI API — dev shell"
            echo "  Go $(go version | awk '{print $3}')"
            echo ""

            # Load .env automatically
            if [ -f .env ]; then
              set -a; source .env; set +a
              echo "  ✓ .env loaded"
            else
              echo "  ⚠  cp .env.example .env  and fill in Supabase values"
            fi

            # Keep Go cache inside the project dir
            export GOPATH="$PWD/.gopath"
            export PATH="$GOPATH/bin:$PATH"
            mkdir -p "$GOPATH"

            echo ""
            echo "  make run     → start server"
            echo "  make watch   → live reload"
            echo "  make test    → run tests"
            echo "  make migrate → apply SQL to Supabase"
            echo ""
          '';
        };

        # nix develop .#ci  — minimal shell for GitHub Actions
        devShells.ci = pkgs.mkShell {
          packages = with pkgs; [
            goVersion
            golangci-lint
            govulncheck
            gnumake
          ];
        };

        # nix fmt
        formatter = pkgs.nixfmt-rfc-style;

        # nix flake check
        checks.build = maradhi-api;
      }
    );
}
