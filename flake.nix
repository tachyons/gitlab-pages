{
  description = "A very basic flake";
  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
  inputs.utils.url = "github:numtide/flake-utils";
  inputs.gomod2nix.url = "github:tweag/gomod2nix";

  outputs = { self, nixpkgs, utils, gomod2nix }:
    utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs {
          inherit system;
          overlays = [ gomod2nix.overlay ];
        };
      in rec {
        devShell = pkgs.mkShell {
          buildInputs = [ packages.gitlab-pages pkgs.gomod2nix ];
        };
        packages.gitlab-pages = pkgs.buildGoApplication {
          pname = "gitlab-pages";
          version = "1.3.8";
          src = ./.;
          modules = ./gomod2nix.toml;
          doCheck = false;
        };
        defaultPackage = packages.gitlab-pages;
        apps.gitlab-pages = utils.lib.mkApp { drv = packages.gitlab-pages; };
        defaultApp = apps.gitlab-pages;
      });
}
