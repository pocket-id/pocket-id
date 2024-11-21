# <div align="center"><img  src="https://github.com/user-attachments/assets/4ceb2708-9f29-4694-b797-be833efce17d" width="100"/> </br>Pocket ID</div>

Pocket ID is a simple OIDC provider that allows users to authenticate with their passkeys to your services.

<img src="https://github.com/user-attachments/assets/96ac549d-b897-404a-8811-f42b16ea58e2" width="1200"/>

The goal of Pocket ID is to be a simple and easy-to-use. There are other self-hosted OIDC providers like [Keycloak](https://www.keycloak.org/) or [ORY Hydra](https://www.ory.sh/hydra/) but they are often too complex for simple use cases.

Additionally, what makes Pocket ID special is that it only supports [passkey](https://www.passkeys.io/) authentication, which means you don’t need a password. Some people might not like this idea at first, but I believe passkeys are the future, and once you try them, you’ll love them. For example, you can now use a physical Yubikey to sign in to all your self-hosted services easily and securely.

## Setup

> [!WARNING]  
> Pocket ID is in its early stages and may contain bugs. There might be OIDC features that are not yet implemented. If you encounter any issues, please open an issue.

### Before you start

Pocket ID requires a [secure context](https://developer.mozilla.org/en-US/docs/Web/Security/Secure_Contexts), meaning it must be served over HTTPS. This is necessary because Pocket ID uses the [WebAuthn API](https://developer.mozilla.org/en-US/docs/Web/API/Web_Authentication_API) which requires a secure context.

### Installation with Docker (recommended)

1. Download the `docker-compose.yml` and `.env` file:

   ```bash
    curl -O https://raw.githubusercontent.com/stonith404/pocket-id/main/docker-compose.yml

    curl -o .env https://raw.githubusercontent.com/stonith404/pocket-id/main/.env.example
   ```

2. Edit the `.env` file so that it fits your needs. See the [environment variables](#environment-variables) section for more information.
3. Run `docker compose up -d`

You can now sign in with the admin account on `http://localhost/login/setup`.

### Unraid

Pocket ID is available as a template on the Community Apps store.

### Stand-alone Installation

Required tools:

- [Node.js](https://nodejs.org/en/download/) >= 20
- [Go](https://golang.org/doc/install) >= 1.23
- [Git](https://git-scm.com/downloads)
- [PM2](https://pm2.keymetrics.io/)
- [Caddy](https://caddyserver.com/docs/install) (optional)

1. Copy the `.env.example` file in the `frontend` and `backend` folder to `.env` and change it so that it fits your needs.

   ```bash
   cp frontend/.env.example frontend/.env
   cp backend/.env.example backend/.env
   ```

2. Run the following commands:

   ```bash
   git clone https://github.com/stonith404/pocket-id
   cd pocket-id

   # Checkout the latest version
   git fetch --tags && git checkout $(git describe --tags `git rev-list --tags --max-count=1`)

   # Start the backend
   cd backend/cmd
   go build -o ../pocket-id-backend
   cd ..
   pm2 start pocket-id-backend --name pocket-id-backend

   # Optional: Download the GeoLite2 city database.
   # If not downloaded the ip location in the audit log will be empty.
   MAXMIND_LICENSE_KEY=<your-key> sh scripts/download-ip-database.sh

   # Start the frontend
   cd ../frontend
   npm install
   npm run build
   pm2 start --name pocket-id-frontend --node-args="--env-file .env" build/index.js

   # Optional: Start Caddy (You can use any other reverse proxy)
   cd ..
   pm2 start --name pocket-id-caddy caddy -- run --config Caddyfile
   ```

You can now sign in with the admin account on `http://localhost/login/setup`.

### Nginx Reverse Proxy

To use Nginx in front of Pocket ID, add the following configuration to increase the header buffer size because, as SvelteKit generates larger headers.

```nginx
proxy_busy_buffers_size   512k;
proxy_buffers   4 512k;
proxy_buffer_size   256k;
```

## Proxy Services with Pocket ID

As the goal of Pocket ID is to stay simple, we don't have a built-in proxy provider. However, you can use [OAuth2 Proxy](https://oauth2-proxy.github.io/) to add authentication to your services that don't support OIDC.

See the [guide](docs/proxy-services.md) for more information.

## Update

#### Docker

```bash
docker compose pull
docker compose up -d
```

#### Stand-alone

1. Stop the running services:
   ```bash
   pm2 delete pocket-id-backend pocket-id-frontend pocket-id-caddy
   ```
2. Run the following commands:

   ```bash
   cd pocket-id

   # Checkout the latest version
   git fetch --tags && git checkout $(git describe --tags `git rev-list --tags --max-count=1`)

   # Start the backend
   cd backend/cmd
   go build -o ../pocket-id-backend
   cd ..
   pm2 start pocket-id-backend --name pocket-id-backend

   # Optional: Update the GeoLite2 city database
   MAXMIND_LICENSE_KEY=<your-key> sh scripts/download-ip-database.sh

   # Start the frontend
   cd ../frontend
   npm install
   npm run build
   pm2 start build/index.js --name pocket-id-frontend

   # Optional: Start Caddy (You can use any other reverse proxy)
   cd ..
   pm2 start caddy --name pocket-id-caddy -- run --config Caddyfile
   ```

## Environment variables

| Variable               | Default Value           | Recommended to change | Description                                                                                                                                                                           |
| ---------------------- | ----------------------- | --------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `PUBLIC_APP_URL`       | `http://localhost`      | yes                   | The URL where you will access the app.                                                                                                                                                |
| `TRUST_PROXY`          | `false`                 | yes                   | Whether the app is behind a reverse proxy.                                                                                                                                            |
| `PUID`                  and `PGID` | `1000`                  | yes                   | The user and group ID of the user who should run Pocket ID inside the Docker container and owns the files that are mounted with the volume. You can get the `PUID` and `GUID` of your user on your host machine by using the command `id`. For more information see [this article](https://docs.linuxserver.io/general/understanding-puid-and-pgid/#using-the-variables).                                                                                                                       |
| `DB_PATH`              | `data/pocket-id.db`     | no                    | The path to the SQLite database.                                                                                                                                                      |
| `UPLOAD_PATH`          | `data/uploads`          | no                    | The path where the uploaded files are stored.                                                                                                                                         |
| `INTERNAL_BACKEND_URL` | `http://localhost:8080` | no                    | The URL where the backend is accessible.                                                                                                                                              |
| `CADDY_PORT`           | `80`                    | no                    | The port on which Caddy should listen. Caddy is only active inside the Docker container. If you want to change the exposed port of the container then you sould change this variable. |
| `PORT`                 | `3000`                  | no                    | The port on which the frontend should listen.                                                                                                                                         |
| `BACKEND_PORT`         | `8080`                  | no                    | The port on which the backend should listen.                                                                                                                                          |

## Contribute

You're very welcome to contribute to Pocket ID! Please follow the [contribution guide](/CONTRIBUTING.md) to get started.
