#!/bin/bash

set -euo pipefail

DIR="$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
CONFIG_ARG="${1:-configs/node1.yml}"

cd "$DIR"

if [[ -f "$CONFIG_ARG" ]]; then
  CONFIG_PATH="$CONFIG_ARG"
elif [[ -f "configs/${CONFIG_ARG}.yml" ]]; then
  CONFIG_PATH="configs/${CONFIG_ARG}.yml"
else
  echo "???????: $CONFIG_ARG ? configs/${CONFIG_ARG}.yml"
  exit 1
fi

mkdir -p bin
go build -o bin/node ./cmd

echo "????: $CONFIG_PATH"
./bin/node -config "$CONFIG_PATH"
