# sike

*sike* is toki pona for circle, cycle: the week that repeats. a personal planner for
two people: routines, dated things, pending things and wishes, placed in a week.

go + net/http + htmx + sqlite, one binary, self-hosted behind tailscale. templates and
static files are embedded (`view/ui/`); htmx is vendored, not loaded from a cdn.

## dev loop

| recipe       | what it does                              |
| ------------ | ----------------------------------------- |
| `just run`   | start the server on `localhost:8090`      |
| `just test`  | `go test -race ./...`                     |
| `just lint`  | `gofmt` + `go vet`                        |
| `just check` | `lint` + `test`, the gate before a commit |
| `just vendor` | re-download pinned htmx, checked against a second cdn |

## cli

| command    | description          |
| ---------- | -------------------- |
| `sike run` | start the web server |

## env

| var         | default | what              |
| ----------- | ------- | ----------------- |
| `SIKE_ADDR` | `:8090` | listen address    |
