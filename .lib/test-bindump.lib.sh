# vim: set filetype=sh tabstop=2 shiftwidth=2 expandtab :
# shellcheck shell=sh
"${sourced_54bac15-false}" && return 0; sourced_54bac15=true

set -- "$PWD" "$@"; if test "${2:+$2}" = _LIBDIR; then cd "$3" || exit 1; fi
set -- _LIBDIR . "$@"
. ./bindump.lib.sh
. ./assert.lib.sh
shift 2
cd "$1" || exit 1; shift


test_octdump() {
  local plain_text="hello"
  local oct_text="150 145 154 154 157 "
  local oct_text_lf="150 145 154 154 157 012 "

  local dumped_text

  dumped_text="$(echo "$plain_text" | oct_dump)"
  assert_eq "$oct_text_lf" "$dumped_text" "d3fd480"
  assert_eq "$plain_text" "$(echo "$dumped_text" | oct_restore)" "8f74313"

  dumped_text="$(oct_dump "$plain_text")"
  assert_eq "$oct_text" "$dumped_text" "d30c86c"
  assert_eq "$plain_text" "$(oct_restore "$dumped_text")" "41747c7"
}

test_hexdump() {
  local plain_text="hello"
  local hex_text="68 65 6c 6c 6f "
  local hex_text_lf="68 65 6c 6c 6f 0a "

  local dumped_text

  dumped_text="$(echo "$plain_text" | hex_dump)"
  assert_eq "$hex_text_lf" "$dumped_text" "d905a11"
  assert_eq "$plain_text" "$(echo "$dumped_text" | hex_restore)" "d1c1d02"

  dumped_text="$(hex_dump "$plain_text")"
  assert_eq "$hex_text" "$dumped_text" "7b663d6"
  assert_eq "$plain_text" "$(hex_restore "$dumped_text")" "e08570f"

  local utf8_text="こんにちは、ごきげんいかがですか？"
  assert_eq "$utf8_text" "$(hex_restore "$(hex_dump "$utf8_text")")" "d1ced86"
}
