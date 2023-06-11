#!/bin/bash

set -e

if [ "$EUID" -ne 0 ]
  then echo "Please run as root"
  exit
fi

echo "Building CRUDD"

go build .

echo "Installing CRUDD"

cp ./crudd /usr/bin/crudd

echo "Installing CRUDD service"

cp ./crudd.service /etc/systemd/system/crudd.service

echo "Enabling CRUDD service"

systemctl daemon-reload

systemctl enable --now crudd.service

echo "Done"