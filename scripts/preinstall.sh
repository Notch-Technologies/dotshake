#!/bin/sh
sudo apt-get update && apt-get install -y \
  net-tools \
  network-manager \
  && apt-get clean