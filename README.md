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

dev pages: `/dev/components` shows every component variant (compare with figma),
`/dev/forget` forgets the chosen person and opens the picker.

## cli

| command    | description          |
| ---------- | -------------------- |
| `sike run` | start the web server (migrates the db on boot) |
| `sike migrate` | apply pending migrations, print the version |
| `sike op user add <name> <emoji>` | add a person (the emoji is one of the 30 animals) |
| `sike op user list` | list people, oldest first |
| `sike op user rm <id> [-y]` | delete a person and everything theirs (asks first) |
| `sike op area list <user-id>` | a person's seven areas (color, name) |
| `sike op area rename <user-id> <color> <name>` | rename one of them |
| `sike op task add <user-id> <name> --area <area> [--when someday]` | add a task (area: color or name; when: pending, the default, or someday) |
| `sike op task list <user-id>` | a person's tasks, in tab order |
| `sike op task edit <user-id> <task-id> [--name] [--area] [--when]` | change a task; a new when puts it on top there |
| `sike op task rm <user-id> <task-id>` | delete a task: done |
| `sike op task restore <user-id> <task-id>` | undo a delete (only a person's last one) |
| `sike op task move <user-id> <task-id> <when>` | move a task to pending or someday, on top |
| `sike op week show [--date YYYY-MM-DD]` | a week (today's by default): its days and the timeline's hours |

## env

| var         | default | what              |
| ----------- | ------- | ----------------- |
| `SIKE_ADDR` | `:8090` | listen address    |
| `SIKE_DB`   | `sike.db` | sqlite database file, created if missing |
| `SIKE_TZ`   | `Europe/Madrid` | the timezone every time is wall-clock in |
