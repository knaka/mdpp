# vim: set filetype=sh tabstop=2 shiftwidth=2 expandtab :
# shellcheck shell=sh
"${sourced_7ecf25d-false}" && return 0; sourced_7ecf25d=true

set -- "$PWD" "${0%/*}" "$@"; test -z "${_APPDIR-}" && { test "$2" = "$0" && _APPDIR=. || _APPDIR="$2"; cd "$_APPDIR" || exit 1; }
set -- _LIBDIR .lib "$@"
. ./.lib/utils.lib.sh
  register_temp_cleanup
  register_child_cleanup
. ./.lib/tools.lib.sh
shift 2
cd "$1" || exit 1; shift 2

# Run Go tests.
task_test() {
  if test $# = 0
  then
    set -- ./...
  fi
  go test "$@"
}

# Run application with debug information.
task_run() {
  local package=./cmd/mdpp/
  local a_out="$TEMP_DIR/a.out$exe_ext"
  go build -gcflags='all=-N -l' -tags=debug,nop -o "$a_out" "$package"
  "$a_out" "$@"
}

# Update documentation files.
task_doc() {
  mdpp --in-place --allow-remote \
    DEVELOPMENT.md \
    CLAUDE.md \
    #nop
}

case "${0##*/}" in
  (tasks-*)
    set -o nounset -o errexit
    "$@"
    ;;
esac
