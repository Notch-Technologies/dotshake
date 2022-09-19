# dotshake

## env
TODO: (shinta) preparing enviroment variables

## for install
TODO: (shinta) preparing how install for dotshake command
### linux

### darwin

### freebsd

### windows

## for build without docker
TODO: (shinta) preparing arguments for each commands
check the command line arguments carefully!

### server
start the server without docker
`sudo go run cmd/server/server.go`

### signaling
start the signaling without docker
`go run cmd/signaling/signaling.go`

### dotshake
start the dotshake without docker
`sudo go run cmd/dotshake/up.go --key <your setup key>`

## for build with docker
start the server using docker-compose
`make up-server`
if you want to build
`make build-server`

start the signaling server using docker-compose
`make up-signaling`
if you want to build
`make build-signaling`

start the dotshake using docker-compose
`make up-dotshake`
if you want to build
`make build-dotshake`

## for development
if you want develop server
`cmd/server` is the server that manages the peer's information.

if you want develop signaling server
`cmd/signaling` is the server to negotiate peer.

if you want develop dotshake
`cmd/dotshake` is the connects to peer clients, signaling servers and servers, and performs peer communication.

## for NixOS
TODO: Instead of using make, use flake to create the dotshake binary. This will make development much easier since the build binaries can be placed directly in the store without using the nix-store command.

if you want to run it in a daemon,
1. run `make store` to create a dotshake build binary in store
2. add the store path as follows and rebuild nixos

```
  { pkgs, ... }:

  let
  in {
    nixpkgs.overlays = [(self: super: {
      dotshake = pkgs.writeScriptBin "dotshake" ''
        #! ${pkgs.stdenv.shell} -e
        exec </your store binary> up // place your own built binaries in the store
      '';
    })];
  
    # for development dotshake
    systemd.services.dotshake = {
      description = "dotshake daemon";
      wants  = [ "network-online.target" "systemd-networkd-wait-online.service"];
      after = [ "network-online.target" ];
      path = [ pkgs.iproute ];
      serviceConfig = {
        User = "root";
        Type = "simple";
        ExecStart = "${pkgs.dotshake}/bin/dotshake";
        Restart = "on-failure";
        RestartSec = "15";
      };
      wantedBy = [ "multi-user.target" ];
    };
  }

```

enjoy the daemons.
