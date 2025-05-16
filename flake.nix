{
  description = "Bowtie Terraform provider development environment";

  inputs = {
    devshell.url = "github:numtide/devshell";
    flake-compat.url = "https://flakehub.com/f/edolstra/flake-compat/1.tar.gz";
    flake-parts.url = "github:hercules-ci/flake-parts";
    flake-utils.url = "github:numtide/flake-utils";
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
  };

  outputs = inputs@{ self, devshell, flake-parts, flake-utils, nixpkgs, ... }:
    flake-parts.lib.mkFlake { inherit inputs; } {
      imports = [ devshell.flakeModule ];
      systems = flake-utils.lib.defaultSystems;

      perSystem = { pkgs, system, ... }: {
        _module.args.pkgs = import nixpkgs {
          inherit system;
          config.allowUnfree = true;
	  overlays = [
	    (final: prev: {
	      devshellPython = prev.python3.withPackages (p: [
		p.requests
		p.pyyaml
	      ]);
	    })
	  ];
        };

        devshells.default = {
          imports = [ (devshell.lib.importTOML ./devshell.toml) ];
        };
      };
    };
}
