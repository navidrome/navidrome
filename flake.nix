{
  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs =
    {
      nixpkgs,
      flake-utils,
      ...
    }:
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in
      {
        devShells.default = pkgs.mkShell {

          env.CGO_ENABLED = 1;
          buildInputs = with pkgs; [
            nodejs
            pkg-config
            taglib
            zlib

            zip # needed for tests
            sqlite # to check the db
          ];
        };
      }
    );
}
