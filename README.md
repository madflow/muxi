# muxi

muxi is a Go-based tmux project orchestrator. Define sessions, windows, panes,
layouts, and lifecycle hooks in YAML, then start your workspace with one command.

## Requirements and build

- tmux available on `PATH`.
- A Unix-like shell environment (`$SHELL` is used to run generated scripts,
  falling back to `sh`).
- Go 1.26.2 or newer to build from source.

From the repository root:

```sh
make build
# Or:
go build -o muxi .
```

The examples below use `./muxi`, so installing the binary into `PATH` is not
required. Run them from the repository root, where the binary was built.

## Quick start

Create a project configuration in the current directory:

```sh
./muxi new --local demo
```

This creates `.tmuxinator.yml`. If `$EDITOR` is set, muxi opens the file with that
executable; otherwise, it prints the file path. Use an executable name or path for
`EDITOR`, not a command containing extra arguments.

Replace the generated Rails-oriented template with this Go-oriented example:

```yaml
name: demo
root: .
windows:
  - workspace:
      layout: even-horizontal
      panes:
        - pwd
        -
          - echo 'Project files'
          - ls
  - tests:
      panes:
        - go test ./...
```

`name` and a nonempty `windows` list are required. Here, `root: .` means the
invocation directory, not the directory containing the YAML file. A pane may
contain a single command or a list of commands to send in sequence.

Start the local project:

```sh
./muxi
# Equivalent:
./muxi local
```

Attachment is enabled by default. To create the session without attaching:

```sh
./muxi start --attach=false
```

You can also set `attach: false` in the project YAML. An explicit `--attach` flag
overrides that setting.

## Project discovery

**Local projects:** with no project name, `start` uses `.tmuxinator.yml` in the
current directory, falling back to `.tmuxinator.yaml`. `./muxi` automatically
starts a local project when one exists; otherwise, it prints help. Parent
directories are not searched.

**Named projects:** `./muxi start demo` searches these directories in order,
matching the configuration's filename without its extension:

1. `$TMUXINATOR_CONFIG`, if set.
2. `$XDG_CONFIG_HOME/tmuxinator`, or `~/.config/tmuxinator` when
   `XDG_CONFIG_HOME` is unset.
3. `~/.tmuxinator`.

Named lookup accepts both `.yml` and `.yaml`. The filename used for lookup and
the YAML `name` (the default tmux session name) can differ.

**Creating named projects:** `./muxi new demo` writes `demo.yml` into
`$TMUXINATOR_CONFIG` when set. Otherwise, it uses the existing XDG directory,
then an existing `~/.tmuxinator`; if neither exists, it creates the XDG directory.
`new --local` instead writes `.tmuxinator.yml` in the current directory.

**Listing:** `list` currently includes `.yml` files only. With
`TMUXINATOR_CONFIG` set, it lists only that directory; otherwise, it lists the
existing XDG and legacy home directories. This differs from named lookup, which
also accepts `.yaml` and searches the fallback directories.

## Everyday commands

| Command | Purpose |
| --- | --- |
| `./muxi start [project]` | Start a project, or reuse its existing session. |
| `./muxi stop [project]` | Stop the session identified by the project configuration. |
| `./muxi stop-all` | Stop active sessions whose names match discovered project filenames, not every tmux session. |
| `./muxi local` | Start the current directory's local project (alias: `.`). |
| `./muxi list` | List named projects. |
| `./muxi new demo` | Create or open a named configuration (aliases include `edit` and `open`). |
| `./muxi new demo existing-session` | Import an existing tmux session into a project configuration. |
| `./muxi copy demo demo-copy` | Copy a named configuration. |
| `./muxi delete demo-copy` | Delete a named configuration without prompting; does not stop its session. |
| `./muxi debug [project]` | Print the generated start script without executing it. |
| `./muxi doctor` | Check tmux availability and whether `EDITOR` and `SHELL` are set. |
| `./muxi version` | Print the muxi version. |
| `./muxi commands` | List commands. |
| `./muxi completions start` | Print project-name completion candidates for a supported command. |

A named project can also be started with `./muxi demo`. Command names take
precedence over project-name shorthand.

Common options:

```sh
# Use an explicit project file (also supported by stop and debug).
./muxi start --project-config ./path/to/project.yml

# Override the session name.
./muxi start demo --name demo-extra

# Append windows to the current tmux session; run from inside tmux.
./muxi start demo --append

# Skip the project's pre_window commands.
./muxi start demo --no-pre-window

# List discovered projects with matching active session names, one per line.
./muxi list --active --newline
```

Append mode requires a current tmux session and does not attach again. Custom
session names may not match `stop-all`'s project-filename matching rule.

Use `./muxi <command> --help` for command-specific flags. `completions` prints
candidates, not shell setup scripts; supported arguments are `start`, `stop`,
`edit`, `open`, `copy`, and `delete`.

## Configuration overview

Beyond the quick-start example, supported settings include:

- **Roots:** project `root` sets the working directory; a window's `root`
  overrides it. Relative window roots are resolved against the project root.
  Paths beginning with `~` or `~/` expand to the home directory.
- **Windows and panes:** a window can hold a command, a list of commands, or a
  mapping with `layout`, `root`, and `panes`. Pane entries can be commands,
  command lists, or titled mappings. A null pane starts an empty shell.
- **Setup commands:** project `pre_window` runs before commands in each pane;
  a window's `pre` supplies setup for each explicitly configured pane.
- **Lifecycle hooks:** `on_project_start` runs on each start;
  `on_project_first_start` runs when creating the workspace (also in append
  mode); `on_project_restart` runs when reusing an existing session.
  `on_project_exit` runs after the generated start script's attach step returns
  (or immediately after setup when detached). `on_project_stop` runs before
  `stop` kills the session. `stop-all` kills matching sessions directly and
  does not run those hooks.
- **Startup selection:** `startup_window` and `startup_pane` select the initial
  window and pane, using tmux's names/indexes and configured base indexes.
- **tmux options:** `socket_name`, `socket_path`, `tmux_options`, and
  `tmux_command` customize tmux invocation. `socket_path` takes precedence over
  `socket_name` when both are supplied.

The global `--config` option selects muxi's own configuration file (by default
`~/.muxi.yaml`), **not** a project YAML file. The current implementation loads
that global file but does not consume application settings from it. Use
`--project-config` on `start`, `stop`, or `debug` to select project YAML.

## Compatibility and safety

muxi can reuse supported tmuxinator-style **plain YAML** configurations and
configuration locations. It is not a fully compatible replacement for
[tmuxinator](https://github.com/tmuxinator/tmuxinator). Ruby/ERB templates are not
evaluated; convert templated configurations to plain YAML first. The bundled
`tmuxinator/` directory is a reference project, not muxi's implementation.

**Only run configurations you trust:** pane commands and hooks execute shell
commands with your user's permissions.

`debug` prints the generated script without running its project commands, but
preparation still invokes tmux to query session state and base indexes. It is
not an offline YAML validator or an isolated sandbox.

## Development

Run the Go tests from the repository root:

```sh
make test
# Or:
go test ./...
```

## License

[MIT](LICENSE), copyright 2026 madflow.
