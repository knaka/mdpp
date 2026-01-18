# vim: set filetype=sh tabstop=2 shiftwidth=2 expandtab :
# shellcheck shell=sh
"${sourced_7ecf25d-false}" && return 0; sourced_7ecf25d=true

. ./task.sh
. ./go.lib.sh

# Run Go tests.
subcmd_test() {
  if test $# = 0
  then
    set -- ./...
  fi
  go test "$@"
}

# Run application with debug information.
subcmd_run() {
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
