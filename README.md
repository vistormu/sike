# sike

*sike* is toki pona for circle, cycle: the week that repeats. a personal planner for
two people: routines, dated things, pending things and wishes, placed in a week.

go + net/http + htmx + sqlite, one binary, self-hosted behind tailscale. templates and
static files are embedded (`view/ui/`); htmx is vendored, not loaded from a cdn.

## dev loop

| recipe       | what it does                              |
| ------------ | ----------------------------------------- |
| `just run`   | build the css, start the server on `localhost:8090` |
| `just test`  | `go test -race ./...`                     |
| `just image [version]` | build the image locally (`sike:dev`) |
| `just release [vX.Y.Z]` | publish a version: checks main is clean and on github, runs `just check`, tags and pushes the tag; no version says the last one |
| `just lint`  | `gofmt` + `go vet`                        |
| `just check` | `lint` + `css-check` + `test`, the gate before a commit |
| `just vendor` | re-download pinned htmx, checked against a second cdn |
| `just css`   | compile tailwind: `input.css` → `site.css`  |
| `just css-watch` | recompile the css on change              |
| `just css-check` | fail if the committed `site.css` is stale |
| `just fonts` | copy the self-hosted fonts out of `@fontsource` |

the css is tailwind v4 with the design tokens in `view/ui/static/css/input.css`.
the compiled `site.css` is committed and embedded, so `go build` and `just test` work
without node; only the css recipes need node + npm (they run `npm ci` themselves when
`node_modules` is missing or was installed on another os).

dev pages: `/dev/components` shows every component variant (compare with figma).
settings' "Switch person" forgets the chosen person and opens the picker.

## release

`just release v0.1.0` (main committed and pushed): github tests it and publishes the image
`ghcr.io/vistormu/sike:v0.1.0` (and `:latest`); the homelab pins that version.
`just image` builds the same image locally (`sike:dev`):
`docker run --rm -p 8090:8090 -v sike-data:/data sike:dev`. the image runs as
nonroot (65532), keeps the db in `/data`, and checks itself with `sike health`.

## cli

| command    | description          |
| ---------- | -------------------- |
| `sike run` | start the web server (migrates the db on boot) |
| `sike migrate` | apply pending migrations, print the version |
| `sike backup <file>` | a consistent copy of the db while the server runs (sqlite `VACUUM INTO`); the homelab takes one before each upgrade |
| `sike health` | ask the server on `SIKE_ADDR` whether it is up and its db answers (`/healthz`); exit 1 if not |
| `sike --version` | the version it was built as (`dev` outside the image) |
| `sike op user add <name> <emoji>` | add a person (the emoji is one of the 30 animals) |
| `sike op user list` | list people, oldest first, with their deadline warning |
| `sike op user edit <id> [--name] [--emoji] [--warn 1h\|1d\|2d] [--topic]` | change a person; what is not given stays |
| `sike op user rm <id> [-y]` | delete a person and everything theirs (asks first) |
| `sike op area list <user-id>` | a person's seven areas (color, name) |
| `sike op area rename <user-id> <color> <name>` | rename one of them |
| `sike op task add <user-id> <name> --area <area> [--when …] [--at …] [--date …] [--notify]` | add a task (area: color or name; when: pending, the default, someday, weekly with one `--at "mon,wed 18:00-19:30"` per time, or date with `--date "15/10 11:00-12:00"`, `"15/10"`, `"16/10..18/10"` or `"due 20/10 14:00"`) |
| `sike op task list <user-id>` | a person's tasks, in tab order |
| `sike op task edit <user-id> <task-id> [--name] [--area] [--when] [--at …] [--date …] [--notify]` | change a task; a new when puts it on top there; `--at` replaces the weekly times, `--date` the dates |
| `sike op task rm <user-id> <task-id>` | delete a task: done |
| `sike op task restore <user-id> <task-id>` | undo a delete (only a person's last one) |
| `sike op task move <user-id> <task-id> <when>` | move a task to pending or someday, on top |
| `sike op placement list <user-id> <task-id>` | a weekly task's times |
| `sike op placement add <user-id> <task-id> <days> <start-end> [--notify-start] [--notify-end]` | add a time: `mon,wed 18:00-19:30`, `mon-fri 09:00-17:00` |
| `sike op placement rm <user-id> <placement-id>` | remove a time (not a task's last one) |
| `sike op week show <user-id> [--date YYYY-MM-DD]` | a person's week (today's by default): days, hours, what repeats, what is dated |
| `sike op mood show <user-id> [--date YYYY-MM-DD]` | a person's mood today (or on that day) |
| `sike op mood set <user-id> <1-5 \| face> [--date YYYY-MM-DD]` | give the day its mood: `4` or `🙂` |
| `sike op mood clear <user-id> [--date YYYY-MM-DD]` | take the day's mood away |
| `sike op notify test <user-id>` | send a test message to the person's ntfy topic |
| `sike op notify tick [--at "YYYY-MM-DD HH:MM"] [--dry-run]` | one tick of the notifier: what is due in the 10 minutes up to now (or `--at`) and not sent yet; `--dry-run` only prints |

## env

| var         | default | what              |
| ----------- | ------- | ----------------- |
| `SIKE_ADDR` | `:8090` | listen address    |
| `SIKE_DB`   | `sike.db` | sqlite database file, created if missing |
| `SIKE_TZ`   | `Europe/Madrid` | the timezone every time is wall-clock in |
| `SIKE_NTFY_URL` | (none: notifications off) | the ntfy server, like `https://ntfy.example.com` |
| `SIKE_NTFY_TOKEN` | (none) | an ntfy access token (`tk_…`) with write access to the people's topics |
