{
  description = "Go development environment";
  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { nixpkgs, flake-utils, ... }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs {
          inherit system;
          config.allowUnfree = true;
        };
      in {
        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [ go gopls gotools goreleaser ];

          shellHook = ''
            export GOROOT="${pkgs.go}/share/go"
            export GOPATH="$PWD/.go"
            export PATH="$GOPATH/bin:$PATH"
            export GOBIN="$GOPATH/bin"
          '';
        };
      });
}
