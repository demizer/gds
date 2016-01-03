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
        plain "DEBUG: ${1}"
    fi
}

export RUN_CMD_RETURN=0

run_cmd() {
    # $1: The command to run
    if [[ $DRY_RUN -eq 1 ]]; then
        # for pos in $@; do
        msg "CMD: $@"
        # done
    else
        plain "Running command: $@"
        # eval "$@"
        echo "$@" | source /dev/stdin
        RUN_CMD_RETURN=$?
        plain "Command returned: $RUN_CMD_RETURN"
    fi
}

cleanup() {
    exit $1 || true
}

abort() {
	msg 'Aborting...'
	cleanup 1
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
