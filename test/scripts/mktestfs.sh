#!/bin/bash -e
#
# A script for setting up manual tests
#
# Requires setting user mountable paths in /etc/fstab:
#
# /home/demizer/.config/gds/test/gds-test-dev-0       /mnt/gds-test-0         ext4 noauto,defaults,user 0 0
# /home/demizer/.config/gds/test/gds-test-dev-1       /mnt/gds-test-1         ext4 noauto,defaults,user 0 0
# /home/demizer/.config/gds/test/gds-test-dev-2       /mnt/gds-test-2         ext4 noauto,defaults,user 0 0
# /home/demizer/.config/gds/test/gds-test-dev-3       /mnt/gds-test-3         ext4 noauto,defaults,user 0 0
# /home/demizer/.config/gds/test/gds-test-dev-4       /mnt/gds-test-4         ext4 noauto,defaults,user 0 0
# /home/demizer/.config/gds/test/gds-test-dev-5       /mnt/gds-test-5         ext4 noauto,defaults,user 0 0
# /home/demizer/.config/gds/test/gds-test-dev-6       /mnt/gds-test-6         ext4 noauto,defaults,user 0 0
# /home/demizer/.config/gds/test/gds-test-dev-7       /mnt/gds-test-7         ext4 noauto,defaults,user 0 0
# /home/demizer/.config/gds/test/gds-test-dev-8       /mnt/gds-test-8         ext4 noauto,defaults,user 0 0
# /home/demizer/.config/gds/test/gds-test-dev-9       /mnt/gds-test-9         ext4 noauto,defaults,user 0 0
#


#
# START CONFIG SECTION
#

DEVICE_DESTPATH="/home/demizer/.config/gds/test"

# Manual test using my emulator directory
# 57489123330 is the catalog size of the backup, extra is added for FS
# overhead
EMU_DATASIZE_BYTES=65489123330
EMU_NUM_DEVICES=8
EMU_DEVICE_NAME_PREFIX="gds-emu-test"
EMU_DEV_UUID[0]="51f5a503-f670-46a5-8098-59fa69af6fed"
EMU_DEV_UUID[1]="40b46262-96dc-4fb2-a765-a0c948794305"
EMU_DEV_UUID[2]="79d583e5-dab9-40ad-ad98-253ae0fde964"
EMU_DEV_UUID[3]="50a38e6c-2bfd-4cb1-8741-78632304a8fe"
EMU_DEV_UUID[4]="beb2cd83-72e3-40d9-ae7f-d73f3897366a"
EMU_DEV_UUID[5]="2a52ef55-be52-4e40-803b-39f0a350695e"
EMU_DEV_UUID[6]="5a6cfaa6-7827-4222-b6e7-a29c38e5ebe9"
EMU_DEV_UUID[7]="808312f2-e11d-49e5-a46f-d4dd18f6514e"

#
# END CONFIG SECTION
#

shopt -s nullglob

MKT_MAKE=0
MKT_WIPE=0
MKT_MOUNT=0
MKT_UMOUNT=0
MKT_EMU_TEST=0
DEBUG=0
DRY_RUN=0

# check if messages are to be printed using color
unset ALL_OFF BOLD BLUE GREEN RED YELLOW WHITE

# prefer terminal safe colored and bold text when tput is supported
ALL_OFF="$(tput sgr0 2> /dev/null)"
BOLD="$(tput bold 2> /dev/null)"
BLUE="${BOLD}$(tput setaf 4 2> /dev/null)"
GREEN="${BOLD}$(tput setaf 2 2> /dev/null)"
RED="${BOLD}$(tput setaf 1 2> /dev/null)"
YELLOW="${BOLD}$(tput setaf 3 2> /dev/null)"
WHITE="${BOLD}$(tput setaf 7 2> /dev/null)"
readonly ALL_OFF BOLD BLUE GREEN RED YELLOW

plain() {
	local mesg=$1; shift
	printf "${WHITE}     ○ ${ALL_OFF}${BOLD}${mesg}${ALL_OFF}\n" "$@"
}

msg() {
	local mesg=$1; shift
	printf "${GREEN}====${ALL_OFF}${WHITE}${BOLD} ${mesg}${ALL_OFF}\n" "$@"
}

msg2() {
	local mesg=$1; shift
	printf "${BLUE}++++ ${ALL_OFF}${WHITE}${BOLD}${mesg}${ALL_OFF}\n" "$@"
}

warning() {
	local mesg=$1; shift
	printf "${YELLOW}==== WARNING: ${ALL_OFF}${WHITE}${BOLD} ${mesg}${ALL_OFF}\n" "$@"
}

error() {
	local mesg=$1; shift
	printf "${RED}==== ERROR: ${ALL_OFF}${BOLD}${WHITE}${mesg}${ALL_OFF}\n" "$@" >&2
}

debug() {
    # $1: The message to print.
    if [[ $DEBUG -eq 1 ]]; then
        plain "DEBUG: $1"
    fi
}

run_cmd() {
    # $1: The command to run
    if [[ $DRY_RUN -eq 1 ]]; then
        # for pos in $@; do
        plain "$@"
        # done
    else
        plain "Running command: $@"
        eval "$@"
        plain "Command returned: $?"
    fi
}

cleanup() {
	# [[ -n $WORKDIR ]] && rm -rf "$WORKDIR"
	[[ $1 ]] && exit $1
    exit 0
}

abort() {
	msg 'Aborting...'
	cleanup 0
}

trap_abort() {
	trap - EXIT INT QUIT TERM HUP
	abort
}

trap_exit() {
	trap - EXIT INT QUIT TERM HUP
	cleanup
}

die() {
	(( $# )) && error "$@"
	cleanup 1
}

trap 'trap_abort' INT QUIT TERM HUP
trap 'trap_exit' EXIT

NAME=$(basename $0)

usage() {
    echo "$NAME - gds test device management tool"
    echo
	echo "Usage: $NAME [options] <command> <test>"
    echo
    echo "Options:"
    echo
    echo "    -h:    Show help information."
    echo "    -n:    Dryrun; Output commands, but don't do anything."
    echo "    -d:    Show debug info."
    echo
    echo "Commands:"
    echo
    echo "    make       Make the test devices."
    echo "    wipe       Wipe the test devices."
    echo "    mount      Mount the test devices."
    echo "    umount     Unmount the test devices."
    echo
    echo "Tests:"
    echo
    echo "    emu_test   Create the test devices needed for testing using my emulation files (50GiB in Size)."
    echo
	echo "Examples:"
    echo
    echo "    $NAME make_emu_test :: Create the test devices."
}

if [[ $# -lt 1 ]]; then
    usage;
    exit 0;
fi

ARGS=("$@")
for (( a = 0; a < $#; a++ )); do
    if [[ ${ARGS[$a]} == "make" ]]; then
        MKT_MAKE=1
    elif [[ ${ARGS[$a]} == "wipe" ]]; then
        MKT_WIPE=1
    elif [[ ${ARGS[$a]} == "mount" ]]; then
        MKT_MOUNT=1
    elif [[ ${ARGS[$a]} == "umount" ]]; then
        MKT_UMOUNT=1
    elif [[ ${ARGS[$a]} == "emu_test" ]]; then
        MKT_EMU_TEST=1
    elif [[ ${ARGS[$a]} == "-n" ]]; then
        DRY_RUN=1
    elif [[ ${ARGS[$a]} == "-d" ]]; then
        DEBUG=1
    elif [[ ${ARGS[$a]} == "-h" ]]; then
        usage;
        exit 0;
    fi
done

function format_ext4() {
    # $1 - Device file path
    # $2 - UUID of device
    msg2 "Formatting ext4..."
    run_cmd "mkfs.ext4 -F -U $2 $1"
}

function make_devices() {
    # $1 - Number of devices
    # $2 - Destination path
    # $3 - Name prefix
    # $4 - Disk size in blocks
    # $5 - Block size (for dd)
    # $6 - Disk size in bytes
    if [[ "$3" == "gds-emu-test" ]]; then
        arry=(${EMU_DEV_UUID[@]})
    fi
    for (( x = 0; x < $1; x++)); do
        OF="$2/$3-$x"
        msg "Setting '$OF' to all zeroes"
        dd if=/dev/zero count=$4 bs=$5 2> /dev/null | pv -prb -s $6 | dd of="$OF" 2> /dev/null
        format_ext4 "$OF" "${arry[$x]}"
    done
}

function mount_devices() {
    # $1 - Number of devices
    # $2 - Device name prefix
    # Clear out old symlinks
    run_cmd "find $DEVICE_DESTPATH -iname 'gds-test-dev*' -type l -exec rm {} \;"
    for (( x = 0; x < $1; x++)); do
        symName="gds-test-dev-$x"
        msg2 "Creating symlink for $symName"
        run_cmd "ln -s $DEVICE_DESTPATH/$2-$x $DEVICE_DESTPATH/$symName"
        mnt="gds-test-$x"
        msg2 "Mounting $mnt"
        run_cmd "mount /mnt/$mnt"
    done
}

function umount_devices() {
    # $1 - Number of devices
    for (( x = 0; x < $1; x++)); do
        mnt="gds-test-$x"
        if [[ "$(mountpoint /mnt/$mnt; echo $?)" == 0 ]]; then
            msg2 "Un-mounting $mnt"
            run_cmd "umount /mnt/$mnt"
        fi
    done
    # Clear out old symlinks
    run_cmd "find $DEVICE_DESTPATH -iname 'gds-test-dev*' -type l -exec rm {} \;"
}

function make_devices_writable() {
    # $1 - Number of devices
    for (( x = 0; x < $1; x++)); do
        mnt="gds-test-$x"
        msg2 "chgrp users for $mnt"
        run_cmd "sudo chgrp -R users /mnt/$mnt"
        run_cmd "sudo chmod -R g+w /mnt/$mnt"
    done
}

function wipe_devices() {
    # $1 - Number of devices
    # $2 - Device name prefix
    if [[ "$2" == "gds-emu-test" ]]; then
        arry=(${EMU_DEV_UUID[@]})
    fi
    for (( x = 0; x < $1; x++)); do
        dev="$DEVICE_DESTPATH/$2-$x"
        format_ext4 "$dev" "${arry[$x]}"
    done
}

[[ ! -d $DEVICE_DESTPATH ]] && mkdir -p $DEVICE_DESTPATH

if [[ "$MKT_EMU_TEST" == 1 ]]; then
    if [[ "$MKT_MAKE" == 1 ]]; then
        DISKSIZE=$(($EMU_DATASIZE_BYTES/$EMU_NUM_DEVICES))
        DISKSIZE_IN_BLOCKS=$(($DISKSIZE/4096))
        msg "Creating $EMU_NUM_DEVICES devices!"
        make_devices $EMU_NUM_DEVICES $DEVICE_DESTPATH $EMU_DEVICE_NAME_PREFIX $DISKSIZE_IN_BLOCKS "4k" $DISKSIZE
    fi
    if [[ "$MKT_MOUNT" == 1 ]]; then
        msg "Mounting emulation backup devices"
        mount_devices $EMU_NUM_DEVICES $EMU_DEVICE_NAME_PREFIX
        gown=$(stat '/mnt/gds-test-0' | grep 'Gid:' | awk '{print $10}' | grep -o '[[:alnum:]]*')
        debug "gown: $gown"
        if [[ "$gown" != "users" ]]; then
            msg "Setting write permissions"
            make_devices_writable $EMU_NUM_DEVICES
        fi
    elif [[ "$MKT_UMOUNT" == 1 || "$MKT_WIPE" == 1 ]]; then
        msg "Un-mounting emulation backup devices"
        umount_devices $EMU_NUM_DEVICES $EMU_DEVICE_NAME_PREFIX
        if [[ "$MKT_WIPE" == 1 ]]; then
            wipe_devices $EMU_NUM_DEVICES $EMU_DEVICE_NAME_PREFIX
        fi
    fi
fi
