# vim: set filetype=sh tabstop=2 shiftwidth=2 expandtab :
# shellcheck shell=sh
"${sourced_cba2d9f-false}" && return 0; sourced_cba2d9f=true

set -- "$PWD" "$@"; if test "${2:+$2}" = _LIBDIR; then cd "$3" || exit 1; fi
set -- _LIBDIR . "$@"
. ./utils.lib.sh
. ./assert.lib.sh
shift 2
cd "$1" || exit 1; shift

test_abs2rel() {
  local relpath
  if is_windows
  then
    relpath="$(abs2rel "$PWD/sh" "$PWD/go")"
    assert_eq "$relpath" "C:../sh"
    relpath="$(abs2rel C:/Windows/System32)"
    assert_match -m "bf12b50" '^[A-Z]:\.\.' "$relpath"
    assert test -d "$relpath"
    relpath="$(abs2rel C:/Windows/System32 C:/Windows)"
    assert_eq "C:System32" "$relpath"
    relpath="$(abs2rel C:/Windows C:/Windows/System32)"
    assert_eq "C:.." "$relpath"
    relpath="$(abs2rel /Windows /Windows/System32)"
    assert_eq ".." "$relpath"
    relpath="$(abs2rel /Windows/System32 /Windows/System)"
    assert_eq "../System32" "$relpath"
    relpath="$(abs2rel C:/Windows/System32 C:/Windows/System32)"
    assert_eq -m "e057121" "C:." "$relpath"
    relpath="$(abs2rel C:/ C:/Windows/System32)"
    assert_eq -m "6724104" "C:../.." "$relpath"
    relpath="$(abs2rel C:/Windows/System32 D:/Somewhere/Foo/Bar)"
    assert_eq -m "349094b" "C:/Windows/System32" "$relpath"
  else
    relpath="$(abs2rel "$PWD/sh" "$PWD/go")"
    assert_eq "$relpath" "../sh"
    relpath="$(abs2rel /usr/bin)"
    assert_match -m "bf12b50" '\.\.' "$relpath"
    assert test -d "$relpath"
    relpath="$(abs2rel /usr/bin /usr)"
    assert_eq "bin" "$relpath"
    relpath="$(abs2rel /usr /usr/bin)"
    assert_eq ".." "$relpath"
    relpath="$(abs2rel /usr /usr/lib)"
    assert_eq ".." "$relpath"
    relpath="$(abs2rel /usr/bin /usr/lib)"
    assert_eq "../bin" "$relpath"
    relpath="$(abs2rel /usr/bin /usr/bin)"
    assert_eq -m "e057121" "." "$relpath"
    relpath="$(abs2rel / /usr/bin)"
    assert_eq -m "6724104" "../.." "$relpath"
  fi
}
