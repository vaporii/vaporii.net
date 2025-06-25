{ pkgs ? (import <nixpkgs> {}), ... }: let
  vaporiiPkg = pkgs.buildGoModule {
    pname = "vaporii.net";
    version = "0.0.1";

    src = ../.;

    vendorHash = "sha256-AAAKAJHNEFJSB";

    # installPhase = ''
    #   mkdir -p $out/bin/v8p.me
    #   cp -r build $out/bin/v8p.me
    #   cp -r node_modules $out/bin/v8p.me
    # '';
  };
in
pkgs.dockerTools.buildImage {
  name = "vaporii.net";
  tag = "latest";

  config = {
    Cmd = [ "${vaporiiPkg}/bin/vaporii.net" ];
  };

#   copyToRoot = pkgs.buildEnv {
#     name = "v8p.me";
#     paths = [ v8pPkg pkgs.coreutils pkgs.nodejs_22 ];
#     pathsToLink = [ "/bin" "/lib" ];
#   };

#   keepContentsDirlinks = true;
}
