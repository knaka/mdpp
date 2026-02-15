# vim: set filetype=sh tabstop=2 shiftwidth=2 expandtab :
# shellcheck shell=sh
"${sourced_23969e5-false}" && return 0; sourced_23969e5=true

set -- "$PWD" "$@"; if test "${2:+$2}" = _LIBDIR; then cd "$3" || exit 1; fi
set -- _LIBDIR . "$@"
. ./assert.lib.sh
. ./json2sh.lib.sh
shift 2
cd "$1" || exit 1; shift



json2sh_expected() {
  cat <<EOF
json__user__name="Alice"
json__user__age="30"
json__items__0="apple"
json__items__1="banana"
EOF
}

test_json2sh() {
  local expected="$TEMP_DIR/390f638.sh"
  json2sh_expected >"$expected"

  local actual="$TEMP_DIR/d06580e.sh"
  echo '{"user":{"name":"Alice","age":30},"items":["apple","banana"]}' | json2sh >"$actual"

# cat -n "$expected"
# cat "$expected" | od

# cat -n "$actual"
# cat "$actual" | od

  assert_eq -m "4a3762e" \
    "$(sha256sum "$expected" | field 1)" \
    "$(sha256sum "$actual" | field 1)"
}
