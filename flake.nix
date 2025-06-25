{
  description = "vaporii.net flake";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs?ref=nixos-unstable";
  };

  outputs = { self, nixpkgs }@inputs:
  let
    system = "x86_64-linux";
    pkgs = nixpkgs.legacyPackages.${system};
  in {
    packages.${system}.default = (import ./nix { inherit pkgs; });

    nixosModules.default = self.nixosModules.vaporii;
    nixosModules.vaporii = import ./nix/nixos.nix inputs;

    devShells.x86_64-linux.default = pkgs.mkShell {
      packages = with pkgs; [
        go
      ];
    };
  };
}