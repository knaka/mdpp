# vim: set filetype=sh tabstop=2 shiftwidth=2 expandtab :
# shellcheck shell=sh
"${sourced_eaf97e4-false}" && return 0; sourced_eaf97e4=true

set -- "$PWD" "$@"; if test "${2:+$2}" = _LIBDIR; then cd "$3" || exit 1; fi
set -- _LIBDIR . "$@"
. ./utils.lib.sh
. ./tools.lib.sh
script_902b082="$(canon_path ./embed.py)"
shift 2
cd "$1" || exit 1; shift

# This function is tested, do not inlined.
embed_minified_sub() {
  python -u "$script_902b082" "$1"
}

# Embeds minified file contents into shell script files in-place.
# Processes files containing #EMBED directives and replaces the content
# between single quotes with the minified contents of the referenced file.
# Files are only updated if the content actually changes.
embed_minified() {
  if test $# = 0
  then
    echo "Usage: embed_minified <file>..." >&2
    return 1
  fi

  register_temp_cleanup
  local path
  local temp_path="$TEMP_DIR/2163b17"
  for path in "$@"
  do
    embed_minified_sub "$path" >"$temp_path" 
    if test -s "$temp_path" && cmp -s "$path" "$temp_path"
    then
      echo "\"$path\" is up to date." >&2
    else
      cat "$temp_path" >"$path"
      echo "Wrote \"$path\"." >&2
    fi
  done
}
