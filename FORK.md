# pkg.lyuba.dev: a fork of pkgsite

The package site of the [Lyuba](https://github.com/GlazKrovi/lyuba) language, forked from [pkgsite](https://go.googlesource.com/pkgsite) (pkg.go.dev, BSD-3-Clause license, see `LICENSE` and `PATENTS`, both kept). It reads the same index and the same proxy as pkg.go.dev, and only keeps modules that contain `.lyu` files.

## What differs from pkgsite

| | files |
| --- | --- |
| a directory with `.lyu` files is a Lyuba package; the generated `.go` next to them is ignored | `internal/fetch/lyuba.go`, `internal/fetch/package.go` |
| Lyuba docs: `LYU1` encoding (the sources), rendered by the compiler's parser and `lyuba/doc` | `internal/godoc/lyuba.go`, `encode.go`, `render.go` |
| a module without `.lyu` files is excluded (403), quietly (no error in the logs) | `internal/fetch/lyuba.go`, `internal/queue/pgqueue/queue.go` |
| `fetch.LyubaOnly` is enabled by the site's commands only: pkgsite's own tests stay valid | `cmd/worker`, `cmd/frontend`, `cmd/pkgsite` |
| `cmd/pkgsite` reads a local Lyuba module as is (no go/packages) | `cmd/internal/pkgsite/server.go` |
| Lyuba branding: logo, header, footer, home page; no gopher, nothing from Google (logo, Tag Manager, cookie banner) | `static/` |
| install page and scripts: `/install`, `/install.sh`, `/install.ps1` (those of `lyuba/install`) | `internal/frontend/server.go`, `static/frontend/install` |
| `/why`: the design rationale page, in `static/frontend/why` | `static/frontend/why`, `internal/frontend/server.go` |
| the header's active tab follows the page | `internal/frontend/page/page.go`, `static/shared/header` |
| deployment without Google Cloud | `lyuba-deploy/compose.yaml` |

The module path stays `golang.org/x/pkgsite`: renaming it would touch every import and make every rebase painful. `go.mod` depends on `github.com/GlazKrovi/lyuba`; until that repository is public, a `replace` points to `../lyuba` (the repository cloned next to this one).

## Run

Docs of a local Lyuba module, without a database:

```sh
go run ./cmd/pkgsite -open -list=false path/to/module
```

The full site (Postgres, worker, frontend):

```sh
cd lyuba-deploy
docker compose up -d     # http://localhost:8080; the worker reads the index every minute
docker compose down      # stop; the database is kept (pgdata volume)
```

## Rebase on pkgsite

```sh
git remote add upstream https://go.googlesource.com/pkgsite
git fetch upstream && git rebase upstream/master
```

The changes are concentrated in the files above; `go test ./internal/fetch/ -run TestLyubaModule` checks the core. On Windows, some of pkgsite's own tests already fail on upstream (line endings): compare with a `git worktree` of upstream before digging further.
