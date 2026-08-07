<!-- +INCLUDE: DEVELOPMENT.md -->
<!-- +INCLUDE: ./.claude/skills/mise/SKILL.md -->
# “Mise” — tool version and task manager

mise manages project-local tool versions and runs tasks. Tool versions are listed in `.mise/config*.toml` and `.config/mise/conf.d/*.toml`.

mise can be invoked without a global installation using the bootstrap scripts: `./mise` on Linux and macOS, or `.\mise.cmd` on Windows.

````bash
# install all tools
./mise install

# execute a project-locally-installed command
./mise exec -- jq --help

# list available tasks
./mise tasks

# run task `foo`
./mise run foo

# show a task `foo`'s full description and the path of the file that defines it
./mise tasks foo
````
<!-- +END -->

# Testing

- Task `test` runs all tests.
<!-- +END -->

# Guide for Claude

## Documentation, Comments or strings in program code

- Written in simple, technical English.
- If there is an unnatural expression in the English text, correct it to a natural expression.
- If you find a sentence written in Japanese, translate it into English and replace it.
- When referencing tasks in documentation, use the task name (e.g., `astro:build`, `rr:build`, `merge`) rather than the implementation function name (e.g., `task_astro__build`, `task_rr__build`, `task_merge`).
