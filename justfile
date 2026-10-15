htmx-version := "2.0.11"

run:
    go run . run

test:
    go test -race ./...

# the milestone gate
check: lint test

# gofmt over the go files git knows about (tracked and new, minus ignored), then vet
lint:
    #!/usr/bin/env bash
    set -euo pipefail
    files=$(git ls-files --cached --others --exclude-standard '*.go' | while read -r f; do [ -f "$f" ] && echo "$f"; done)
    unformatted=$(gofmt -l $files)
    if [ -n "$unformatted" ]; then echo "gofmt needed (run: gofmt -w <file>):"; echo "$unformatted"; exit 1; fi
    go vet ./...

# re-download pinned vendor files, verifying each against a second cdn
vendor:
    curl -sf https://unpkg.com/htmx.org@{{htmx-version}}/dist/htmx.min.js -o view/ui/static/js/htmx.min.js
    @test "$(sha256sum < view/ui/static/js/htmx.min.js | cut -d' ' -f1)" = "$(curl -sf https://cdn.jsdelivr.net/npm/htmx.org@{{htmx-version}}/dist/htmx.min.js | sha256sum | cut -d' ' -f1)" \
        || { echo "htmx {{htmx-version}}: hash mismatch between cdns, removing download" >&2; rm view/ui/static/js/htmx.min.js; exit 1; }
    @echo "htmx {{htmx-version}} ok (sha256 matches across unpkg and jsdelivr)"
