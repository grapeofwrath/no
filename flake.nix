{
  description = "NO";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  };

  outputs = {
    self,
    nixpkgs,
  }: let
    allSystems = [
      "x86_64-linux"
      "aarch64-linux"
      "x86_64-darwin"
      "aarch64-darwin"
    ];
    forAllSystems = f:
      nixpkgs.lib.genAttrs allSystems (system:
        f {
          pkgs = import nixpkgs {inherit system;};
        });
  in {
    packages = forAllSystems ({pkgs}: {
      default = pkgs.buildGoModule rec {
        pname = "no";
        version = "0.1.0";

        src = ./.;

        vendorHash = null;

        meta = {
          description = "A NixOS and Home Manager CLI helper, written in Go";
          homepage = "https:github.com/grapeofwrath/no";
          license = pkgs.lib.licenses.gpl3Plus;
        };
      };
    });
  };
}
