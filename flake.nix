{
  description = "Navidrome is an open source web-based music collection server and streamer.";
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  };
  outputs =
    {
      self,
      nixpkgs,
    }:
    let
      systems = [
        "x86_64-linux"
        "aarch64-linux"
        "x86_64-darwin"
        "aarch64-darwin"
      ];
      forAllSystems = nixpkgs.lib.genAttrs systems;
      pkgsForEach = system: nixpkgs.legacyPackages.${system};
    in
    {
      devShells = forAllSystems (
        system:
        let
          pkgs = pkgsForEach system;
          goModVersion = builtins.head (
            builtins.match ".*\ngo ([0-9]+\\.[0-9]+).*" (builtins.readFile ./go.mod)
          );
          goAttr = "go_${builtins.replaceStrings [ "." ] [ "_" ] goModVersion}";
          nvmrcVersion = builtins.head (builtins.match ".([0-9]+).*" (builtins.readFile ./.nvmrc));
          nodeAttr = "nodejs_${nvmrcVersion}";
        in
        {
          default = pkgs.mkShell {
            name = "navidrome-shell";
            packages = with pkgs; [
              pkgs.${goAttr}
              gopls
              pkgs.${nodeAttr}
              gnumake
              perl
            ];
            shellHook = ''
              export PATH="$HOME/go/bin:$PATH"
              echo "Navidrome dev shell ready"
            '';
          };
        }
      );
    };
}
