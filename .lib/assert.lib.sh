# vim: set filetype=sh tabstop=2 shiftwidth=2 expandtab :
# shellcheck shell=sh
"${sourced_8f5a035-false}" && return 0; sourced_8f5a035=true

# set -- "$PWD" "$@"; if test "${2:+$2}" = _LIBDIR; then cd "$3" || exit 1; fi
# set -- _LIBDIR . "$@"
# shift 2
# cd "$1" || exit 1; shift

# Assertion functions

assert_eq() {
  local message="${3:-}"
  OPTIND=1; while getopts _-:m: OPT
  do
    test "$OPT" = - && OPT="${OPTARG%%=*}" && OPTARG="${OPTARG#"$OPT"=}"
    case "$OPT" in
      (m|message) message="$OPTARG";;
      (*) echo "Unexpected option: $OPT" >&2; exit 1;;
    esac
  done
  shift $((OPTIND-1))

  test "$1" = "$2" && return 0
  test "$1" -eq "$2" && return 0
  printf "Equality assertion failed%s\n" "${message:+ ($message)}"
  printf "  LHS: %s\n" "$1"
  printf "  RHS: %s\n" "$2"
  return 1
}

assert_neq() {
  local message="${3:-}"
  OPTIND=1; while getopts _-:m: OPT
  do
    test "$OPT" = - && OPT="${OPTARG%%=*}" && OPTARG="${OPTARG#"$OPT"=}"
    case "$OPT" in
      (m|message) message="$OPTARG";;
      (*) echo "Unexpected option: $OPT" >&2; exit 1;;
    esac
  done
  shift $((OPTIND-1))

  test "$1" = "$2" || return 0
  printf "Inequality assertion failed%s\n" "${message:+ ($message)}"
  printf "  LHS: %s\n" "$1"
  printf "  RHS: %s\n" "$2"
  return 1
}

assert() {
  local message=
  OPTIND=1; while getopts _-:m: OPT
  do
    test "$OPT" = - && OPT="${OPTARG%%=*}" && OPTARG="${OPTARG#"$OPT"=}"
    case "$OPT" in
      (m|message) message="$OPTARG";;
      (*) echo "Unexpected option: $OPT" >&2; exit 1;;
    esac
  done
  shift $((OPTIND-1))

  "$@" && return 0
  printf "Failed: \"%s\" is not true%s\n" "$*" "${message:+ ($message)}"
  return 1
}

assert_true() {
  local message=
  OPTIND=1; while getopts _-:m: OPT
  do
    test "$OPT" = - && OPT="${OPTARG%%=*}" && OPTARG="${OPTARG#"$OPT"=}"
    case "$OPT" in
      (m|message) message="$OPTARG";;
      (*) echo "Unexpected option: $OPT" >&2; exit 1;;
    esac
  done
  shift $((OPTIND-1))

  "$@" && return 0
  printf "Failed: \"%s\" is not true%s\n" "$*" "${message:+ ($message)}"
  return 1
}

assert_false() {
  local message=
  OPTIND=1; while getopts _-:m: OPT
  do
    test "$OPT" = - && OPT="${OPTARG%%=*}" && OPTARG="${OPTARG#"$OPT"=}"
    case "$OPT" in
      (m|message) message="$OPTARG";;
      (*) echo "Unexpected option: $OPT" >&2; exit 1;;
    esac
  done
  shift $((OPTIND-1))

  "$@" || return 0
  printf "Failed: \"%s\" is not false%s\n" "$*" "${message:+ ($message)}"
  return 1
}

# assert_match <expected> <actual>
assert_match() {
  local message="${3:-}"
  OPTIND=1; while getopts _-:m: OPT
  do
    test "$OPT" = - && OPT="${OPTARG%%=*}" && OPTARG="${OPTARG#"$OPT"=}"
    case "$OPT" in
      (m|message) message="$OPTARG";;
      (*) echo "Unexpected option: $OPT" >&2; exit 1;;
    esac
  done
  shift $((OPTIND-1))

  echo "$2" | grep -E -q "$1" && return 0
  printf "Failed: \"%s\" does not match \"%s\"%s\n" "$2" "$1" "${message:+ ($message)}"
  return 1
}
