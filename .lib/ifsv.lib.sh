# vim: set filetype=sh tabstop=2 shiftwidth=2 expandtab :
# shellcheck shell=sh
test "${sourced_89a99a9-}" = true && return 0; sourced_89a99a9=true

set -- "$PWD" "$@"; if test "${2:+$2}" = _LIBDIR; then cd "$3" || exit 1; fi
set -- _LIBDIR . "$@"
. ./utils.lib.sh
shift 2
cd "$1" || exit 1; shift

# --------------------------------------------------------------------------
# IFS-separated value functions.
# --------------------------------------------------------------------------

# Head of IFSV.
ifsv_head() {
  test $# -eq 0 && return 1
  # shellcheck disable=SC2086
  set -- $1
  printf "%s" "$1"
}

# Tail of IFSV.
ifsv_tail() {
  test $# -eq 0 && return 1
  # shellcheck disable=SC2086
  set -- $1
  shift
  local delim="${IFS%"${IFS#?}"}"
  printf "%s$delim" "$@"
}

ifsv_length() {
  # shellcheck disable=SC2086
  set -- $1
  echo "$#"
}

ifsv_empty() {
  test -z "$1"
}

# Join IFS-separated values with the specified delimiter.
ifsv_join() {
  local out_delim="$2"
  # shellcheck disable=SC2086
  set -- $1
  printf "%s$out_delim" "$@"
}

# Get an item at a specified index in 0-based. If the 3rd argument is provided, it is used as a replacement for the item and returns the new IFSV.
ifsv_at() {
  local index="$2"
  if test "${3+set}" = set
  then
    local new_val="$3"
  fi
  # shellcheck disable=SC2086
  set -- $1
  if ! test "${new_val+set}" = set
  then
    shift "$index"
    printf "%s" "$1"
    return
  fi
  local delim="${IFS%"${IFS#?}"}"
  while test "$index" -gt 0
  do
    printf "%s$delim" "$1"
    index=$((index - 1))
    shift
  done
  shift
  printf "%s$delim" "$new_val" "$@"
}

apply_cmd_to_values() {
  while test "$1" != "--"
  do
    local substituted=false
    local arg1="$1"
    shift
    local arg
    for arg in "$@"
    do
      if ! "$substituted" && test "$arg" = "_"
      then
        arg="$arg1"
        substituted=true
      fi
      set -- "$@" "$arg"
      shift
    done
    if ! $substituted
    then
      set -- "$@" "$arg1"
    fi
  done
  shift
  "$@"
}

# Map IFS-separated values with a command. If the command contains argument "_", then it is replaced with the item.
ifsv_map() {
  local vec="$1"
  shift
  local delim="${IFS%"${IFS#?}"}"
  local elem
  for elem in $vec
  do   
    printf "%s$delim" "$(apply_cmd_to_values "$elem" -- "$@")"
  done
}

# Filter IFS-separated values with a command. If the command contains argument "_", then it is replaced with the item.
ifsv_filter() {
  local vec="$1"
  shift
  local delim="${IFS%"${IFS#?}"}"
  local elem
  for elem in $vec
  do
    if apply_cmd_to_values "$elem" -- "$@" >/dev/null 2>&1
    then
      printf "%s%s" "$elem" "$delim"
    fi
  done
}

# Reduce IFS-separated values with a function. If the function contains two "_", then it is replaced with the accumulator and the item.
ifsv_reduce() {
  local vec="$1"
  shift
  local acc="$1"
  shift
  local elem
  for elem in $vec
  do
    acc="$(apply_cmd_to_values "$acc" "$elem" -- "$@")"
  done
  echo "$acc"
}

# Check if an IFS-separated value contains a specified item.
ifsv_contains() {
  local target="$2"
  # shellcheck disable=SC2086
  set -- $1
  while test $# -gt 0
  do
    test "$1" = "$target" && return 0
    shift
  done
  return 1
}

# Sort IFS-separated values.
ifsv_sort() {
  local vec="$1"
  shift
  test -z "$vec" && return
  local lines; lines="$(
    # shellcheck disable=SC2086
    printf "%s\n" $vec \
    | if test "$#" -eq 0
      then
        sort
      else
        "$@"
      fi
  )"
  local saved_ifs="$IFS"; IFS="$newline_char"
  # shellcheck disable=SC2086
  set -- $lines
  IFS="$saved_ifs"
  local delim="${IFS%"${IFS#?}"}"
  printf "%s$delim" "$@"
}
