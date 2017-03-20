# futa.world

A text adventure engine written in Go, created for an erotic text adventure at [futa.world](http://futa.world) via web and telnet.

### Usage

`config.json` contains the basic configuration for the server, `game.json` contains the game world in its entirety.

Use the command-line arguments `-config=path/to/file` and `-game=path/to/file` to manually specify files, otherwise it will simply look in the working directory for `config.json` and `game.json`.

The included `game.json` is identical to the one hosted at [futa.world](http://futa.world).

### Operation

Unless disabled in `config.json`, the server will launch both a telnet server and an HTTP server which allow connection to the game.