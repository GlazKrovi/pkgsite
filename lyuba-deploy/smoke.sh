#!/usr/bin/env bash
# Starts the local documentation server on a Lyuba module and checks the pages
# a visitor would open. Usage: lyuba-deploy/smoke.sh [path/to/lyuba/module]
set -euo pipefail

root=$(cd "$(dirname "$0")/.." && pwd)
mod=${1:-$root/../lyuba/testdata/bonjour}
mod=$(cd "$mod" && pwd)
modpath=$(awk '/^module /{print $2; exit}' "$mod/go.mod")
port=${PORT:-18080}
base=http://localhost:$port
tmp=$(mktemp -d)
cd "$root"
go build -o "$tmp/pkgsite$(go env GOEXE)" ./cmd/pkgsite
"$tmp/pkgsite$(go env GOEXE)" -list=false -http=localhost:$port "$mod" >"$tmp/log" 2>&1 &
pid=$!
trap 'kill $pid 2>/dev/null || true; rm -rf "$tmp"' EXIT

for _ in $(seq 1 60); do curl -s -o /dev/null "$base/" && break; sleep 1; done

fails=0
page() { # page <path> <expected status> [regexp the body must match]
	local code
	code=$(curl -s -o "$tmp/body" -w '%{http_code}' "$base$1")
	if [ "$code" != "$2" ]; then echo "FAIL $1: HTTP $code, wanted $2"; fails=$((fails + 1)); return; fi
	if [ -n "${3:-}" ] && ! grep -Eiq "$3" "$tmp/body"; then echo "FAIL $1: body lacks /$3/"; fails=$((fails + 1)); return; fi
	echo "ok   $1"
}

page /              200 'Lyuba'
page /about         200 'pkg.lyuba.dev'
page /install       200 'install'
page /install.sh    200 'lyuba'
page /install.ps1   200 'lyuba'
page /why           200 'Lyuba'
page /license-policy 200
page "/$modpath"    200 "$(basename "$modpath")"
page "/search?q=$(basename "$modpath")" 200
# nothing Go-branded may leak into the pages
for p in / /about "/$modpath"; do
	curl -s "$base$p" | grep -Eiq 'googletagmanager|illustration-gopher|go-logo|gstatic' && { echo "FAIL $p: Google/Go asset referenced"; fails=$((fails + 1)); }
done
# a pure Go module must be refused, not documented
code=$(curl -s -o /dev/null -w '%{http_code}' "$base/golang.org/x/text")
case $code in 403|404|424|500) echo "ok   pure Go module not served ($code)";; 200) echo "FAIL pure Go module served"; fails=$((fails + 1));; esac

[ "$fails" -eq 0 ] && echo "smoke: all good" || { echo "smoke: $fails failure(s)"; sed 's/^/  log: /' "$tmp/log" | tail -20; exit 1; }
