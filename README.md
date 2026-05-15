# ToolBox

ToolBox est organise pour accueillir un front React/TypeScript, une API HTTP Go, des modules Go/Cobra executables seuls, et des donnees locales par module.

## Structure

- `apps/api` : back HTTP/API Go. Les routes modulaires sont exposees dans `apps/api/internal/http`.
- `apps/web` : front React/TypeScript.
- `apps/web-server` : serveur Go statique pour servir le build du front.
- `modules/test-sheet` : module Cobra pour les fiches de test, avec SQLite dans `BDD/test-sheet.db` a cote de l'executable lance.
- `modules/test-env` : module Cobra pour les maquettes de test, avec configuration dans `config/test-env.json` a cote de l'executable lance.
- `pkg/modulecontract` : contrat JSON partage entre API, front et modules.
- `BDD` : bases SQLite runtime creees a cote des executables lances.
- `_build` : binaires generes.

## API

Lancer l'API en developpement :

```bash
go run ./apps/api/cmd/api server
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

En mode dev, Webpack reste expose sur `http://localhost:3000`. Le fichier
`apps/web/public/toolbox.config.js` est servi par le dev server et configure le
front pour appeler directement `http://localhost:20250/api`. Les CORS de l'API
autorisent `http://localhost:3000` par defaut.

Builder le front :

```bash
go run ./tools/build web
```

## Web Server et reseau

Le navigateur charge le front depuis `web-server-toolbox`. Le front lit au
demarrage une config runtime servie par `/toolbox.config.js`, puis appelle une
URL relative `/api`. Le web-server reverse-proxy ensuite `/api/*` vers l'API
interne. En usage serveur, les CORS ne sont donc pas necessaires pour le front :
un seul service, le web-server, doit etre expose sur le reseau.

Le serveur statique embarque le build React dans l'executable Go avec
`go:embed`. Le build `web-server` compile donc d'abord le front, copie
`apps/web/dist` dans `apps/web-server/cmd/dist`, puis produit un binaire
autonome.

```bash
go run ./tools/build web-server
./_build/web-server-toolbox.exe start
```

En developpement, il est aussi possible de servir un dossier `dist` depuis le disque :

```bash
go run ./apps/web-server/cmd/web-server start --dist ./apps/web/dist
```

La config runtime retournee par le web-server est :

```js
window.TOOLBOX = {
  services: {
    api: {
      url: "/api"
    }
  }
};
```

### Configuration

ToolBox charge les valeurs dans cet ordre : valeurs par defaut, fichier
`toolbox.cfg` s'il existe, variables d'environnement, puis flags Cobra.

Valeurs disponibles :

- `TOOLBOX_FQDN` ou `[platform] fqdn`, defaut `localhost`
- `TOOLBOX_PORT` ou `[platform] port`, defaut `20251`
- `TOOLBOX_TLS` ou `[platform] tls`, defaut `false`
- `TOOLBOX_BIND` ou `[platform] bind`, defaut vide
- `TOOLBOX_API_HOST` ou `[services.api] host`, defaut `127.0.0.1:20250`
- `TOOLBOX_CORS_ORIGINS` ou `[cors] origins`, defaut `http://localhost:3000,http://localhost:20251`

Les valeurs runtime sont deduites :

- `WebAddr` vaut `:<port>` si `bind` est vide, sinon `<bind>:<port>`
- `PublicURL` vaut `http(s)://<fqdn>:<port>`
- `APIAddr` vaut `services.api.host`
- `APITarget` vaut `http://<services.api.host>`

Les anciens champs `[web].addr`, `[web].public_url`, `[api].addr` et
`[api].target` restent acceptes comme legacy, mais le fichier recommande ne les
utilise plus.

Commandes Windows possibles :

```bat
set TOOLBOX_BIND=0.0.0.0
set TOOLBOX_PORT=20251
set TOOLBOX_API_HOST=127.0.0.1:20250
```

Flags utiles :

```bash
api-toolbox.exe server --config toolbox.cfg
web-server-toolbox.exe start --config toolbox.cfg --addr 0.0.0.0:20251 --api-target http://127.0.0.1:20250
```

### Mode serveur LAN

Exemple `toolbox.cfg` :

```toml
[platform]
fqdn = "192.168.1.50"
port = 20251
tls = false
bind = "0.0.0.0"

[services.api]
host = "127.0.0.1:20250"

[cors]
origins = [
  "http://localhost:3000"
]
```

Lancement :

```bat
api-toolbox.exe server --config toolbox.cfg
web-server-toolbox.exe start --config toolbox.cfg
```

Depuis un autre poste, ouvrir `http://192.168.1.50:20251`. Le navigateur ne voit
pas `localhost:20250` : il appelle `/api`, et le web-server proxy vers l'API
locale.

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
build.bat module test-env
```

Commandes Go directes :

```bash
go run ./tools/build api
go run ./tools/build web
go run ./tools/build web-server
go run ./tools/build module test-sheet
go run ./tools/build module test-env
go run ./tools/build modules
go run ./tools/build all
```

Depuis un autre dossier, indiquez explicitement la racine :

```powershell
go run C:\chemin\ToolBox\tools\build\main.go --root C:\chemin\ToolBox all
```

L'outil resout la racine via `--root`, puis `TOOLBOX_ROOT`, puis en remontant jusqu'a un `go.mod` contenant `module toolBox`.
