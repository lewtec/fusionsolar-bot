{
  description = "Bot to fetch fusionsolar production and send to a list of emails";

  inputs = {
    nixpkgs.url = "nixpkgs";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { nixpkgs, flake-utils, self, ... }:
  flake-utils.lib.eachDefaultSystem (system: let
    pkgs = import nixpkgs { inherit system; };
  in {
    packages = rec {
      default = pkgs.python3Packages.callPackage ./package.nix {};
      container = pkgs.python3Packages.callPackage ./container.nix { inherit self default; };
      docker-deploy = let
        inherit (self.packages.${system}) container;
        version = "$(cat version.txt)";
      in pkgs.writeShellScriptBin "docker-deploy" ''
        ${container} | docker load

        # Tag with different versions
        for tag in ${container.tag} ${version} latest; do
          docker tag ${container.name}:${container.tag} ${container.name}:$tag
          docker push ${container.name}:$tag
        done
      '';
    };
    nixosModules.default = import ./nixos.nix;
  });
}
