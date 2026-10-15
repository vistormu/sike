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

# the image the homelab runs, built here to try it (github publishes it on a
# v* tag). run it: docker run --rm -p 8090:8090 -v sike-data:/data sike:dev
image version="dev":
    docker build --build-arg VERSION={{version}} -t sike:{{version}} .

# tags main and pushes the tag; github tests it and publishes
# ghcr.io/vistormu/sike:<version> (.github/workflows/image.yml), which the
# homelab pins. with no version, it says the last one.
# publish a version on github: just release vX.Y.Z
release version="":
    #!/usr/bin/env bash
    set -euo pipefail
    v="{{version}}"
    last=$(git describe --tags --abbrev=0 --match 'v*' 2>/dev/null || echo none)
    [ -n "$v" ] || { echo "last release: $last. release one with: just release vX.Y.Z"; exit 0; }
    [[ "$v" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]] || { echo "a version is vX.Y.Z, like v0.1.0 (got $v)"; exit 1; }
    [ "$(git branch --show-current)" = main ] || { echo "release from main"; exit 1; }
    [ -z "$(git status --porcelain)" ] || { echo "the tree has changes: commit them first"; exit 1; }
    git fetch -q origin main --tags
    [ "$(git rev-parse HEAD)" = "$(git rev-parse origin/main)" ] || { echo "main is not on github: git push first"; exit 1; }
    ! git rev-parse -q --verify "refs/tags/$v" >/dev/null || { echo "$v exists (last release: $last)"; exit 1; }
    just check
    git tag -a "$v" -m "sike $v"
    git push origin "$v"
    echo "$v is on its way: https://github.com/vistormu/sike/actions"
    echo "then: ghcr.io/vistormu/sike:$v"

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

# install node_modules from the lockfile when it is missing (it is not synced)
# or was built for another os (tailwind loads native binaries per os/cpu).
_node:
    #!/usr/bin/env bash
    set -euo pipefail
    node -e 'require("@tailwindcss/oxide"); require("@parcel/watcher")' 2>/dev/null && exit 0
    echo "node_modules is missing or built for another os — npm ci for $(uname -sm)"
    npm ci --no-audit --no-fund
