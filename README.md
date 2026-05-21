# ToolBox

ToolBox est organise pour accueillir un front React/TypeScript, une API HTTP Go, des modules Go/Cobra executables seuls, et une installation runtime autonome.

## Structure

- `apps/api` : back HTTP/API Go. Les routes modulaires sont exposees dans `apps/api/internal/http`.
- `apps/web` : front React/TypeScript.
- `apps/web-server` : serveur Go statique pour servir le build du front.
- `modules/test-sheet` : module Cobra pour les fiches de test.
- `modules/v10-lab` : module Cobra pour les maquettes Gedix V10.
- `pkg/modulecontract` : contrat JSON partage entre API, front et modules.
- `pkg/toolboxruntime` : resolution des chemins runtime installes.
- `_build` : binaires generes.

## Installation Runtime

Le build final produit un installeur autonome :

```bat
build.bat installer
```

ou :

```bash
go run ./tools/build installer
```

Le resultat final visible est `_build/toolbox-setup.exe`. Lancer cet exe installe ToolBox dans `.\ToolBox` par defaut :

```bat
_build\toolbox-setup.exe
```

Pour choisir le dossier parent :

```bat
_build\toolbox-setup.exe --dir C:\Apps
```

L'installation cree cette architecture :

```text
ToolBox/
├─ api-toolbox.exe
├─ web-server-toolbox.exe
├─ toolbox.cfg
└─ modules/
   ├─ test-sheet/
   │  ├─ test-sheet.exe
   │  ├─ data/
   │  │  └─ test-sheet.db
   │  └─ files/
   │     ├─ documents/
   │     └─ runs/
   └─ v10-lab/
      ├─ v10-lab.exe
      ├─ data/
      └─ files/
```

Lors d'une mise a jour, l'installeur remplace les exe et conserve `toolbox.cfg`, `data/` et `files/`. `toolbox.cfg` est cree uniquement s'il n'existe pas, sauf avec `--force-config`. Les donnees utilisateur sont dans `modules/*/data` et `modules/*/files`.

## API

Lancer l'API en developpement :

```bash
go run ./apps/api/cmd/api server
```

En mode installe :

```bat
cd ToolBox
api-toolbox.exe server
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

En mode installe :

```bat
cd ToolBox
web-server-toolbox.exe start
```

Acces :

```text
http://localhost:20251
http://NOM_SERVEUR:20251
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
fqdn = "localhost"
port = 20251
tls = false
bind = "0.0.0.0"

[services.api]
host = "127.0.0.1:20250"
```

La section `[cors]` reste supportee pour le dev Webpack, mais elle n'est pas generee par defaut et n'est pas necessaire en mode serveur same-origin.

Regenerer la config par defaut :

```bat
web-server-toolbox.exe config init --output toolbox.cfg --force
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

go run ./modules/v10-lab/cmd/v10-lab info --json
go run ./modules/v10-lab/cmd/v10-lab products
go run ./modules/v10-lab/cmd/v10-lab actions --product gedix-prod-v10
go run ./modules/v10-lab/cmd/v10-lab validate --config ./examples/v10-lab/ticket-T5808.json
go run ./modules/v10-lab/cmd/v10-lab run --config ./examples/v10-lab/ticket-T5808.json
go run ./modules/v10-lab/cmd/v10-lab register --config ./examples/v10-lab/ticket-T5808.json
go run ./modules/v10-lab/cmd/v10-lab list
```

## V10 Lab

V10 Lab est un generateur et gestionnaire de maquettes V10.

Phase 1 :

- registre de produits
- registre d'actions
- validation de fichier JSON
- execution fictive de pipeline
- base multi-maquettes

Phase 2 - Actions systeme Gedix :

- creation d'une maquette depuis une release ZIP
- detection du dossier Gedix, de `env_*` et de `app_prod`
- generation si besoin puis modification controlee du `gedix.cfg`
- demarrage standard de la maquette
- demarrage avec exclusions pour debug
- lancement de services/connecteurs en debug
- commande manuelle `taskkill gx-*`

Phase 3 - Interface V10 Lab :

- page web V10 Lab
- liste des maquettes
- creation / edition d'une maquette
- edition de la configuration Gedix
- edition services/connecteurs
- builder graphique de pipeline
- validation et lancement depuis le front
- consultation des logs

Notes phase 3 :

- le chemin ZIP release est saisi manuellement pour le moment
- la suppression d'une maquette supprime seulement l'enregistrement V10 Lab, pas le dossier Gedix physique
- `taskkill gx-*` reste une action manuelle avec confirmation

Exemples :

```bash
go run ./modules/v10-lab/cmd/v10-lab products
go run ./modules/v10-lab/cmd/v10-lab actions --product gedix-prod-v10
go run ./modules/v10-lab/cmd/v10-lab actions --product gedix-prod-v10 --json
go run ./modules/v10-lab/cmd/v10-lab validate --config ./examples/v10-lab/ticket-T5808.json
go run ./modules/v10-lab/cmd/v10-lab run --config ./examples/v10-lab/ticket-T5808.json
go run ./modules/v10-lab/cmd/v10-lab register --config ./examples/v10-lab/ticket-T5808.json
go run ./modules/v10-lab/cmd/v10-lab list
go run ./modules/v10-lab/cmd/v10-lab db-templates
go run ./modules/v10-lab/cmd/v10-lab validate --config ./examples/v10-lab/ticket-T5808-system.json
go run ./modules/v10-lab/cmd/v10-lab run --config ./examples/v10-lab/ticket-T5808-system.json
go run ./modules/v10-lab/cmd/v10-lab kill-gx-processes --force
```

Debug Gedix :

```bat
gx-app.exe run -e auth connector-focas-01
gx-auth.exe listen --debug -v2
connector-focas-01\gx-connector.exe listen --debug -v2
```

`kill-gx-processes` est volontairement manuel et ne doit pas etre appele automatiquement par `stop-maquette`.

## Build

Le build principal est un outil Go cross-platform, utilisable depuis Windows sans Git Bash. Pour distribuer ToolBox, utilisez `installer` ou `package`, qui produisent uniquement `_build/toolbox-setup.exe`.

Commandes Windows :

```bat
build.bat all
build.bat api
build.bat web-server
build.bat module test-sheet
build.bat module v10-lab
build.bat installer
build.bat package
```

Commandes Go directes :

```bash
go run ./tools/build api
go run ./tools/build web
go run ./tools/build web-server
go run ./tools/build module test-sheet
go run ./tools/build module v10-lab
go run ./tools/build modules
go run ./tools/build installer
go run ./tools/build package
go run ./tools/build all
```

Depuis un autre dossier, indiquez explicitement la racine :

```powershell
go run C:\chemin\ToolBox\tools\build\main.go --root C:\chemin\ToolBox all
```

L'outil resout la racine via `--root`, puis `TOOLBOX_ROOT`, puis en remontant jusqu'a un `go.mod` contenant `module toolBox`.
