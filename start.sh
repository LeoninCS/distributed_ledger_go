#!/bin/bash

set -euo pipefail

DIR="$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
CONFIG_PATH="${1:-config/config.yml}"

cd "$DIR"

mkdir -p bin
go build -o bin/node ./cmd

./bin/node -config "$CONFIG_PATH"

