# vim: set filetype=sh tabstop=2 shiftwidth=2 expandtab :
# shellcheck shell=sh
"${sourced_6cc6268-false}" && return 0; sourced_6cc6268=true

set -- "$PWD" "$@"; if test "${2:+$2}" = _LIBDIR; then cd "$3" || exit 1; fi
set -- _LIBDIR . "$@"
. ./utils.lib.sh
. ./time.lib.sh
. ./assert.lib.sh
shift 2
cd "$1" || exit 1; shift


test_time() {
  local result

  # Outputs current date and time in ISO-8601 format.
  result="$(date_iso)"
  assert_match '^[[:digit:]]{4}-[[:digit:]]{2}-[[:digit:]]{2}T[[:digit:]]{2}:[[:digit:]]{2}:[[:digit:]]{2}\+[[:digit:]]{4}$' "$result"

  # Outputs current date and time in ISO-8601 format in UTC.
  result="$(TZ=UTC0 date_iso)"
  assert_match '^[[:digit:]]{4}-[[:digit:]]{2}-[[:digit:]]{2}T[[:digit:]]{2}:[[:digit:]]{2}:[[:digit:]]{2}\+0000$' "$result"

  local file="$TEMP_DIR"/file
  touch "$file"

  # Sets timestamp with UTC date.
  set_last_mod_iso "$file" "2024-01-01T12:00:00Z"
  assert_eq "$(TZ=UTC0 last_mod_iso "$file")" "2024-01-01T12:00:00+0000"
  # assert_eq "$(TZ=Asia/Tokyo last_mod_iso "$file")" "2024-01-01T21:00:00+0900"

  # Sets timestamp with a date with timezone offset.
  set_last_mod_iso "$file" "2024-01-01T09:00:00+0900"
  assert_eq "$(TZ=UTC0 last_mod_iso "$file")" "2024-01-01T00:00:00+0000"
  # assert_eq "$(TZ=Asia/Tokyo last_mod_iso "$file")" "2024-01-01T09:00:00+0900"

  # Converts date to epoch.
  assert_eq 1735732800 "$(iso_to_epoch "2025-01-01T12:00:00Z")"
  assert_eq 1735700400 "$(iso_to_epoch "2025-01-01T12:00:00+0900")"

  # Converts epoch to date.
  assert_eq "2025-01-01T12:00:00+0000" "$(TZ=UTC0 epoch_to_iso 1735732800)"
  # assert_eq "2025-01-01T12:00:00+0900" "$(TZ=Asia/Tokyo epoch_to_iso 1735700400)"
}
