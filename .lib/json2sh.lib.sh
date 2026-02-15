# vim: set filetype=sh tabstop=2 shiftwidth=2 expandtab :
# shellcheck shell=sh
"${sourced_a5dd01a-false}" && return 0; sourced_a5dd01a=true

set -- "$PWD" "$@"; if test "${2:+$2}" = _LIBDIR; then cd "$3" || exit 1; fi
set -- _LIBDIR . "$@"
. ./tools.lib.sh
shift 2
cd "$1" || exit 1; shift

# Convert JSON object or array to shell variable assignment expressions.
# Usage: json2sh [--prefix=PREFIX] [--local]
# Options:
#   --prefix=PREFIX  Set variable name prefix (default: "json__")
#   --local          Add "local" declaration to variables
# Examples:
#   echo '{"foo":{"bar":"baz"}}' | json2sh --prefix="config__" --local
#   # Outputs: local config__foo__bar="baz"
#   echo '["a","b","c"]' | json2sh --prefix="list__"
#   # Outputs: list__0="a"
#   #          list__1="b" 
#   #          list__2="c"
json2sh() {
  local prefix="json__"
  local local_decl=""
  local delim="__"
  OPTIND=1; while getopts _-: OPT
  do
    if test "$OPT" = "-"
    then
      OPT="${OPTARG%%=*}"
      # shellcheck disable=SC2030
      OPTARG="${OPTARG#"$OPT"}"
      OPTARG="${OPTARG#=}"
    fi
    case "$OPT" in
      (local) local_decl="local ";;
      (prefix) prefix="$OPTARG";;
      (delim|delimiter|separator) delim="$OPTARG";;
      (\?) exit 1;;
      (*) echo "Unexpected option: $OPT" >&2; exit 1;;
    esac
  done
  shift $((OPTIND-1))

  # shellcheck disable=SC2016
  local scr='def to_sh(prefix): to_entries[] | $ARGS.named.delim // "__" as $delim | $ARGS.named.local_decl // "" as $local_decl | ( if .key | type == "number" then  .key | tostring else .key | gsub("[-\\.]"; "_") end ) as $shell_key | if .value | type == "object" or type == "array" then .value | to_sh("\(prefix)\($shell_key)\($delim)") else "\($local_decl)\(prefix)\($shell_key)=\"\(.value)\"" end;.| $ARGS.named.prefix // "json__" as $prefix| to_sh($prefix)' #EMBED: ./json2sh.jq
  jq -r "$scr" \
    --arg prefix "$prefix" \
    --arg local_decl "$local_decl" \
    --arg delim "$delim" \
    #nop
}

# Convert JSON object to shell variable assignment expressions.
subcmd_json2sh() {
  json2sh "$@"
}
