{
  description = "Flake utils demo";

  inputs.flake-utils.url = "github:numtide/flake-utils";

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let pkgs = nixpkgs.legacyPackages.${system}; in
      {
        devShells.default = pkgs.mkShellNoCC {
          nativeBuildInputs = with pkgs; [
            go
          ] ++ (with llvmPackages_16; [
            bintools-unwrapped
            clang
          ]);
        };
      }
    );
}
