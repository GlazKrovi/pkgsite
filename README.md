# pkgsite (Lyuba fork)

The package site of the [Lyuba](https://github.com/GlazKrovi/lyuba) language, served at pkg.lyuba.dev. It is a fork of [pkgsite](https://go.googlesource.com/pkgsite), the source of pkg.go.dev, and only indexes modules that contain `.lyu` files: pure Go modules are excluded.

Lyuba is an independent project, not affiliated with, endorsed by or sponsored by Google LLC or the Go project. "Go" and the Go gopher/logo are trademarks of Google LLC (see [go.dev/brand](https://go.dev/brand)); the Go logo and gopher are not used here. The fork is named "pkgsite (Lyuba fork)" and the site "Lyuba".

What the fork changes, how to run it, deploy it and rebase it on upstream: [FORK.md](FORK.md).

## Documentation server

`cmd/pkgsite` serves the docs of a local Lyuba module:

```sh
go run ./cmd/pkgsite -open -list=false path/to/module
```

## Full site

Postgres, worker and frontend, through docker compose:

```sh
cd lyuba-deploy
docker compose up -d     # http://localhost:8080
```

## API

The REST API of upstream is kept: `/v1/package/{path}`, `/v1/module/{path}`, `/v1/versions/{path}`, `/v1/packages/{path}`, `/v1/search`, `/v1/symbols/{path}`, `/v1/imported-by/{path}`, `/v1/vulns/{path}`.

## License

BSD-3-Clause, the license of upstream pkgsite: see [LICENSE](LICENSE) ("The Go Authors" copyright kept, Lyuba modifications added) and [PATENTS](PATENTS). As the license states, the names of Google LLC and of the Go contributors may not be used to endorse or promote this fork without specific prior written permission.

[CONTRIBUTING.md](CONTRIBUTING.md) is upstream's and refers to the Go project's process (Gerrit, CLA): it does not apply to this fork.
