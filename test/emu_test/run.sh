#!/bin/bash

if [[ "$(basename $PWD)" != "gds" ]]; then
    echo "Please run from toplevel gds project directory"
    exit 1;
fi

# lp="test/log/$(date --iso-8601=seconds).log"
lp="test/log/output.log"
ctxl="test/log/$(date --iso-8601=seconds)_context.json"
conf="test/emu_test/config.yml"

# Prepare the mount points
./test/scripts/mktestfs.sh wipe emu_test
./test/scripts/mktestfs.sh mount emu_test

# Build the f'n thing
gb build -q
if [[ $? == 0 ]]; then
    ./bin/gds -c "$conf" --log "$lp" --context "$ctxl" --log-level debug sync
    rm -rf src/cmd/gds/gds 2> /dev/null
fi
