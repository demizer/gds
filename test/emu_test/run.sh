#!/bin/bash

#
# DO NOT USE "bash -e"... if the main gds command fails, the panic output will not be recorded!
#

# lp="test/log/$(date --iso-8601=seconds).log"
lp="test/log/output.log"
ctxl="test/log/$(date --iso-8601=seconds)_context.json"
conf="test/emu_test/config.yml"

ALL_OFF="$(tput sgr0 2> /dev/null)"
BOLD="$(tput bold 2> /dev/null)"
RED="${BOLD}$(tput setaf 1 2> /dev/null)"

error() {
	local mesg=$1; shift
	printf "${RED}====  ERROR: ${ALL_OFF}${BOLD}${WHITE}${mesg}${ALL_OFF}\n" "$@" >&2
}

if [[ "$(basename ${PWD})" != "gds" ]]; then
    echo "Please run from toplevel gds project directory"
    exit 1;
fi

# Prepare the mount points
if [[ ! -n "$1" ]]; then
    # Unmount any mounted drives
    ./test/scripts/mktestfs.sh umount emu_test

    # Create the devices if they do not exist
    if [[ ! -f "${HOME}/.config/gds/test/gds-test-0" ]]; then
        if ! ./test/scripts/mktestfs.sh make emu_test; then
            exit 1
        fi
    fi

    if ! ./test/scripts/mktestfs.sh mount emu_test; then
        exit 1
    fi
    if ! ./test/scripts/mktestfs.sh wipe emu_test; then
        exit 1
    fi
fi

# Build the f'n thing
gb build -q
if [[ $? == 0 ]]; then
    if ! ./bin/gds -c "${conf}" --log "${lp}" --context "${ctxl}" --log-level debug sync 2> /tmp/gds-error; then
        reset
        echo -e "\n" >> "${lp}"
        cat /tmp/gds-error >> "${lp}"
        error "A fatal error has occurred. See '${lp}' for more details."
    fi
    rm -rf src/cmd/gds/gds 2> /dev/null
fi
