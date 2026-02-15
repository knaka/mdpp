# vim: set filetype=sh tabstop=2 shiftwidth=2 expandtab :
# shellcheck shell=sh
"${sourced_8e8321e-false}" && return 0; sourced_8e8321e=true

set -- "$PWD" "${0%/*}" "$@"; test -z "${_APPDIR-}" && { test "$2" = "$0" && _APPDIR=. || _APPDIR="$2"; cd "$_APPDIR" || exit 1; }
set -- _LIBDIR .. "$@"
. ./../utils.lib.sh
shift 2
cd "$1" || exit 1; shift 2

py_scr_abbb7c6='print("???")' #EMBED: ./some.py

original3() {
  python -c "$py_scr_abbb7c6"
}

set -o nounset -o errexit
original3 "$@"
