# vim: set filetype=sh tabstop=2 shiftwidth=2 expandtab :
# shellcheck shell=sh
test "${sourced_a642529-}" = true && return 0; sourced_a642529=true

set -- "$PWD" "$@"; if test "${2:+$2}" = _LIBDIR; then cd "$3" || exit 1; fi
set -- _LIBDIR . "$@"
. ./utils.lib.sh
  register_temp_cleanup
shift 2
cd "$1" || exit 1; shift

# List IP ports in use.
ip_ports_in_use() {
  if is_windows
  then
    # -a: Displays all connections and listening ports.
    # -n: Displays addresses and port numbers in numerical form.
    # -p protocol: Shows connections for the protocol specified by protocol.
    netstat.exe -a -n -p TCP | grep TCP | awk '{ print $2 }' | sed -n -e 's/^.*://p' | sort -n | uniq
  elif is_macos
  then
    # -a: Show the state of all sockets.
    # -n: Show numerical addresses instead of trying to determine symbolic host, port or user names.
    # -v: Verbose.
    # -s protocol: Display statistics for the specified protocol.
    netstat -anvp tcp | grep ^tcp4 | awk '{ print $4 }' | sed 's/.*\.//'
  elif is_linux
  then
    if ! command -v ss >/dev/null
    then
      if is_debian
      then
        echo "ss(8) not found. Please install iproute2 package." >&2
        exit 1
      fi
      echo "ss(8) not found." >&2
      exit 1
    fi
    # --tcp: Display TCP sockets.
    # --all: Display all sockets. (Not only listening sockets (-n), but also established connections.)
    # --numeric: Do not resolve service names.
    ss --tcp --all --numeric --no-header | awk '{ print $4 }' | sed -n -e 's/^.*://p'
  else
    echo "Not implemented" >&2
    exit 1
  fi
}

ports_used_in_session_path=

init_ports_used_in_session_path() {
  if test -z "$ports_used_in_session_path"
  then
    ports_used_in_session_path="$TEMP_DIR"/0545610
    touch "$ports_used_in_session_path"
  fi
}

# List free IP ports.
ip_free_ports() {
  local priv_begin=49152
  local priv_end=65535
  readonly priv_begin priv_end
  local port="${1:-$priv_begin}"
  local end="${2:-$priv_end}"
  local priv_ports_path="$TEMP_DIR"/f5c41b5
  seq "$port" "$end" >"$priv_ports_path"
  local used_ports_path="$TEMP_DIR"/6157e29
  (ip_ports_in_use | cat "$ports_used_in_session_path") | sort | uniq >"$used_ports_path"
  comm -23 "$priv_ports_path" "$used_ports_path"
}

ip_free_port() {
  init_ports_used_in_session_path
  local port
  port="$(ip_free_ports "$@" | head -n 1)"
  echo "$port" >>"$ports_used_in_session_path"
  echo "$port"
}

# [begin end] Search for a free port in the range.
subcmd_ip__free_port() {
  ip_free_port "$@"
}

ip_random_free_port() {
  init_ports_used_in_session_path
  local port
  port="$(ip_free_ports "$@" | shuf | head -n 1)"
  echo "$port" >>"$ports_used_in_session_path"
  echo "$port"
}

# [begin end] Search for a random free port in the range.
subcmd_ip__random_free_port() {
  ip_random_free_port "$@"
}
