#!/usr/bin/env sh
# vim: set filetype=sh :
# shellcheck shell=sh
"${sourced_897a0c7-false}" && return 0; sourced_897a0c7=true

# ==========================================================================
#region Constants

# Return code when a test is skipped
# shellcheck disable=SC2034
rc_test_skipped=10

#endregion

# ==========================================================================
#region Environment variables. If not set by the caller, they are set later in `tasksh_main`

# The initial working directory when the script was started.
: "${INITIAL_PWD:=$PWD}"

# The path to the shell executable which is running this script.
: "${SH:=/bin/sh}"

# The path to the file which was called.
: "${ARG0:=}"

# Basename of the file which was called.
: "${ARG0BASE:=}"

# Directory in which the task files are located.
: "${TASKS_DIR:=}"

# The root directory of the project.
: "${PROJECT_DIR:=${0%/*}}"

# Verbosity flag.
: "${VERBOSE:=false}"

# Cache directory path for the task runner
: "${CACHE_DIR:=$HOME/.cache/task-sh}"
mkdir -p "$CACHE_DIR"

# For platforms other than Windows
: "${LOCALAPPDATA:=/}"

#endregion

# ==========================================================================
#region Temporary directory and cleaning up

TEMP_DIR=; unset TEMP_DIR

# Create a temporary directory and assign $TEMP_DIR env var
init_temp_dir() {
  test "${TEMP_DIR+set}" = set && return 0
  TEMP_DIR="$(mktemp -d)"
  # shellcheck disable=SC2064
  trap "rm -fr '$TEMP_DIR'" EXIT
}

readonly stmts_file_id=523f163

# Chain traps to avoid overwriting the previous trap.
chaintrap() {
  local stmts_new="$1"
  shift 
  init_temp_dir || return $?
  # Basename of the script file containing the statements to be called during finalization
  local stmts_file_base="$TEMP_DIR"/"$stmts_file_id"
  local stmts_old_file="$TEMP_DIR"/347803f
  local sigspec
  for sigspec in "$@"
  do
    local stmts_file="$stmts_file_base"-"$sigspec"
    if test -f "$stmts_file"
    then
      cp "$stmts_file" "$stmts_old_file"
    else
      touch "$stmts_old_file"
    fi
    echo "{ $stmts_new; };" >"$stmts_file"
    cat "$stmts_old_file" >>"$stmts_file"
    # shellcheck disable=SC2064 # "Use single quotes, otherwise this expands now rather than when signalled."
    # shellcheck disable=SC2154 # "var is referenced but not assigned."
    trap "rc=\$?; test -r '$stmts_file' && . '$stmts_file'; rm -fr '$TEMP_DIR'; exit \$rc" "$sigspec"
  done
}

# Call the finalization function before `exec` which does not call trap function.
finalize() {
  test "${TEMP_DIR+set}" = set || return 0
  local stmts_file_base="$TEMP_DIR"/"$stmts_file_id"
  local stmts_file="$stmts_file_base"-EXIT
  # shellcheck disable=SC1090 # "Can't follow non-constant source. Use a directive to specify location"
  test -f "$stmts_file" && . "$stmts_file"
  rm -fr "$TEMP_DIR"
}

#endregion

# ==========================================================================
#region Utilities

# Guard against multiple calls. $1 is a unique ID
first_call() {
  eval "\${called_$1-false}" && return 1
  eval "called_$1=true"
}

is_terminal() {
  test -t 1
}

#endregion

# ==========================================================================
#region Platform detection. Detect platform without using subprocesses whenever possible.

is_linux() {
  test -d /proc -o -d /sys
}

is_macos() {
  test -r /System/Library/CoreServices/SystemVersion.plist
}

is_windows() {
  test -d "c:/" -a ! -d /proc
}

# Executable file extension.
exe_ext=
# shellcheck disable=SC2034
is_windows && exe_ext=".exe"

is_debian() {
  test -f /etc/debian_version
}

is_bsd() {
  is_macos || test -r /etc/rc.subr
}

is_alpine() {
  test -f /etc/alpine-release
}

#endregion

# ==========================================================================
#region IFS manipulation

# shellcheck disable=SC2034
readonly unit_sep=""

# Unit separator (US), Information Separator One (0x1F)
# shellcheck disable=SC2034
readonly us=""
# shellcheck disable=SC2034
readonly is="$us"
# shellcheck disable=SC2034
readonly is1="$us"

# Information Separator Two (0x1E)
# shellcheck disable=SC2034
readonly is2=""

# Information Separator Three (0x1D)
# shellcheck disable=SC2034
readonly is3=""

# Information Separator Four (0x1C)
# shellcheck disable=SC2034
readonly is4=""

set_ifs_newline() {
  IFS="$(printf '\n\r')"
}

# shellcheck disable=SC2034
readonly newline_char="
"

# To split paths.
set_ifs_slashes() {
  printf "/\\"
}

set_ifs_default() {
  printf ' \t\n\r'
}

set_ifs_blank() {
  printf ' \t'
}

#endregion

# ==========================================================================
#region Directory stack

psv_dirs_4c15d80=""

# `pushd` alternative.
push_dir() {
  local pwd="$PWD"
  if ! cd "$1"
  then
    echo "Directory does not exist: $1" >&2
    return 1
  fi
  psv_dirs_4c15d80="$pwd|$psv_dirs_4c15d80"
}

# `popd` alternative.
pop_dir() {
  if test -z "$psv_dirs_4c15d80"
  then
    echo "Directory stack is empty" >&2
    return 1
  fi
  local dir="${psv_dirs_4c15d80%%|*}"
  psv_dirs_4c15d80="${psv_dirs_4c15d80#*|}"
  cd "$dir" || return 1
}

#endregion

# ==========================================================================
#region Map (associative array) functions. "IFS-Separated Map"

# Put a value in an associative array implemented as a property list.
ifsm_put() {
  local key="$2"
  local value="$3"
  # shellcheck disable=SC2086
  set -- $1
  # First char of IFS
  local delim="${IFS%"${IFS#?}"}"
  while test $# -gt 0
  do
    test "$1" != "$key" && printf "%s%s%s%s" "$1" "$delim" "$2" "$delim"
    shift 2
  done
  printf "%s%s%s%s" "$key" "$delim" "$value" "$delim"
}

# Get a value from an associative array implemented as a property list.
ifsm_get() {
  local key="$2"
  # shellcheck disable=SC2086
  set -- $1
  while test $# -gt 0
  do
    test "$1" = "$key" && printf "%s" "$2" && return 0
    shift 2
  done
  return 1
}

# Keys of an associative array implemented as a property list.
ifsm_keys() {
  # shellcheck disable=SC2086
  set -- $1
  # First char of IFS
  local delim="${IFS%"${IFS#?}"}"
  while test $# -gt 0
  do
    printf "%s%s" "$1" "$delim"
    shift 2
  done
}

# Values of an associative array implemented as a property list.
ifsm_values() {
  # shellcheck disable=SC2086
  set -- $1
  # First char of IFS
  local delim="${IFS%"${IFS#?}"}"
  while test $# -gt 0
  do
    printf "%s%s" "$2" "$delim"
    shift 2
  done
}

#endregion

# ==========================================================================
#region Fetch and run a command from an archive

windows_exe_extensions=".exe .EXE .cmd .CMD .bat .BAT"

# Invoke external command with proper executable extension, with the specified invocation mode.
#
# Invocation mode can be specified via INVOCATION_MODE environment variable:
#   INVOCATION_MODE=standard: (Default) Run the command in the current process.
#   INVOCATION_MODE=exec: Replace this process with the command.
#   INVOCATION_MODE=exec-direct: Replace this process with the command, without calling cleanups.
#   INVOCATION_MODE=background: Run the command in the background.
#
# Invocation mode can be specified also with `--invocation-mode=...` option. The option is excluded from final arguments which are passed to the external command.
#
# Command-specific invocation mode can be set using INVOCATION_MODE_<command> variables:
#   INVOCATION_MODE_foo=background: Run external command `foo` in the background.
#   The command name is extracted from the basename of the first argument, with Windows
#   executable extensions (.exe, .cmd, etc.) stripped if on Windows platform.
invoke() {
  if test $# -eq 0
  then
    echo "No command specified" >&2
    exit 1
  fi
  local invocation_mode="${INVOCATION_MODE:-standard}"
  local base="${1##*/}"
  if is_windows
  then
    local ext
    for ext in $windows_exe_extensions
    do
      base="${base%"$ext"}"
    done
  fi
  if eval "test \"\${INVOCATION_MODE_$base+set}\" = set"
  then
    eval "invocation_mode=\"\${INVOCATION_MODE_$base}\""
  fi
  local arg
  for arg in "$@"
  do
    case "$arg" in
      (--invocation-mode=*)
        invocation_mode="${arg#--invocation-mode=}"
        ;;
      (*)
        set -- "$@" "$arg"
        ;;
    esac
    shift
  done
  local cmd="$1"
  case "$1" in
    (*/*)
      if is_windows
      then
        local ext
        for ext in $windows_exe_extensions
        do
          if test -x "$cmd$ext"
          then
            shift
            set -- "$cmd$ext" "$@"
            break
          fi
        done
      fi
      if ! test -x "$1"
      then
        echo "Command not found: $1" >&2
        exit 1
      fi
      ;;
    (*)
      if is_windows
      then
        local ext
        for ext in $windows_exe_extensions
        do
          if command -v "$1$ext" >/dev/null 2>&1
          then
            shift
            set -- "$cmd$ext" "$@"
            break
          fi
        done
      fi
      if ! command -v "$1" >/dev/null 2>&1
      then
        echo "Command not found: $1" >&2
        exit 1
      fi
      ;;
  esac
  "$VERBOSE" && echo "Launching $* in mode $invocation_mode, in $PWD." >&2
  case "$invocation_mode" in
    (exec)
      finalize
      # exec executes the external command even if a function with the same name is defined.
      exec "$@"
      ;;
    (exec-direct)
      exec "$@"
      ;;
    (background)
      command "$@" &
      ;;
    (standard)
      command "$@"
      ;;
    (*)
      echo "Unknown invocation mode: $invocation_mode" >&2
      exit 1
      ;;
  esac
}

# Canonicalize `uname -s` result
uname_s() {
  local os_name; os_name="$(uname -s)"
  case "$os_name" in
    (Windows_NT|MINGW*|CYGWIN*) os_name="Windows" ;;
  esac
  echo "$os_name"
}

map_os() {
  ifsm_get "$1" "$(uname_s)"
}

map_arch() {
  ifsm_get "$1" "$(uname -m)"
}

# Fetch and run a command from a remote archive
# Usage: fetch_cmd_run [OPTIONS] -- [COMMAND_ARGS...]
# Options:
#   --name=NAME           Application name. Used as the directory name to store the command.
#   --ver=VERSION         Application version
#   --cmd=COMMAND         Command name to execute. If not specified, the application name is used.
#   --ifs=IFS             IFS to split the os_map and arch_map options. Default: $IFS
#   --os-map=MAP          OS name mapping (IFS-separated key-value pairs)
#   --arch-map=MAP        Architecture name mapping (IFS-separated key-value pairs)
#   --ext=EXTENSION       Archive file extension (e.g., ".zip", ".tar.gz"). Takes precedence over --ext-map.
#   --ext-map=MAP         Archive extension mapping (IFS-separated key-value pairs). Used when --ext is not specified. If neither option is provided, the URL template points directly to a command binary rather than an archive file
#   --url-template=TEMPLATE URL template string to generate the download URL with ${ver}, ${os}, ${arch}, ${ext}, ${exe_ext} (=.exe on Windows) variables
#   --rel-dir-template=TEMPLATE   Relative path template within archive to the directory containing the command (default: ".")
#   --print-dir           Print the directory path where the command is installed instead of executing the command
#   --macos-remove-signature      Remove code signature from the downloaded binary on macOS to bypass security checks
run_fetched_cmd() {
  local name=
  local ver=
  local cmd=
  local ifs=
  local os_map=
  local arch_map=
  local ext=
  local ext_map=
  local url_template=
  local rel_dir_template=.
  local print_dir=false
  local macos_remove_signature=false
  OPTIND=1; while getopts _-: OPT
  do
    test "$OPT" = - && OPT="${OPTARG%%=*}" && OPTARG="${OPTARG#"$OPT"=}"
    case "$OPT" in
      (name) name=$OPTARG;;
      (ver) ver=$OPTARG;;
      (cmd) cmd=$OPTARG;;
      (ifs) ifs=$OPTARG;;
      (os-map) os_map=$OPTARG;;
      (arch-map) arch_map=$OPTARG;;
      (ext) ext=$OPTARG;;
      (ext-map) ext_map=$OPTARG;;
      (url-template) url_template=$OPTARG;;
      (rel-dir-template) rel_dir_template=$OPTARG;;
      (print-dir) print_dir=true;;
      (macos-remove-signature) macos_remove_signature=true;;
      (*) echo "Unexpected option: $OPT" >&2; exit 1;;
    esac
  done
  shift $((OPTIND-1))

  if test -z "$cmd"
  then
    cmd="$name"
  fi
  local app_dir_path="$CACHE_DIR"/"$name"@"$ver"
  mkdir -p "$app_dir_path"
  local cmd_path="$app_dir_path"/"$cmd""$exe_ext"
  if ! command -v "$cmd_path" >/dev/null 2>&1
  then
    local ifs_saved=
    if test -n "$ifs"
    then
      ifs_saved="$IFS"
      IFS="$ifs"
    fi
    local ver="$ver"
    local os
    # shellcheck disable=SC2034
    os="$(map_os "$os_map")" || return $?
    local arch
    # shellcheck disable=SC2034
    arch="$(map_arch "$arch_map")" || return $?
    if test -z "$ext" -a -n "$ext_map"
    then
      ext="$(map_os "$ext_map")"
    fi
    if test -n "$ifs_saved"
    then
      IFS="$ifs_saved"
    fi
    local url; url="$(eval echo "$url_template")" || return $?
    init_temp_dir
    local out_file_path="$TEMP_DIR"/"$name""$ext"
    if ! curl --fail --location "$url" --output "$out_file_path"
    then
      echo "Failed to download: $url" >&2
      return 1
    fi
    local work_dir_path="$TEMP_DIR"/"$name"ec85463
    mkdir -p "$work_dir_path"
    push_dir "$work_dir_path"
    case "$ext" in
      (.zip) unzip "$out_file_path" ;;
      (.tar.gz) tar -xf "$out_file_path" ;;
      (*) ;;
    esac
    pop_dir
    if test -n "$ext"
    then
      local rel_dir_path; rel_dir_path="$(eval echo "$rel_dir_template")"
      mv "$work_dir_path"/"$rel_dir_path"/* "$app_dir_path"
    else
      mv "$out_file_path" "$cmd_path"
    fi
    chmod +x "$cmd_path"
    if is_macos && "$macos_remove_signature"
    then
      codesign --remove-signature "$cmd_path"
    fi
  fi
  if "$print_dir"
  then
    echo "$app_dir_path"
  else
    PATH="$app_dir_path":$PATH invoke "$cmd_path" "$@"
  fi
}

# Uname kernel name -> GOOS mapping
# Installing Go from source - The Go Programming Language https://go.dev/doc/install/source#environment
# shellcheck disable=SC2140
# shellcheck disable=SC2034
goos_map=\
"Linux linux "\
"Darwin darwin "\
"Windows windows "\
""

# Uname kernel name -> GOOS in CamelCase mapping
# shellcheck disable=SC2140
# shellcheck disable=SC2034
goos_camel_map=\
"Linux Linux "\
"Darwin Darwin "\
"Windows Windows "\
""

# Uname architecture name -> GOARCH mapping
# shellcheck disable=SC2140
# shellcheck disable=SC2034
goarch_map=\
"x86_64 amd64 "\
"arm64 arm64 "\
"armv7l arm "\
"i386 386 "\
""

# Uname kernel name -> generally used archive file extension mapping
# shellcheck disable=SC2140
# shellcheck disable=SC2034
archive_ext_map=\
"Linux .tar.gz "\
"Darwin .tar.gz "\
"Windows .zip "\
""

#endregion

# ==========================================================================
#region Package command management

# Map: command name -> Homebrew package ID
usm_brew_ids=

# Map: command name -> WinGet package ID
usm_winget_ids=

# Map: command name -> Debian package ID
usm_deb_ids=

# Map: command name -> pipe-separated vector of commands
usm_psv_cmds=

# Register a command with optional package IDs for various package managers.
# This function maps a command name to package IDs for installation via different package managers.
# The remaining arguments are treated as the command paths to be tried in order. The last argument is treated as the command name.
# Options:
#   --brew-id=<id>    Package ID for Homebrew (macOS)
#   --deb-id=<id>     Package ID for Debian/Ubuntu package manager
#   --winget-id=<id>  Package ID for Windows Package Manager
require_pkg_cmd() {
  local name=
  local brew_id=
  local deb_id=
  local winget_id=
  OPTIND=1; while getopts _-: OPT
  do
    test "$OPT" = - && OPT="${OPTARG%%=*}" && OPTARG="${OPTARG#"$OPT"=}"
    case "$OPT" in
      (name) name="$OPTARG";;
      (brew-id) brew_id=$OPTARG;;
      (deb-id) deb_id=$OPTARG;;
      (winget-id) winget_id=$OPTARG;;
      (\?) exit 1;;
      (*) echo "Unexpected option: $OPT" >&2; exit 1;;
    esac
  done
  shift $((OPTIND-1))

  # Last argument is treated as the command name.
  local cmd_name=
  local psv_cmds=
  local cmd
  for cmd in "$@"
  do
    cmd_name="$cmd"
    psv_cmds="$psv_cmds$cmd|"
  done
  if test -n "$name"
  then
    cmd_name="$name"
  fi
  test -n "$brew_id" && usm_brew_ids="$usm_brew_ids$cmd_name$us$brew_id$us"
  test -n "$winget_id" && usm_winget_ids="$usm_winget_ids$cmd_name$us$winget_id$us"
  test -n "$deb_id" && usm_deb_ids="$usm_deb_ids$cmd_name$us$deb_id$us"
  usm_psv_cmds="$usm_psv_cmds$cmd_name$us$psv_cmds$us"
}

# Run registered, package-provided command. If the command is not found, print the instructions to install it.
run_pkg_cmd() {
  local cmd_name="$1"
  shift
  local saved_ifs="$IFS"; IFS="|"
  local cmd
  for cmd in $(IFS="$us" ifsm_get "$usm_psv_cmds" "$cmd_name")
  do
    if which "$cmd" >/dev/null
    then
      IFS="$saved_ifs"
      invoke "$cmd" "$@"
      return $?
    fi
  done
  IFS="$saved_ifs"
  echo "Command not found: $cmd_name." >&2
  echo >&2
  if is_macos
  then
    echo "Run the \"devinstall\" task or the following command to install necessary packages for this development environment:" >&2
    echo >&2
    printf "  brew install" >&2
    local saved_ifs="$IFS"; IFS="$us"
    # shellcheck disable=SC2046
    # shellcheck disable=SC2086
    printf " %s" $(ifsm_values "$usm_brew_ids") >&2
    IFS="$saved_ifs"
  elif is_windows
  then
    printf "  winget install" >&2
    local saved_ifs="$IFS"; IFS="$us"
    # shellcheck disable=SC2046
    # shellcheck disable=SC2086
    printf " %s" $(ifsm_values "$usm_winget_ids") >&2
    IFS="$saved_ifs"
  fi
  echo >&2
  echo >&2
  return 1
}

# Install necessary packages for this development environment.
task_devinstall() {
  if is_macos
  then
    set - brew install
    local saved_ifs="$IFS"; IFS="$us"
    # shellcheck disable=SC2046
    set -- "$@" $(ifsm_values "$usm_brew_ids")
    IFS="$saved_ifs"
    invoke "$@"
  elif is_windows
  then
    set - winget install
    local saved_ifs="$IFS"; IFS="$us"
    # shellcheck disable=SC2046
    set -- "$@" $(ifsm_values "$usm_winget_ids")
    IFS="$saved_ifs"
    invoke "$@"
  fi
}

#endregion

# ==========================================================================
#region curl(1) // curl https://curl.se/

# curl(1) is available on macOS and Windows as default.
require_pkg_cmd \
  --deb-id=curl \
  curl

curl() {
  run_pkg_cmd curl "$@"
}

# Run curl(1).
subcmd_curl() {
  curl "$@"
}

#endregion

# ==========================================================================
#region jq(1) // jqlang/jq: Command-line JSON processor https://github.com/jqlang/jq

jq_prefer_pkg_ec51165=false

# Make use of jq(1) which is installed by platform-specific package manager rather than fetched binary.
jq_prefer_pkg() {
  jq_prefer_pkg_ec51165=true
  require_pkg_cmd \
    --brew-id=jq \
    --winget-id=jqlang.jq \
    /usr/local/bin/jq \
    "$LOCALAPPDATA"/Microsoft/WinGet/Links/jq.exe \
    jq
}

# Releases · jqlang/jq · GitHub https://github.com/jqlang/jq/releases
jq_version_6d4ce66=1.8.1

set_jq_version() {
  jq_version_6d4ce66="$1"
}

jq() {
  if is_windows
  then
    set -- --binary "$@"
  fi
  if "$jq_prefer_pkg_ec51165"
  then
    run_pkg_cmd jq "$@"
    return 0
  fi
  # shellcheck disable=SC2016
  run_fetched_cmd \
    --name="jq" \
    --ver="$jq_version_6d4ce66" \
    --os-map="Darwin macos $goos_map" \
    --arch-map="$goarch_map" \
    --url-template='https://github.com/jqlang/jq/releases/download/jq-$ver/jq-$os-$arch$exe_ext' \
    -- \
    "$@"
}

# Run jq(1).
subcmd_jq() {
  jq "$@"
}

#endregion

# ==========================================================================
#region .env* file management

# Load environment variables from the specified file.
load_env_file() {
  if ! test -r "$1"
  then
    return 0
  fi
  local line
  local key
  local value
  while read -r line
  do
    key="${line%%=*}"
    if test -z "$key" || test "$key" = "$line"
    then
      continue
    fi
    value="$(eval "echo \"\${$key:=}\"")"
    # Not to overwrite the existing, previously set value.
    if test -n "$value"
    then
      continue
    fi
    eval "$line"
  done <"$1"
}

# Load environment variables from .env* files
load_env() {
  first_call 8005f70 || return 0
  # Load the files in the order of priority.
  if test "${APP_ENV+set}" = set
  then
    load_env_file "$PROJECT_DIR"/.env."$APP_ENV".session
    load_env_file "$PROJECT_DIR"/.env."$APP_ENV".local
  fi
  if test "${APP_ENV+set}" != set || test "${APP_ENV}" != "test"
  then
    load_env_file "$PROJECT_DIR"/.env.session
    load_env_file "$PROJECT_DIR"/.env.local
  fi
  if test "${APP_ENV+set}" = set
  then
    load_env_file "$PROJECT_DIR"/.env."$APP_ENV"
  fi
  # shellcheck disable=SC1091
  load_env_file "$PROJECT_DIR"/.env
}

#endregion

# ==========================================================================
#region Misc

# Wait for one or more servers to respond with HTTP 200. Checks each URL sequentially with a 60-second timeout per URL.
wait_for_server() {
  local url
  local max_attempts=60
  for url in "$@"
  do
    echo "Waiting for server at $url to be ready ..." >&2
    local attempts=0
    while :
    do
      if curl -s -o /dev/null -w "%{http_code}" "$url" 2>/dev/null | grep -q "200"
      then
        echo "✓ Server is ready at $url" >&2
        break
      fi
      attempts=$((attempts + 1))
      if test $attempts -ge $max_attempts
      then
        echo "✗ Server at $url did not respond with 200 after $max_attempts seconds" >&2
        return 1
      fi
      sleep 1
    done
  done
}

strip_escape_sequences() {
  # ANSI escape code - Wikipedia https://en.wikipedia.org/wiki/ANSI_escape_code
  # BusyBox sed(1) does not accept `\octal` or `\xhex`.
  sed -E -e 's/\[[0-9;]*[ABCDEFGHJKSTmin]//g'
}

# Absolute path to relative path
abs2rel() {
  local target="$1"
  shift
  local source="$PWD"
  if test "$#" -gt 0
  then
    source="$1"
  fi
  local common="$source"
  local back=
  while test "${target#"$common"}" = "${target}"
  do
    common=$(dirname "$common")
    back="../${back}"
  done
  echo "${back}""${target#"$common"/}"
}

# shuf(1) for macOS environment.
if ! command -v shuf >/dev/null 2>&1
then
  alias shuf='sort -R'
fi

is_macos && alias sha1sum='shasum -a 1'

# Memoize the (mainly external) command output.
memoize() {
  local cache_file_path
  cache_file_path="$TEMP_DIR"/cache-"$(echo "$@" | sha1sum | cut -d' ' -f1)"
  if ! test -r "$cache_file_path"
  then
    "$@" >"$cache_file_path" || return $?
  fi
  cat "$cache_file_path"
}

# Current cache file path for memoization.
cache_file_path_cb3727b=

begin_memoize() {
  cache_file_path_cb3727b="$TEMP_DIR"/cache-"$(echo "$@" | sha1sum | cut -d' ' -f1)"
  if test -r "$cache_file_path_cb3727b"
  then
    cat "$cache_file_path_cb3727b"
    return 1
  fi
  exec 9>&1
  exec >"$cache_file_path_cb3727b"
}

end_memoize() {
  exec 1>&9
  exec 9>&-
  cat "$cache_file_path_cb3727b"
}

# The path to the shell executable which is running the script.
shell_path() {
  begin_memoize d57754a "$@" || return 0

  if test "${BASH+set}" = set
  then
    echo "$BASH"
  elif is_windows && test "${SHELL+set}" = set && test "$SHELL" = "/bin/sh" && "$SHELL" --help 2>&1 | grep -q "BusyBox"
  then
    echo "$SHELL"
  else
    local path=
    if test -e /proc/$$/exe
    then
      path="$(realpath /proc/$$/exe)" || return 1
    else
      path="$(realpath "$(ps -p $$ -o comm=)")" || return 1
    fi 
    echo "$path"
  fi

  end_memoize
}

# The implementation name of the shell which is running the script. Not "sh" but "bash", "ash", "dash", etc.
shell_name() {
  if test "${BASH+set}" = set
  then
    echo "bash"
  elif is_windows && test "${SHELL+set}" = set && test "$SHELL" = "/bin/sh" && "$SHELL" --help 2>&1 | grep -q "BusyBox"
  then
    echo "ash"
  else
    local path=
    if test -e /proc/$$/exe
    then
      path="$(realpath /proc/$$/exe)" || return 1
    else
      path="$(realpath "$(ps -p $$ -o comm=)")" || return 1
    fi
    case "${path##*/}" in
      (bash) echo "bash";;
      (ash) echo "ash";;
      (dash) echo "dash";;
      (sh|busybox)
        if "$path" --help 2>&1 | grep -q "BusyBox"
        then
          echo "ash"
        else
          echo "Cannot detect shell: $path" >&2
          return 1
        fi
        ;;
      (*)
        echo "Unknown shell: $path" >&2
        return 1
        ;;
    esac
  fi
}

is_dash() {
  test "$(shell_name)" = "dash"
}

is_ash() {
  test "$(shell_name)" = "ash"
}

is_bash() {
  test "$(shell_name)" = "bash"
}

# Check if the file(s)/directory(s) are newer than the destination.
newer() {
  local found_than=false
  local dest=
  local arg
  for arg in "$@"
  do
    shift
    if test "$arg" = "--than"
    then
      found_than=true
    elif $found_than
    then
      dest="$arg"
    else
      set -- "$@" "$arg"
    fi
  done
  if test -z "$dest"
  then
    echo "Missing --than option" >&2
    exit 1
  fi
  if test "$#" -eq 0
  then
    echo "No source files specified" >&2
    exit 1
  fi
  # If the destination does not exist, sources are considered newer than the destination.
  if ! test -e "$dest"
  then
    echo "Destination does not exist: $dest" >&2
    return 0
  fi
  # If the destination is a directory, the newest file in the directory is used.
  if test -d "$dest"
  then
    if is_bsd
    then
      dest="$(find "$dest" -type f -exec stat -l -t "%F %T" {} \+ | cut -d' ' -f6- | sort -n | tail -1 | cut -d' ' -f3)"
    else
      dest="$(find "$dest" -type f -exec stat -Lc '%Y %n' {} \+ | sort -n | tail -1 | cut -d' ' -f2)"
    fi
  fi
  if test -z "$dest"
  then
    echo "No destination file found" >&2
    return 0
  fi
  test -n "$(find "$@" -newer "$dest" 2>/dev/null)"
}

# Returns true if any source file is older than the destination file.
older() {
  ! newer "$@"
}

# Kill child processes for each shell/platform.
kill_child_processes() {
  if is_windows
  then
    # Windows BusyBox ash
    # If the process is killed by PID, ash does not kill `exec`ed subprocesses.
    local jids
    jids="$TEMP_DIR"/jids
    # ash provides a "jobs pipe".
    jobs | sed -E -e 's/^[^0-9]*([0-9]+).*Running *(.*)/\1/' >"$jids"
    while read -r jid
    do
      kill "%$jid" || :
      wait "%$jid" || :
      echo "Killed %$jid" >&2
    done <"$jids"
  elif is_macos
  then
    pkill -P $$ || :
  elif is_linux
  then
    if is_bash
    then
      local jids
      jids="$TEMP_DIR"/jids
      # Bash provides a "jobs pipe".
      jobs | sed -E -e 's/^[^0-9]*([0-9]+).*Running *(.*)/\1/' >"$jids"
      while read -r jid
      do
        kill "%$jid" || :
        wait "%$jid" || :
        echo "Killed %$jid" >&2
      done <"$jids"
    else
      pkill -P $$ || :
    fi
  else
    echo "kill_child_processes: Unsupported platform or shell." >&2
    exit 1
  fi
}

# Open the URL in the browser.
browse() {
  if is_linux
  then
    xdg-open "$1"
  elif is_macos
  then
    open "$1"
  elif is_windows
  then
    PowerShell -Command "Start-Process '$1'"
  else
    echo "Unsupported OS: $(uname -s)" >&2
    exit 1
  fi
}

# Get a key from the user without echoing.
get_key() {
  if is_linux || is_macos
  then
    local saved_stty; saved_stty="$(stty -g)" || return $?
    stty -icanon -echo
    dd bs=1 count=1 2>/dev/null
    stty "$saved_stty"
    return
  fi
  local key
  # Bash and BusyBox Ash provide the `-s` (silent mode) option.
  if is_ash || is_bash
  then
    # shellcheck disable=SC3045
    read -rsn1 key
  # Otherwise, the input is echoed
  else
    read -r key
  fi
  echo "$key"
}

# Show a message and get an input from the user.
prompt() {
  local message="${1:-Text}"
  local default="${2:-}"
  printf "%s: (%s) " "$message" "$default" >&2
  local response
  read -r response
  if test -z "$response"
  then
    response="$default"
  fi
  printf "%s" "$response"
}

# Print a message and get confirmation.
prompt_confirm() {
  local message="${1:-Text}"
  local default="${2:-n}"
  local selection
  case "$default" in
    (y|Y|yes|Yes|YES)
      default=y
      selection="Y/n"
      ;;
    (n|N|no|No|NO)
      default=n
      selection="y/N"
      ;;
    (*)
      echo "Invalid default value: $default" >&2
      return 1
  esac
  printf "%s [%s]: " "$message" "$selection" >&2
  local response
  response="$(get_key)"
  if test -z "$response"
  then
    response="$default"
  fi
  echo "$response" >&2
  case "$response" in
    (y|Y)
      return 0
      ;;
    (n|N)
      return 1
      ;;
  esac
}

# Create a file from the standard input if it does not exist.
ensure_file() {
  local file_path="$1"
  if test -f "$file_path"
  then
    echo "File $file_path already exists. Skipping creation." >&2
    return 0
  fi
  echo "Creating file $file_path." >&2
  mkdir -p "$(dirname "$file_path")"
  cat >"$file_path"
}

underline() {
  printf '\033[4m%s\033[0m' "$1"
}

bold() {
  printf '\033[1m%s\033[0m' "$1"
}

enclose_with_brackets() {
  printf '[%s]' "$1"
}

# Emphasize text.
emph() {
  if test -z "$1"
  then
    return
  fi
  if is_windows
  then
    enclose_with_brackets "$(bold "$(underline "$1")")"
  else
    bold "$(underline "$1")"
  fi
}

# Sort version strings.
# Version strings that are composed of three parts are sorted considering the third part as a patch version.
# Long option `--version-sort` is specific to BSD sort(1).
# shellcheck disable=SC2120
sort_version() {
  sed -E -e '/-/! { s/^([^.]+(\.[^.]+){2})$/\1_/; }' -e 's/-patch/_patch/' | sort -V "$@" | sed -e 's/_$//' -e 's/_patch/-patch/'
}

# Check if the version is greater than the specified version.
version_gt() {
  test "$(printf '%s\n' "$@" | sort_version | head -n 1)" != "$1"
}

version_ge() {
  test "$(printf '%s\n' "$@" | sort_version -r | head -n 1)" = "$1"
}

# Left/Right-Word-Boundary regex is incompatible with BSD sed // re_format(7) https://man.freebsd.org/cgi/man.cgi?query=re_format&sektion=7
lwb='\<'
rwb='\>'
# shellcheck disable=SC2034
if is_bsd
then
  lwb='[[:<:]]'
  rwb='[[:>:]]'
fi

# Print a menu item with emphasis if a character is prefixed with "&".
menu_item() {
  echo "$1" | sed -E \
    -e 's/&&/@ampersand_ff37f3a@/g' \
    -e "s/^([^&]*)(&([^& ]))?(.*)$/\1${is1}\3${is1}\4/" \
  | (
    IFS="$is1" read -r pre char_to_emph post
    if test -n "$char_to_emph"
    then
      printf -- "%s%s%s" "$pre" "$(emph "$char_to_emph")" "$post"
    else
      printf -- "%s" "$pre"
    fi
  ) | sed -E -e 's/@ampersand_ff37f3a@/\&/g'
  echo
}

# Print a menu
menu() {
  local arg
  for arg in "$@"
  do
    printf -- "- "
    menu_item "$arg"
  done
}

# Get the space-separated nth (1-based) field.
field() {
  # shellcheck disable=SC2046
  printf "%s\n" $(cat) | head -n "$1" | tail -n 1
}

# tac(1) for macOS environment.
if ! command -v tac >/dev/null 2>&1
then
  tac() {
    tail -r
  }
fi

# Check if a directory is empty.
is_dir_empty() {
  if ! test -d "$1"
  then
    return 1
  fi
  if ! test -e "$1"/* 2>/dev/null
  then
    return 0
  fi
  return 1
}

# [<file>] Read the file and print substituting environment variables. Unlike envsubst(1), this tries to expand undefined environment variables and fails for that.
env_subst() {
  local template_file="$1"
  eval "cat <<EOF
$(cat "$template_file")
EOF"
}

# [regex replacement ...] Substitute text that matches regex patterns in stdin input. Takes pairs of regex/replacement arguments and applies them via sed(1).
resubst() {
  local step=2
  local i=0 n=$(($# / step))
  while test "$i" -lt "$n"
  do
    set -- "$@" -e "s${us}$1${us}$2${us}g"
    shift $step
    i=$((i + 1))
  done
  sed "$@"
}

#endregion

# ==========================================================================
#region Install/Update task-sh task scripts

github_prepare_token() {
  first_call b1929c9 || return 0
  if test "${GITHUB_TOKEN+set}" = set
  then
    echo "Using existing \$GITHUB_TOKEN environment variable." >&2
    return 0
  fi
  if command -v gh >/dev/null
  then
    if gh auth status >/dev/null
    then
      echo "Using GitHub token gh(1) provides." >&2
      GITHUB_TOKEN="$(gh auth token)"
      return 0
    fi
  fi
  echo "Accessing GitHub API with anonymous access." >&2
}

github_api_request() {
  local url="$1"
  github_prepare_token
  set -- \
    --silent \
    --header "X-GitHub-Api-Version: 2022-11-28" \
    --header "Accept: application/vnd.github+json" \
    --fail
  if test "${GITHUB_TOKEN+set}" = set
  then
    set -- "$@" --header "Authorization: Bearer $GITHUB_TOKEN"
  fi
  "$VERBOSE" && echo "Accessing GitHub API: $url" >&2
  curl "$@" "$url"
}

github_tree_get() {
  local owner=
  local repos=
  local tree_sha=main
  OPTIND=1; while getopts _-: OPT
  do
    test "$OPT" = - && OPT="${OPTARG%%=*}" && OPTARG="${OPTARG#"$OPT"=}"
    case "$OPT" in
      (owner) owner="$OPTARG";;
      (repos) repos="$OPTARG";;
      (branch|tag|tree|tree-sha) tree_sha="$OPTARG";;
      (*) echo "Unexpected option: $OPT" >&2; exit 1;;
    esac
  done
  shift $((OPTIND-1))

  # REST API endpoints for Git trees - GitHub Docs https://docs.github.com/en/rest/git/trees
  local url
  url="$(printf "https://api.github.com/repos/%s/%s/git/trees/%s" "$owner" "$repos" "$tree_sha")"
  github_api_request "$url"
}

# Fetch raw content of a file from a GitHub repository
# Usage: github_raw_fetch [OPTIONS]
# Options:
#   --owner=OWNER         GitHub repository owner/organization
#   --repos=REPOS         GitHub repository name
#   --tree-sha=SHA        Tree SHA, branch name, or tag name (default: main). Aliases: --branch, --tag, --tree
#   --path=PATH           Path to the file within the repository
github_raw_fetch() {
  local owner=
  local repos=
  local tree_sha=main
  local path=
  OPTIND=1; while getopts _-: OPT
  do
    test "$OPT" = - && OPT="${OPTARG%%=*}" && OPTARG="${OPTARG#"$OPT"=}"
    case "$OPT" in
      (owner) owner="$OPTARG";;
      (repos) repos="$OPTARG";;
      (branch|tag|tree|tree-sha) tree_sha="$OPTARG";;
      (path) path="$OPTARG";;
      (*) echo "Unexpected option: $OPT" >&2; exit 1;;
    esac
  done
  shift $((OPTIND-1))

  path="${path#/}"
  local url
  url="$(printf "https://raw.githubusercontent.com/%s/%s/%s/%s" "$owner" "$repos" "$tree_sha" "$path")"
  curl --fail --silent "$url"
}

state_path="$PROJECT_DIR/.task-sh-state.json"

# [<name>...] Install task-sh files.
subcmd_task__install() {
  local force=false
  if test "$#" -gt 0 && test "$1" = "--force"
  then
    shift
    force=true
  fi
  local rc=0
  local resp
  local main_branch=main
  resp="$(github_tree_get --owner="knaka" --repos="task-sh")"
  local latest_commit; latest_commit="$(printf "%s" "$resp" | jq -r .sha)"
  "$VERBOSE" && echo "Latest commit of \"$main_branch\" is \"$latest_commit\"." >&2
  if ! test -r "$state_path"
  then
    echo '{}' >"$state_path"
  fi
  local file
  local name
  for file in "$@"
  do
    name="${file##*/}"
    "$VERBOSE" && echo "Name: \"$name\"."
    local indent="  "
    local node mode last_sha
    local last_sha=
    last_sha="$(jq -r --arg name "$name" '.last_sha[$name] // ""' "$state_path")"
    "$VERBOSE" && echo "${indent}Last installed SHA:" "$last_sha"
    local local_sha=
    if test -r "$file"
    then
      local_sha="$(git hash-object "$file")"
    fi
    "$VERBOSE" && echo "${indent}Local SHA:" "$local_sha"
    if test -n "$last_sha" -a -n "$local_sha" -a "$last_sha" != "$local_sha"
    then
      echo "\"$name\" is modified locally." >&2
      rc=1
      continue
    fi
    if test "$file" = "$name"
    then
      case "$file" in
        (*/*) ;;
        (*) file="$TASKS_DIR"/"$name"
      esac
    fi
    node="$(echo "$resp" | jq -c --arg name "$name" '.tree[] | select(.path == $name)')"
    if test -z "$node"
    then
      echo "\"$name\" does not exist in the remote repository."
      rc=1
      continue
    fi
    local new_sha
    new_sha="$(echo "$node" | jq -r .sha)"
    "$VERBOSE" && echo "${indent}Remote SHA:" "$new_sha" >&2
    if ! "$force" && test -n "$local_sha" -a "$new_sha" = "$last_sha"
    then
      echo "\"$name\" is up to date. Skipping." >&2
      continue
    fi
    # shellcheck disable=SC2059
    printf "Downloading \"$name\" ... " >&2
    if test "$name" = "task.sh"
    then
      "$VERBOSE" && Lazily replacing $file.new to $file.
      chaintrap "mv \"$file.new\" \"$file\"" EXIT
      local file="$file.new"
    fi
    github_raw_fetch --owner="knaka" --repos="task-sh" --tree-sha="$latest_commit" --path=/"$name" >"$file"
    echo "done." >&2
    local temp_json="$TEMP_DIR"/1caef61.json
    jq --arg name "$name" --arg sha "$new_sha" '.last_sha[$name] = $sha' "$state_path" >"$temp_json"
    cat "$temp_json" >"$state_path"
    mode="$(echo "$node" | jq -r .mode)"
    "$VERBOSE" && echo "  Mode:" "$mode"
    chmod "${mode#???}" "$file"
  done
  return "$rc"
}

# Update task-sh files.
task_task__update() {
  local file
  local excludes=":"
  for file in "$TASKS_DIR"/project*.lib.sh
  do
    test -e "$file" || continue
    excludes="$excludes:$file:"
  done
  set --
  for file in "$TASKS_DIR"/*.lib.sh "$TASKS_DIR"/task.sh
  do
    test -r "$file" || continue
    case "$excludes" in
      (*:$file:*) continue;;
    esac
    set -- "$@" "$file"
  done
  subcmd_task__install "$INITIAL_PWD"/task "$INITIAL_PWD"/task.cmd "$@"
}

#endregion

# ==========================================================================
#region Main

sub_helps_e4c531b=""

# Add a function to print a sub help section
add_sub_help() {
  sub_helps_e4c531b="$sub_helps_e4c531b$1 "
}

psv_task_file_paths_4a5f3ab=

# Show task-sh help
tasksh_help() {
  cat <<EOF
Usage:
  $ARG0BASE [flags] <subcommand> [args...]
  $ARG0BASE [flags] <task[arg1,arg2,...]> [tasks...]

Flags:
  -d, --directory=<dir>  Change directory before running tasks.
  -h, --help             Display this help and exit.
  -v, --verbose          Verbose mode.
EOF
  # shellcheck disable=SC2086
  lines="$(
    IFS="|"
    awk \
      '
        /^#/ { 
          desc = $0
          gsub(/^#+[ ]*/, "", desc)
          next
        }
        /^(task_|subcmd_)[[:alnum:]_]()/ {
          func_name = $1
          sub(/\(\).*$/, "", func_name)
          type = func_name
          sub(/_.*$/, "", type)
          name = func_name
          sub(/^[^_]+_/, "", name)
          gsub(/__/, ":", name)
          basename = FILENAME
          sub(/^.*\//, "", basename)
          print type " " name " " basename " " desc
          desc = ""
          next
        }
        {
          desc = ""
        }
      ' \
      $psv_task_file_paths_4a5f3ab
  )"
  local i
  for i in subcmd task
  do
    echo
    if test "$i" = subcmd
    then
      echo "Subcommands:"
    else
      echo "Tasks:"
    fi
    local max_name_len; max_name_len="$(
      echo "$lines" \
      | while read -r t name _
        do
          test "$t" = "$i" || continue
          echo "${#name}"
        done \
      | sort -nr \
      | head -1
    )"
    echo "$lines" \
    | sort \
    | while read -r type name basename desc
      do
        test "$type" = "$i" || continue
        case "${basename}" in
          # Emphasize project tasks/subcommands, not shared ones.
          (project*.lib.sh)
            if is_terminal
            then
              # Underline
              local padding_len=$((max_name_len - ${#name}))
              printf "  \033[4m%s\033[0m%-${padding_len}s  %s\n" "$name" "" "$desc"
            else
              # Asterisk
              printf "* %-${max_name_len}s  %s\n" "$name" "$desc"
            fi
            ;;
          (*)
            printf "  %-${max_name_len}s  %s\n" "$name" "$desc"
            ;;
        esac
      done
  done
  local sub_help
  for sub_help in $sub_helps_e4c531b
  do
    echo
    "$sub_help"
  done
}

# Execute a command in task.sh context.
subcmd_task__exec() {
  local saved_shell_flags; saved_shell_flags="$(set +o)"
  set +o errexit
  if alias "$1" >/dev/null 2>&1
  then
    # shellcheck disable=SC2294
    eval "$@"
  else
    "$@"
  fi
  echo "Exit status: $?" >&2
  eval "$saved_shell_flags"
}

usv_called_task_7ef15a7="$us"

# Call the task/subcommand. If the unique task (including the arguments) is already called before, this returns immediately. Calls before/after hooks accordingly.
call_task() {
  local func_name="$1"
  shift
  local task_name=
  case "$func_name" in
    (task_*)
      local cmd_with_args="$func_name $*"
      case "$usv_called_task_7ef15a7" in
        (*"$us$cmd_with_args$us"*)
          return 0
          ;;
      esac
      usv_called_task_7ef15a7="$usv_called_task_7ef15a7$cmd_with_args$us"
      task_name="${func_name#task_}"
      ;;
    (subcmd_*) task_name="${func_name#subcmd_}";;
    (*) return 1;;
  esac
  local prefix
  prefix="$task_name"
  while :
  do
    if type "before_$prefix" >/dev/null 2>&1
    then
      "$VERBOSE" && echo "Calling before function:" "before_$prefix" "$func_name" "$@" >&2
      "before_$prefix" "$func_name" "$@" || return $?
    fi
    test -z "$prefix" && break
    case "$prefix" in
      (*__*) prefix="${prefix%__*}";;
      (*) prefix=;;
    esac
  done
  "$VERBOSE" && echo "Calling task function:" "$func_name" "$@" >&2
  if alias "$func_name" >/dev/null 2>&1
  then
    # shellcheck disable=SC2294
    eval "$func_name" "$@"
  else
    "$func_name" "$@"
  fi
  prefix="$task_name"
  while :
  do
    if type "after_$prefix" >/dev/null 2>&1
    then
      "$VERBOSE" && echo "Calling after function:" "after_$prefix" "$func_name" "$@" >&2
      "after_$prefix" "$func_name" "$@" || return $?
    fi
    case "$prefix" in
      (*__*) ;;
      (*) break;;
    esac
    prefix="${prefix%__*}"
  done
}

tasksh_main() {
  set -o nounset -o errexit

  chaintrap kill_child_processes EXIT TERM INT

  PROJECT_DIR="$(realpath "$PROJECT_DIR")"
  export PROJECT_DIR
  TASKS_DIR="$(realpath "$TASKS_DIR")"
  export TASKS_DIR

  # Before loading task files, permit running task:install to fetch and overwrite existing task files even when they cannot be loaded due to errors or missing `source`d files.
  if test "$#" -gt 0 && test "$1" = "task:install" -o "$1" = "subcmd_task__install"
  then
    shift
    subcmd_task__install "$@"
    return 0
  fi

  # Load all task files in the tasks directory. All task files are sourced in the $TASKS directory context.
  push_dir "$TASKS_DIR"
  local path
  for path in "$TASKS_DIR"/task.sh "$TASKS_DIR"/*.lib.sh
  do
    test -r "$path" || continue
    psv_task_file_paths_4a5f3ab="$psv_task_file_paths_4a5f3ab$path|"
    # shellcheck disable=SC1090
    . "$path"
  done
  pop_dir

  # Parse the command line arguments.
  shows_help=false
  skip_missing=false
  ignore_missing=false
  OPTIND=1; while getopts hvsi-: OPT
  do
    test "$OPT" = - && OPT="${OPTARG%%=*}" && OPTARG="${OPTARG#"$OPT"=}"
    case "$OPT" in
      (h|help) shows_help=true;;
      (s|skip-missing) skip_missing=true;;
      (i|ignore-missing) ignore_missing=true;;
      (v|verbose)
        export VERBOSE=true
        ;;
      (\?) tasksh_help; exit 1;;
      (*) echo "Unexpected option: $OPT" >&2; exit 1;;
    esac
  done
  shift $((OPTIND-1))

  # Show help message and exit.
  if $shows_help || test "$#" -eq 0
  then
    tasksh_help
    exit 0
  fi

  # Execute the subcommand and exit.
  local subcmd="$1"
  subcmd="$(echo "$subcmd" | sed -r -e 's/:/__/g')"
  if type subcmd_"$subcmd" >/dev/null 2>&1
  then
    shift
    if alias subcmd_"$subcmd" >/dev/null 2>&1
    then
      # shellcheck disable=SC2294
      call_task subcmd_"$subcmd" "$@"
      exit $?
    fi
    call_task subcmd_"$subcmd" "$@"
    exit $?
  fi
  # Called not by subcommand name but by function name.
  case "$subcmd" in
    (subcmd_*)
      if type "$subcmd" >/dev/null 2>&1
      then
        shift
        call_task "$subcmd" "$@"
        exit $?
      fi
      ;;
  esac

  # Run tasks.
  local task_with_args
  for task_with_args in "$@"
  do
    local task_name="$task_with_args"
    args=""
    case "$task_with_args" in
      # Task with arguments.
      (*\[*)
        task_name="${task_with_args%%\[*}"
        args="$(echo "$task_with_args" | sed -r -e 's/^.*\[//' -e 's/\]$//' -e 's/,/ /')"
        ;;
    esac
    task_name="$(echo "$task_name" | sed -r -e 's/:/__/g')"
    if type task_"$task_name" >/dev/null 2>&1
    then
      # shellcheck disable=SC2086
      call_task task_"$task_name" $args
      continue
    fi
    # Called not by task name but by task function name.
    case "$task_name" in
      (task_*)
        # shellcheck disable=SC2086
        call_task "$task_name" $args
        continue
        ;;
    esac
    if ! $skip_missing
    then
      echo "Unknown task: $task_with_args" >&2
    fi
    if ! $skip_missing && ! $ignore_missing
    then
      exit 1
    fi
  done
}

# Run the main function if this script is executed as task runner.
case "${0##*/}" in
  (task|task.sh)
    tasksh_main "$@"
    ;;
esac

#endregion
