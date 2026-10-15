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

## cli

| command    | description          |
| ---------- | -------------------- |
| `sike run` | start the web server |

## env

| var         | default | what              |
| ----------- | ------- | ----------------- |
| `SIKE_ADDR` | `:8090` | listen address    |
