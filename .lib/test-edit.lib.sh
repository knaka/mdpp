# vim: set filetype=sh tabstop=2 shiftwidth=2 expandtab :
# shellcheck shell=sh
"${sourced_c28ce41-false}" && return 0; sourced_c28ce41=true

. ./utils.lib.sh
. ./edit.lib.sh
. ./assert.lib.sh

hello_sh_7dad95b=./testdata/hello.sh
hello_txt_e48f9dc=./testdata/hello.txt

test_edit() {
  local function_text
  function_text="$(extract_block "^hello()" "^}" "$hello_sh_7dad95b")"
  assert_eq "$(cat <<EOF
hello() {
  echo Hello
}
EOF
)" "$function_text"

  local function_line_num
  function_line_num="$(echo "$function_text" | wc -l)"
  assert test 3 -eq "$function_line_num"

  local count_before
  count_before="$(wc -l <"$hello_sh_7dad95b")"
  local count_after
  count_after="$(exclude_block "^hello()" "^}" "$hello_sh_7dad95b" | wc -l)"
  assert test $((count_before - function_line_num)) -eq "$count_after"

  assert test 4 -eq "$(extract_before 881e6d7 "$hello_txt_e48f9dc" | wc -l)"
  assert test 5 -eq "$(extract_after 881e6d7 "$hello_txt_e48f9dc" | wc -l)"
}
