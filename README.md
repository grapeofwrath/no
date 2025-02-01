<h2 align="center">
    <picture>
        <img src="./assets/no.png" style="margin-left: auto; margin-right: auto">
    </picture>
</h2>

_<p align="center">a NixOS and Home Manager CLI helper, written in Go</p>_

---

This is what could be as simple as command aliases in my shell that I have
decided to wrap up in a go module. Is this necessary? I think the name speaks
for itself. However, it does tickle my fancy.

```shell
$ no --help
no is a NixOS and Home Manager CLI helper written in Go.

Usage:
    no [flags] <command> [command flags]

Commands:
    garbage  Run garbage collection and remove old generations
    home     Rebuild a Home Manager configuration
    rebuild  Rebuild a NixOS configuration
    update   Update a flake.lock file
    help     Print this help

Flags:
    -d, --directory  PATH
        Run in this directory, must be full path. (default '.')
    -h, --help
        Print this help.

Run `no <command> -h` to get help for a specific command
```

## no demo

```sh
nix run github:grapeofwrath/no
```

## no install

1. Add this flake to your inputs.

2. Add the package to your system or user packages list.

3. Profit.

```nix
# flake.nix

{
  inputs.no = {
    url = "github:grapeofwrath/no";
    inputs.nixpkgs.follows = "nixpkgs";
  };

  ...
}

# configuration.nix

{ inputs, system, ... }: {
  environment.systemPackages = [ inputs.no.packages.${system}.default ];
}
```
