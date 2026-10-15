htmx-version := "2.0.11"

run: css
    go run . run

test:
    go test -race ./...

# the milestone gate
check: lint css-check test

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

# site.css is committed and embedded, so go builds without node. compile it
# after touching input.css or adding a class.
css: _node
    npm run css

css-watch: _node
    npm run css:watch

# fail if the committed site.css is stale against input.css + the sources it
# scans (a new class with no rebuild ships unstyled). builds to a temp file.
css-check: _node
    #!/usr/bin/env bash
    set -euo pipefail
    fresh=$(mktemp "${TMPDIR:-/tmp}/sike-css.XXXXXX")
    trap 'rm -f "$fresh"' EXIT
    npx tailwindcss -i view/ui/static/css/input.css -o "$fresh" --minify 2>/dev/null
    cmp -s "$fresh" view/ui/static/css/site.css || { echo "site.css is stale — run: just css"; exit 1; }
    echo "site.css is current"

# copy the self-hosted fonts out of the pinned @fontsource packages
fonts: _node
    cp node_modules/@fontsource/space-grotesk/files/space-grotesk-latin-{500,700}-normal.woff2 view/ui/static/fonts/
    cp node_modules/@fontsource/space-mono/files/space-mono-latin-400-normal.woff2 view/ui/static/fonts/

# tailwind loads native binaries built per os/cpu. a node_modules synced from
# another machine has the wrong ones: reinstall from the lockfile when so.
_node:
    #!/usr/bin/env bash
    set -euo pipefail
    node -e 'require("@tailwindcss/oxide"); require("@parcel/watcher")' 2>/dev/null && exit 0
    echo "node_modules is missing or built for another os — npm ci for $(uname -sm)"
    npm ci --no-audit --no-fund
