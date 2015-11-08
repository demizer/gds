#!/bin/bash

if [[ "$(basename ${PWD})" != "gds" ]]; then
    echo "Please run from toplevel gds project directory"
    exit 1;
fi

# lp="test/log/$(date --iso-8601=seconds).log"
lp="test/log/output.log"
ctxl="test/log/$(date --iso-8601=seconds)_context.json"
conf="test/emu_test/config.yml"

# Prepare the mount points
if [[ ! -n "$1" ]]; then
    if [[ ! -f "${HOME}/.config/gds/test/gds-test-0" ]]; then
        test/scripts/mktestfs.sh make emu_test
    fi
    test/scripts/mktestfs.sh wipe emu_test
    test/scripts/mktestfs.sh mount emu_test
fi

ALL_OFF="$(tput sgr0 2> /dev/null)"
BOLD="$(tput bold 2> /dev/null)"
RED="${BOLD}$(tput setaf 1 2> /dev/null)"

error() {
	local mesg=$1; shift
	printf "${RED}====  ERROR: ${ALL_OFF}${BOLD}${WHITE}${mesg}${ALL_OFF}\n" "$@" >&2
}

# Build the f'n thing
gb build -q
if [[ $? == 0 ]]; then
    ./bin/gds -c "${conf}" --log "${lp}" --context "${ctxl}" --log-level debug sync 2> /tmp/gds-error
    if [[ "$(echo $?)" > 0 ]]; then
        reset
        echo -e "\n" >> "${lp}"
        cat /tmp/gds-error >> "${lp}"
        error "A fatal error has occurred. See '${lp}' for more details."
    fi
    rm -rf src/cmd/gds/gds 2> /dev/null
fi
