# vim: set filetype=sh tabstop=2 shiftwidth=2 expandtab :
# shellcheck shell=sh
"${sourced_7ecf25d-false}" && return 0; sourced_7ecf25d=true

. ./task.sh
. ./go.lib.sh

# Run Go tests.
subcmd_test() {
  go test "$@"
}

# Update documentation files.
task_doc() {
  mdpp --in-place --allow-remote \
    DEVELOPMENT.md \
    CLAUDE.md \
    #nop
}
