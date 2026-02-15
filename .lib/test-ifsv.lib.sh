# vim: set filetype=sh tabstop=2 shiftwidth=2 expandtab :
# shellcheck shell=sh
test "${sourced_78b9c2d-}" = true && return 0; sourced_78b9c2d=true

set -- "$PWD" "$@"; if test "${2:+$2}" = _LIBDIR; then cd "$3" || exit 1; fi
set -- _LIBDIR . "$@"
. ./utils.lib.sh
. ./ifsv.lib.sh
shift 2
cd "$1" || exit 1; shift

tolower_542075d() {
  printf "%s" "$1" | tr '[:upper:]' '[:lower:]'
}

sum_75e35a9() {
  local sum=0
  while test $# -gt 0
  do
    sum=$((sum + $1))
    shift
  done
  echo "$sum"
}

test_ifsv_basic() (
  set -o errexit

  assert_eq "FOO" "$(apply_cmd_to_values "foo" -- toupper_6201a5f)"
  assert_eq "FOO" "$(apply_cmd_to_values "foo" -- toupper_6201a5f _)"
  assert_eq 5 "$(apply_cmd_to_values 2 3 -- sum_75e35a9 _ _)"
  assert_eq 10 "$(apply_cmd_to_values 1 2 3 4 -- sum_75e35a9)"

  assert_eq "foo" "$(IFS=, ifsv_head "foo,bar,baz,")"
  assert_eq "bar,baz," "$(IFS=, ifsv_tail "foo,bar,baz,")"
  
  assert_eq "3" "$(IFS=, ifsv_length "foo,bar,baz,")"

  assert_eq "bar" "$(IFS=, ifsv_at "foo,bar,baz," 1)"
  assert_eq "foo,qux,baz," "$(IFS=, ifsv_at "foo,bar,baz," 1 "qux")"
)

test_ifsv_join() (
  set -o errexit

  assert_eq "foo|bar|baz|" "$(IFS=, ifsv_join "foo,bar,baz," "|")"
)

strlen_e5775ea() {
  echo "${#1}"
}

toupper_6201a5f() {
  printf "%s" "$1" | tr '[:lower:]' '[:upper:]'
}

test_ifsv_map() (
  set -o errexit

  # Procedural
  vec="foo,bar,baz,"
  new_vec=
  saved_ifs="$IFS"; IFS=","
  for elem in $vec
  do
    new_vec="$new_vec$(toupper_6201a5f "$elem"),"
  done
  IFS="$saved_ifs"
  assert_eq "FOO,BAR,BAZ," "$new_vec"

  # Functional
  assert_eq "FOO,BAR,BAZ," "$(IFS=, ifsv_map "foo,bar,baz," toupper_6201a5f)"
  assert_eq "FOO,BAR,BAZ," "$(IFS=, ifsv_map "foo,bar,baz," toupper_6201a5f _)"
  assert_eq "foo,bar,baz," "$(IFS=, ifsv_map "FOO,BAR,BAZ," tolower_542075d)"
  assert_eq "foo,bar,baz," "$(IFS=, ifsv_map "FOO,BAR,BAZ," tolower_542075d _)"

  assert_eq "5,3,7," "$(IFS=, ifsv_map "Alice,Bob,Charlie" strlen_e5775ea)"
)

test_ifsv_filter() (
  set -o errexit

  assert_eq "foo,bar,baz," "$(IFS=, ifsv_filter "foo,bar,,baz," test -n)"
  assert_eq "foo,bar,baz," "$(IFS=, ifsv_filter "foo,bar,,baz," test -n _)"
  assert_eq "4,5,6,7," "$(IFS=, ifsv_filter "1,2,3,4,5,6,7," test _ -gt 3)"
)

test_ifsv_reduce() (
  set -o errexit

  # shellcheck disable=SC2317
  add() (
    echo $(( $1 + $2 ))
  )

  assert_eq 10 "$(IFS=, ifsv_reduce "1,2,3,4" 0 add)"

  # shellcheck disable=SC1102
  # shellcheck disable=SC2005
  # shellcheck disable=SC2086
  # shellcheck disable=SC2046
  # shellcheck disable=SC2317
  rpn() { echo $(($1 $3 $2)); }
  assert_eq 10 "$(IFS="|" ifsv_reduce "4|3|2|1" 0 rpn _ _ '+')"
  assert_eq 24 "$(IFS="|" ifsv_reduce "4|3|2|1" 1 rpn _ _ '*')"
)

test_default_ifs() (
  set -o errexit
  map=
  map="$(ifsm_put "$map" "foo" "FOO")"
  assert_eq "FOO" "$(ifsm_get "$map" "foo")"
)

# Test plist functions.
test_plist() (
  set -o errexit

  IFS=,
  csvpl=
  csvpl="$(ifsm_put "$csvpl" "key1" "val1")"
  csvpl="$(ifsm_put "$csvpl" "key2" "val2")"

  assert_eq "key1,key2," "$(ifsm_keys "$csvpl")"
  assert_eq "" "$(ifsm_keys "")"

  assert_eq "val1,val2," "$(ifsm_values "$csvpl")"
  assert_eq "" "$(ifsm_values "")"

  assert_eq "val2" "$(ifsm_get "$csvpl" "key2")"
  assert_false ifsm_get "$csvpl" "key3"

  assert_eq "key2,val2,key1,mod1," "$(ifsm_put "$csvpl" "key1" "mod1")"
  assert_eq "key1,val1,key2,val2,key3,val3," "$(ifsm_put "$csvpl" "key3" "val3")"

  assert_eq "key1,val1,key2,," "$(ifsm_put "$csvpl" "key2" "")"
  assert_eq "" "$(ifsm_get "key1,val1,key2,," "key2")"

  assert_eq "key1,val1,key2,val2,,empty," "$(ifsm_put "$csvpl" "" "empty")"
  assert_eq "empty" "$(ifsm_get "key1,val1,key2,val2,,empty" "")"

  IFS="$us"
  usvpl=
  usvpl=$(ifsm_put "$usvpl" "foo bar" "FOO BAR")
  usvpl=$(ifsm_put "$usvpl" "baz qux" "BAZ QUX")
  assert_eq "foo bar${us}FOO BAR${us}baz qux${us}BAZ QUX${us}" "$usvpl"
  assert_eq "BAZ QUX" "$(ifsm_get "$usvpl" "baz qux")"
)
