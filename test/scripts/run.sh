#!/bin/bash

#
# DO NOT USE "bash -e"... if the main gds command fails, the panic output will not be recorded!
#

# lp="test/log/$(date --iso-8601=seconds).log"
lp="test/log/output.log"
ctxl="test/log/$(date --iso-8601=seconds)_context.json"
CONF=""

#
# END CONFIG SECTION
#

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

source ${SCRIPT_DIR}/lib.sh

trap 'trap_abort' INT QUIT TERM HUP
trap 'trap_exit' EXIT

NAME=$(basename $0)

RUN_EMULATION=0
RUN_IMAGES=0
RUN_BTRFS_1=0

usage() {
    echo "${NAME} - gds integration test runner"
    echo
	echo "Usage: ${NAME} [options] <test>"
    echo
    echo "Options:"
    echo
    echo "    -h:    Show help information."
    echo
    echo "Tests:"
    echo
    echo "    emulation     Tests using 8 devices and 55GiB of Data."
    echo "    images        Tests using 3 of 8 devices and 17GiB of Data."
    echo "    btrfscomp     btrfs test using two real devices and lzo compression."
    echo
	echo "Examples:"
    echo
    echo "    ${NAME} emulation :: Run 'emulation' integration test."
}

if [[ $# -lt 1 ]]; then
    usage;
    exit 0;
fi

if [[ "$(basename ${PWD})" != "gds" ]]; then
    echo "Please run from toplevel gds project directory"
    exit 1;
fi

ARGS=("$@")
for (( a = 0; a < $#; a++ )); do
    if [[ ${ARGS[$a]} == "emulation" ]]; then
        RUN_EMULATION=1
    elif [[ ${ARGS[$a]} == "images" ]]; then
        RUN_IMAGES=1
    elif [[ ${ARGS[$a]} == "btrfscomp" ]]; then
        RUN_BTRFS_1=1
    elif [[ ${ARGS[$a]} == "-h" ]]; then
        usage;
        exit 0;
    fi
done

prepare_devices_emu_img() {
    # Unmount any mounted drives
    ${SCRIPT_DIR}/mktestfs.sh umount emulation

    # Create the devices if they do not exist
    if [[ ! -f "${HOME}/.config/gds/test/gds-test-0" ]]; then
        if ! ${SCRIPT_DIR}/mktestfs.sh make emulation; then
            exit 1
        fi
    fi

    if ! ${SCRIPT_DIR}/mktestfs.sh mount emulation; then
        exit 1
    fi
    if ! ${SCRIPT_DIR}/mktestfs.sh wipe emulation; then
        exit 1
    fi
    # if ! ${SCRIPT_DIR}/mktestfs.sh umount emulation; then
        # exit 1
    # fi
}

prepare_devices_btrfs_compressed() {
    if ! ${SCRIPT_DIR}/mktestfs.sh umount btrfscomp; then
        exit 1
    fi
    if ! ${SCRIPT_DIR}/mktestfs.sh mount btrfscomp; then
        exit 1
    fi
    if ! ${SCRIPT_DIR}/mktestfs.sh wipe btrfscomp; then
        exit 1
    fi
    # if ! ${SCRIPT_DIR}/mktestfs.sh umount btrfscomp; then
        # exit 1
    # fi
}

run() {
    # $1 configuration file
    # Build the f'n thing
    gb build -q
    if [[ $? == 0 ]]; then
        START=$(date +%s.%N)
        if ! ./bin/gds -c "${1}" --log "${lp}" --context "${ctxl}" --log-level debug sync 2> /tmp/gds-error; then
            reset
            echo -e "\n" >> "${lp}"
            cat /tmp/gds-error >> "${lp}"
            error "A fatal error has occurred. See '${lp}' for more details."
        fi
        END=$(date +%s.%N)
        DIFF=$(echo "${END} - ${START}" | bc)
        msg "Ran in ${DIFF} seconds"
        rm -rf src/cmd/gds/gds 2> /dev/null
    fi
}

if [[ ${RUN_EMULATION} == 1 ]]; then
    prepare_devices_emu_img
    run "test/config/config_emulation.yml"
elif [[ ${RUN_IMAGES} == 1 ]]; then
    prepare_devices_emu_img
    run "test/config/config_images.yml"
elif [[ ${RUN_BTRFS_1} == 1 ]]; then
    prepare_devices_btrfs_compressed
    run "test/config/config_btrfs.yml"
fi
