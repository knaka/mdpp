# vim: set filetype=sh tabstop=2 shiftwidth=2 expandtab :
# shellcheck shell=sh
"${sourced_5aae4b0-false}" && return 0; sourced_5aae4b0=true

# TODO: Not adjusted to current style. Fix me.

set -- "$PWD" "$@"; if test "${2:+$2}" = _LIBDIR; then cd "$3" || exit 1; fi
set -- _LIBDIR . "$@"
. ./utils.lib.sh
. ./tools.lib.sh
shift 2
cd "$1" || exit 1; shift

# Called before `subtree:*` tasks/subcommands by the task runner.
before_subtree() {
  local git_top
  git_top="$(git rev-parse --show-toplevel)"
  test "$(realpath "$git_top")" = "$(realpath "$PWD")" && return 0
  cd "$git_top"
  # shellcheck disable=SC2209
  INVOCATION_MODE=exec invoke ./task "$@"
  # Not reached
  return 1
}

# Add git-subtree to this project.
subtree__add() {
  if test "$#" -eq 1
  then
    local name="$1"
    local info=
    info="$(subtree_info "$name")"
    local repository; repository="$(echo "$info" | yq ".repository")"
    local branch; branch="$(echo "$info" | yq ".branch")"
    local prefix; prefix="$(echo "$info" | yq ".prefix")"
    git subtree add --prefix "$prefix" "$repository" "$branch"
  elif test "$#" -ge 2
  then
    local prefix="$1"
    if test -e "$prefix"
    then
      echo "\"$prefix\" already exists. Aborting." >&2
      return 1
    fi
    local repository="$2"
    local branch
    if test "$#" -gt 2
    then
      branch="$3"
    else
      echo "Detecting the main branch for \"$repository\" ..." >&2
      local refs; refs="$(git ls-remote "$repository")"
      if echo "$refs" | grep -q 'refs/heads/main$'
      then
        branch=main
      elif echo "$refs" | grep -q 'refs/heads/master$'
      then
        branch=master
      else
        echo "No \"main\" or \"master\" branch found in \"$repository\"" >&2
        exit 1
      fi
    fi
    git subtree add --prefix "$prefix" "$repository" "$branch"
    touch .subtree.yaml
    local subtree_alias; subtree_alias="$(basename "$prefix")"
    yq --inplace ". += [{\"prefix\": \"$prefix\", \"alias\": \"$subtree_alias\", \"repository\": \"$repository\", \"branch\": \"$branch\"}]" .subtree.yaml
  else
    cat <<EOF >&2
Usage: subtree:add <target_dir> <repository> [<branch>]
   or: subtree:add <prefix|alias>

Adds a subtree from the specified branch of <repository> to the current repository and records the repository and branch in .subtree.yaml. If no branch is specified, it automatically detects and uses "main" or "master" from the repository. If only the prefix|alias is specified, the repository and branch are retrieved from the configuration in .subtree.yaml.
EOF
    return 0
  fi
}

# Remove git-subtree from this project.
subtree__remove() {
  local target_dir="$1"
  git rm -rf "$target_dir"
  touch .subtree.yaml
  yq --inplace "del(.\"$target_dir\")" .subtree.yaml
}

subtree_push_or_pull() {
  local git_subcmd="$1"
  local name="$2"
  local info=
  info="$(subtree_info "$name")"
  local target_dir; target_dir="$(echo "$info" | yq ".prefix")"
  local repository; repository="$(echo "$info" | yq ".repository")"
  local branch; branch="$(echo "$info" | yq ".branch")"
  git subtree "$git_subcmd" --prefix "$target_dir" "$repository" "$branch"
}

# Push subtree changes to remote repository.
subtree__push() {
  subtree_push_or_pull push "$@"
}

# Pull subtree changes from remote repository.
subtree__pull() {
  subtree_push_or_pull pull "$@"
}

subtree_info() {
  local name="$1"
  local info; info="$(yq ".[] | select(.prefix == \"$name\" or .alias == \"$name\")" .subtree.yaml)"
  if test -z "$info"
  then
    echo "\"$name\" is not a valid subtree. Aborting." >&2
    return 1
  fi
  echo "$info"
}

# Show information about a subtree.
subtree__info() {
  local name="$1"
  local info=
  info="$(subtree_info "$name")"
  local target_dir; target_dir="$(echo "$info" | yq ".prefix")"
  git log --grep="git-subtree-dir: $target_dir"
}

# List subtree-s.
subtree__list() {
  cat .subtree.yaml
}
