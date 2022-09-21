#!/bin/sh
curl -L https://pkgs.dotshake.com/debian/pgp-key.public | sudo apt-key add -
curl -L https://pkgs.dotshake.com/debian/dotshake.list | sudo tee /etc/apt/sources.list.d/dotshake.list
