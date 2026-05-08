# ToolBox

ToolBox est organise pour accueillir un front React/TypeScript, une API HTTP Go, des modules Go/Cobra executables seuls, et des donnees locales par module.

## Structure

- `apps/api` : back HTTP/API Go. Les routes modulaires sont exposees dans `apps/api/internal/http`.
- `apps/web` : front React/TypeScript.
- `apps/web-server` : serveur Go statique pour servir le build du front.
- `modules/test-sheet` : module Cobra pour les fiches de test, avec SQLite dans `BDD/test-sheet.db` a cote de l'executable lance.
- `modules/test-env` : module Cobra pour les maquettes de test, avec configuration dans `config/test-env.json` a cote de l'executable lance.
- `modules/legacy-lmba` et `modules/legacy-perso` : anciens modules conserves temporairement.
- `pkg/modulecontract` : contrat JSON partage entre API, front et modules.
- `pkg/logger`, `pkg/paths` : utilitaires partages simples.
- `BDD` : bases SQLite runtime creees a cote des executables lances.
- `_build` : binaires generes.

## API

Lancer l'API en developpement :

```bash
go run ./apps/api/cmd/api
```

Routes disponibles :

- `GET /api/health`
- `GET /api/modules`
- `GET /api/modules/{moduleId}`
- `POST /api/modules/{moduleId}/actions/{actionId}`
- `GET /api/jobs/{jobId}`

## Front

Installer les dependances puis lancer le serveur de dev Webpack :

```bash
cd apps/web
npm install
npm run start
```

Builder le front :

```bash
go run ./tools/build web
```

## Web Server

Le serveur statique embarque le build React dans l'executable Go avec `go:embed`. Le build `web-server` compile donc d'abord le front, copie `apps/web/dist` dans `apps/web-server/cmd/dist`, puis produit un binaire autonome.

```bash
go run ./tools/build web-server
./_build/web-server-toolbox.exe start
```

En developpement, il est aussi possible de servir un dossier `dist` depuis le disque :

```bash
go run ./apps/web-server/cmd/web-server start --dist ./apps/web/dist
```

## Modules CLI

Chaque module suit le squelette Cobra cible :

```bash
go run ./modules/test-sheet/cmd/test-sheet info --json
go run ./modules/test-sheet/cmd/test-sheet actions --json
go run ./modules/test-sheet/cmd/test-sheet run init-db --json

go run ./modules/test-env/cmd/test-env info --json
go run ./modules/test-env/cmd/test-env actions --json
go run ./modules/test-env/cmd/test-env run init-config --json
```

## Build

Les binaires sont produits dans `_build/`. Le build principal est un outil Go cross-platform, utilisable depuis Windows sans Git Bash.

Commandes Windows :

```bat
build.bat all
build.bat api
build.bat web-server
build.bat module test-sheet
```

Commandes PowerShell :

```powershell
.\build.ps1 all
.\build.ps1 api
.\build.ps1 web-server
.\build.ps1 module test-sheet
```

Commandes Go directes :

```bash
go run ./tools/build api
go run ./tools/build web
go run ./tools/build web-server
go run ./tools/build module test-sheet
go run ./tools/build modules
go run ./tools/build all
```

Depuis un autre dossier, indiquez explicitement la racine :

```powershell
go run C:\chemin\ToolBox\tools\build\main.go --root C:\chemin\ToolBox all
```

L'outil resout la racine via `--root`, puis `TOOLBOX_ROOT`, puis en remontant jusqu'a un `go.mod` contenant `module toolBox`. Les scripts `build.sh` et `scripts/build.sh` restent disponibles pour Linux/macOS et deleguent au meme outil Go.
