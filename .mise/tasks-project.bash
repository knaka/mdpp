#!/usr/bin/env bash
set -- _d5c7489 "$@"; eval "shift; \${$1-false} || ! $1=true" && return # shpp:source_guard

pushd "${BASH_SOURCE[0]%[/\\]*}" &>/dev/null || pushd . >/dev/null
. ../.lib/utils.sh
  init_temp_dir
popd >/dev/null || exit

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
  local a_out="$TEMP_DIR/a.out$EXE_EXT"
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
