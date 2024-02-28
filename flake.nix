{
  description = "Systemd System Extensions using the Nix Store";

  inputs = {
    nixpkgs.url = "nixpkgs/nixpkgs-unstable";
    flake-utils = {
      url = "github:numtide/flake-utils";
    };
  };

  outputs = {
    self,
    nixpkgs,
    flake-utils,
  }:
    flake-utils.lib.eachDefaultSystem (
      system: let
        pkgs = import nixpkgs {inherit system;};

        bext_deps = {
          build = with pkgs; [go pkg-config];
          runtime = with pkgs; [btrfs-progs gpgme lvm2];
        };
      in {
        formatter = pkgs.alejandra;
        devShells.default = pkgs.mkShell {
          packages = with pkgs; [cobra-cli gopls eclint apko melange golangci-lint errcheck go-tools] ++ bext_deps.build ++ bext_deps.runtime;
        };
      }
    );
}
