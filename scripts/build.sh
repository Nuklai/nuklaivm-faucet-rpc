#!/usr/bin/env bash
# Copyright (C) 2024, AllianceBlock. All rights reserved.
# See the file LICENSE for licensing terms.

set -o errexit
set -o nounset
set -o pipefail

# Set the CGO flags to use the portable version of BLST
#
# We use "export" here instead of just setting a bash variable because we need
# to pass this flag to all child processes spawned by the shell.
export CGO_CFLAGS="-O -D__BLST_PORTABLE__" CGO_ENABLED=1

# Root directory
ROOT_PATH=$(
    cd "$(dirname "${BASH_SOURCE[0]}")"
    cd .. && pwd
)

# Set default binary directory location
FAUCET_PATH=$ROOT_PATH/build/nuklai-faucet

echo "Building nuklai-faucet in $FAUCET_PATH"
mkdir -p "$(dirname "$FAUCET_PATH")"
go build -o "$FAUCET_PATH" ./