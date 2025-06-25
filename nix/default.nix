{ pkgs ? (import <nixpkgs> {}), ... }: let
  vaporiiPkg = pkgs.buildGoModule {
    pname = "vaporii.net";
    version = "0.0.1";

    src = ../.;

    vendorHash = "sha256-iAqEgyzxDddu3o9DaZx58tEw00dVgAXb47ZIcFnms6Y=";
  };
in
pkgs.dockerTools.buildImage {
  name = "vaporii.net";
  tag = "latest";

  copyToRoot = pkgs.buildEnv {
    name = "image-root";
    paths = [ vaporiiPkg ];
    pathsToLink = [ "/bin" ];
  };

  config = {
    Cmd = [ "/bin/server" ];
  };
}
