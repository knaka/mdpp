# vim: set filetype=sh tabstop=2 shiftwidth=2 expandtab :
# shellcheck shell=sh
"${sourced_737f6db-false}" && return 0; sourced_737f6db=true

# -A n: input address base - no address
# -t o1: output format type - octal, 1 byte size
# -v: write all instead of replacing lines of duplicate values with a ‘*’
oct_dump() {
  if test $# -eq 0
  then
    cat
  else
    printf "%s" "$@"
  fi \
  | od -A n -t o1 -v \
  | xargs printf "%s "
}

oct_restore() {
  if test $# -eq 0
  then
    cat
  else
    printf "%s" "$@"
  fi \
  | xargs printf '\\\\0%s\n' \
  | xargs printf '%b'
}

# -A n: input address base - no address
# -t o1: output format type - hexadecimal, 1 byte size
# -v: write all instead of replacing lines of duplicate values with a ‘*’
hex_dump() {
  if test $# -eq 0
  then
    cat
  else
    printf "%s" "$@"
  fi \
  | od -A n -t x1 -v \
  | xargs printf "%s "
}

hex_restore() {
  awk "BEGIN { printf \"$(
    if test $# -eq 0
    then
      cat
    else
      printf "%s" "$@"
    fi \
    | xargs printf "\\\x%s"
  )\" }"
}
