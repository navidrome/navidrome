{ pkgs ? (import <nixpkgs> {})}:
pkgs.mkShell {
  pname   = "navidrome";
  version = "dev";

  hardeningDisable = [ "all" ];

  nativeBuildInputs = with pkgs; [
    go
    pkg-config
    nodejs
  ];

  buildInputs = with pkgs; [
    taglib
    zlib
  ];
}
