# vim: set filetype=sh tabstop=2 shiftwidth=2 expandtab :
# shellcheck shell=sh
"${sourced_70ed462-false}" && return 0; sourced_70ed462=true

set -- "$PWD" "$@"; if test "${2:+$2}" = _LIBDIR; then cd "$3" || exit 1; fi
set -- _LIBDIR . "$@"
. ./utils.lib.sh
. ./assert.lib.sh
. ./test.lib.sh
shift 2
cd "$1" || exit 1; shift

test_unconditional_skip() {
  skip
}

func_global_ifs() {
  IFS="$newline_char"
  assert_eq "$newline_char" "$IFS"
  # shellcheck disable=SC2046
  set -- $(printf "foo bar\nbar baz\nhoge fuga\n")
  assert_eq $# 3
}

func_local_ifs() {
  local IFS="$newline_char"
  assert_eq "$newline_char" "$IFS"
  # shellcheck disable=SC2046
  set -- $(printf "foo bar\nbar baz\nhoge fuga\n")
  assert_eq $# 3
}

# Test that IFS with local works and does not affects outer scope one.
test_local_ifs() (
  original_ifs="$IFS"
  
  func_global_ifs
  # IFS should still be changed after function returns
  assert_eq "$newline_char" "$IFS"
  
  # Reset for next test
  IFS="$original_ifs"
  
  # Test that local IFS is restored after function returns
  
  func_local_ifs
  # IFS should be restored to original value
  assert_eq "$original_ifs" "$IFS"

  # shellcheck disable=SC2046
  set -- $(printf "foo bar\nbar baz\nhoge fuga\n")
  assert_eq $# 6
)

test_pos_params() {
  set -- "aaa  bbb" "ccc"
  assert_eq 2 $#

  local IFS
  IFS="$newline_char"

  # shellcheck disable=SC2046
  set -- $(printf "x%s\n" "$@")
  assert_eq 2 $#
  assert_eq "xaaa  bbb" "$1"
  assert_eq "xccc" "$2"

  # shellcheck disable=SC2046
  set -- $(printf '%s' '["foo   bar", "baz"]' | jq -r '.[]')
  assert_eq -m "b0dafd8" 2 $#
  assert_eq "foo   bar" "$1"
  assert_eq "baz" "$2"

  # shellcheck disable=SC2046
  local count="$(printf "%s\n" $(printf "x%s\n" "$@") | wc -l)"
  assert_eq -m "f52c6b3" 2 "$count"
}
