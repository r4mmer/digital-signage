{
  description = "Digital Signage for Raspberry Pi";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};

        # Cross-compilation targets
        crossTargets = {
          pi-zero = {
            GOOS = "linux";
            GOARCH = "arm";
            GOARM = "6";
            suffix = "pi-zero";
          };
          pi-3 = {
            GOOS = "linux";
            GOARCH = "arm";
            GOARM = "7";
            suffix = "pi-3";
          };
          pi-4 = {
            GOOS = "linux";
            GOARCH = "arm64";
            GOARM = "";
            suffix = "pi-4";
          };
        };

        buildForTarget = target: pkgs.stdenv.mkDerivation {
          pname = "digital-signage-${target.suffix}";
          version = "1.0.0";

          src = ./.;

          nativeBuildInputs = [ pkgs.go ];

          buildPhase = ''
            export GOOS=${target.GOOS}
            export GOARCH=${target.GOARCH}
            ${if target.GOARM != "" then "export GOARM=${target.GOARM}" else ""}
            export CGO_ENABLED=0

            go mod download
            go build -ldflags="-w -s" -o digital-signage-${target.suffix} .
          '';

          installPhase = ''
            mkdir -p $out/bin
            cp digital-signage-${target.suffix} $out/bin/
          '';
        };

      in {
        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            go
            gopls
            gotools
            go-tools
            awscli2
          ];

          shellHook = ''
            echo "Digital Signage Development Environment"
            echo "Go version: $(go version)"
            echo ""
            echo "Available commands:"
            echo "  go run . - Run the application locally"
            echo "  ./build.sh - Cross-compile for all Raspberry Pi targets"
            echo "  aws configure - Set up AWS credentials for S3 sync"
            echo ""
          '';
        };

        packages = {
          default = pkgs.stdenv.mkDerivation {
            pname = "digital-signage";
            version = "1.0.0";

            src = ./.;

            nativeBuildInputs = [ pkgs.go ];

            buildPhase = ''
              go mod download
              go build -ldflags="-w -s" -o digital-signage .
            '';

            installPhase = ''
              mkdir -p $out/bin
              cp digital-signage $out/bin/
            '';
          };

          pi-zero = buildForTarget crossTargets.pi-zero;
          pi-3 = buildForTarget crossTargets.pi-3;
          pi-4 = buildForTarget crossTargets.pi-4;
        };

        apps.default = flake-utils.lib.mkApp {
          drv = self.packages.${system}.default;
        };
      });
}
