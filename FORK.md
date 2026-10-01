# pkg.lyuba.dev : fork de pkgsite

Le site de packages du langage [Lyuba](https://github.com/GlazKrovi/lyuba), forké de [pkgsite](https://go.googlesource.com/pkgsite) (pkg.go.dev, licence BSD-3-Clause, voir `LICENSE` et `PATENTS`, conservés). Il lit le même index et le même proxy que pkg.go.dev, et ne garde que les modules qui contiennent des `.lyu`.

## Ce qui change par rapport à pkgsite

| | fichiers |
| --- | --- |
| un dossier avec des `.lyu` est un package Lyuba ; le `.go` généré à côté est ignoré | `internal/fetch/lyuba.go`, `internal/fetch/package.go` |
| doc Lyuba : encodage `LYU1` (les sources), rendue par le parser du compilateur et `lyuba/doc` | `internal/godoc/lyuba.go`, `encode.go`, `render.go` |
| module sans `.lyu` : exclu (403), sans erreur dans les journaux | `internal/fetch/lyuba.go`, `internal/queue/pgqueue/queue.go` |
| `fetch.LyubaOnly` activé par les commandes du site seulement : les tests de pkgsite restent valables | `cmd/worker`, `cmd/frontend`, `cmd/pkgsite` |
| `cmd/pkgsite` lit un module Lyuba local tel quel (pas de go/packages) | `cmd/internal/pkgsite/server.go` |
| marque Lyuba : logo, en-tête, pied de page, accueil en français ; ni gopher, ni Google (logo, Tag Manager, bandeau cookies) | `static/` |
| page et scripts d'installation : `/install`, `/install.sh`, `/install.ps1` (ceux de `lyuba/install`) | `internal/frontend/server.go`, `static/frontend/install` |
| déploiement sans Google Cloud | `lyuba-deploy/compose.yaml` |

Le chemin de module reste `golang.org/x/pkgsite` : le renommer toucherait chaque import et rendrait chaque rebase sur pkgsite pénible. `go.mod` dépend de `github.com/GlazKrovi/lyuba` ; tant que ce dépôt n'est pas public, un `replace` pointe vers `../lyuba` (le dépôt cloné à côté).

## Lancer

Doc d'un module Lyuba local, sans base de données :

```sh
go run ./cmd/pkgsite -open -list=false chemin/du/module
```

Le site complet (Postgres, worker, frontend) :

```sh
cd lyuba-deploy
docker compose up -d     # http://localhost:8080 ; le worker lit l'index toutes les minutes
docker compose down      # arrêt, la base est gardée (volume pgdata)
```

## Rebaser sur pkgsite

```sh
git remote add upstream https://go.googlesource.com/pkgsite
git fetch upstream && git rebase upstream/master
```

Les changements sont concentrés dans les fichiers ci-dessus ; `go test ./internal/fetch/ -run TestLyubaModule` vérifie le cœur. Sous windows, certains tests de pkgsite échouent déjà sur l'amont (fins de ligne) : comparer avec un `git worktree` de l'amont avant de chercher plus loin.
