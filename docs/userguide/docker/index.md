# Self-Hosting with Docker

Clio is distributed as a Docker image. The image contains the compiled application, the admin dashboard, the preview server, and the static site generator. You do not need Go, Node.js, or any build tools on your machine. You only need Docker.

There are two ways to get started. Pick one:

- **[Quick Install](#quick-install)** — A single command that does everything. If it works, skip straight to [Making Your Site Public](#making-your-site-public).
- **[Manual Setup](#manual-setup)** — Step-by-step if you prefer to control each part, or if Quick Install did not work.

Both paths produce the same result: Clio running locally at `http://localhost:8080`.

Once Clio is running, you can optionally make your site public:

| I want to... | Go to |
|---|---|
| A temporary public URL for testing | [Quick Tunnel](#quick-tunnel-temporary-url) |
| A permanent URL with my own domain | [Named Tunnel](#named-tunnel-permanent-custom-domain) |

---

## Quick Install

Requires Docker to be installed and running. On Mac or Windows, install [Docker Desktop](https://www.docker.com/products/docker-desktop/). On Linux, follow the [official guide](https://docs.docker.com/engine/install/).

```bash
bash <(curl -fsSL https://raw.githubusercontent.com/cliossg/clio/main/deploy/install.sh)
```

This creates a `clio` directory, downloads the configuration, generates a session secret, pulls the Docker image, and starts Clio. When it finishes, it prints your dashboard URL and login credentials.

To use a different directory name:

```bash
CLIO_DIR=myblog bash <(curl -fsSL https://raw.githubusercontent.com/cliossg/clio/main/deploy/install.sh)
```

After Quick Install completes, open `http://localhost:8080`, sign in, and you are ready. To make your site public, continue to [Making Your Site Public](#making-your-site-public).

**A note on running scripts from the internet.** The script is short and does exactly four things: checks that Docker is installed, downloads a `docker-compose.yml`, generates an `.env` file with a random secret, and runs `docker compose up -d`. You can [read the source](https://github.com/cliossg/clio/blob/main/deploy/install.sh) before running it. If you prefer to do each step yourself, use the [Manual Setup](#manual-setup) below instead.

---

## Manual Setup

This is the step-by-step alternative to [Quick Install](#quick-install). If you already ran Quick Install successfully, skip to [Making Your Site Public](#making-your-site-public).

### Step 1: Install Docker

If you already have Docker installed, skip to [Step 2](#step-2-create-a-project-folder).

**Mac or Windows**: Download and install [Docker Desktop](https://www.docker.com/products/docker-desktop/). It includes everything you need (Docker Engine and Docker Compose). After installation, open Docker Desktop and wait until it says "Docker is running" in the bottom-left corner.

**Linux**: Follow the [official installation guide](https://docs.docker.com/engine/install/) for your distribution. Docker Compose is included as a plugin. After installation, your user may need to be in the `docker` group:

```bash
sudo usermod -aG docker $USER
```

Log out and log back in for this to take effect.

**Verify Docker is working** by opening a terminal and running:

```bash
docker version
```

You should see version information for both "Client" and "Server". If the Server section shows an error, Docker is not running — start Docker Desktop (Mac/Windows) or start the Docker service (`sudo systemctl start docker` on Linux).

### Step 2: Create a project folder

Create an empty folder anywhere on your computer. This is where Clio will store its configuration and data. The name does not matter.

```bash
mkdir clio && cd clio
```

All commands from this point forward assume you are inside this folder.

### Step 3: Create the docker-compose.yml file

Create a file called `docker-compose.yml` in your project folder with the following content:

```yaml
services:
  app:
    image: ghcr.io/cliossg/clio:latest
    container_name: clio-app
    restart: unless-stopped
    environment:
      CLIO_ENV: prod
      CLIO_LOG_LEVEL: ${LOG_LEVEL:-info}
      CLIO_SERVER_ADDR: ":8080"
      CLIO_DATABASE_PATH: /app/data/db/clio.db
      CLIO_SSG_SITES_PATH: /app/data/sites
      CLIO_SSG_PREVIEW_ADDR: ":3000"
      CLIO_AUTH_SESSION_SECRET: ${SESSION_SECRET}
      CLIO_CREDENTIALS_PATH: /app/data/credentials.txt
    volumes:
      - ./data:/app/data
    ports:
      - "${APP_PORT:-8080}:8080"
      - "${PREVIEW_PORT:-3000}:3000"
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:8080/ping"]
      interval: 30s
      timeout: 3s
      retries: 3
      start_period: 10s
    networks:
      - clio-network

networks:
  clio-network:
    driver: bridge
```

What each setting means:

| Setting | What it does |
|---|---|
| `image: ghcr.io/cliossg/clio:latest` | Downloads the latest Clio image from GitHub Container Registry |
| `restart: unless-stopped` | Restarts the container automatically if it crashes or if your machine reboots |
| `CLIO_SERVER_ADDR: ":8080"` | The admin dashboard listens on port 8080 inside the container |
| `CLIO_SSG_PREVIEW_ADDR: ":3000"` | The preview server (your public site) listens on port 3000 inside the container |
| `CLIO_AUTH_SESSION_SECRET` | A random string used to sign login cookies (you generate this in the next step) |
| `CLIO_CREDENTIALS_PATH` | Where the auto-generated admin password is written on first run |
| `volumes: ./data:/app/data` | Maps a `data` folder on your computer to `/app/data` inside the container. This is where your database and site files live. If you delete the container, your data is safe in this folder. |
| `ports: "8080:8080"` | Makes the dashboard accessible at `http://localhost:8080` on your machine |
| `ports: "3000:3000"` | Makes the preview server accessible at `http://localhost:3000` on your machine |

### Step 4: Create the .env file

The `.env` file contains secrets and configuration that you do not want inside `docker-compose.yml`. Create a file called `.env` (note the dot at the beginning) in the same folder:

```bash
SESSION_SECRET=
LOG_LEVEL=info
APP_PORT=8080
PREVIEW_PORT=3000
```

Now generate a random session secret. Run one of these commands:

**Mac or Linux:**

```bash
openssl rand -base64 32
```

**Windows (PowerShell):**

```powershell
[Convert]::ToBase64String((1..32 | ForEach-Object { Get-Random -Maximum 256 }) -as [byte[]])
```

**If none of the above work**, any long random string will do. Go to a password generator website, generate a 40+ character password, and use that.

Copy the output and paste it as the `SESSION_SECRET` value in your `.env` file. The result should look like this (your value will be different):

```bash
SESSION_SECRET=a7Bx9kL2mN4pQ8rT1uW3yZ6cE0fH5jD
LOG_LEVEL=info
APP_PORT=8080
PREVIEW_PORT=3000
```

### Step 5: Start Clio

```bash
docker compose up -d
```

What this does:
- Downloads the Clio image from the internet (first time only, may take a minute)
- Creates and starts the container in the background (`-d` means "detached")

**Verify it is running:**

```bash
docker compose ps
```

You should see `clio-app` with status `Up` or `healthy`. If the status says `starting`, wait 10 seconds and run it again. If it says `Restarting` or `Exit`, jump to [Troubleshooting](#troubleshooting).

### Step 6: Get your login credentials

On first run, Clio creates an admin account and writes the username and password to a file. Read it:

**Mac or Linux:**

```bash
cat ./data/credentials.txt
```

**Windows (PowerShell):**

```powershell
Get-Content .\data\credentials.txt
```

You will see something like:

```
Username: admin
Password: xK9mL2pQ4rT7uW
```

### Step 7: Open the dashboard

Open your browser and go to:

```
http://localhost:8080
```

Sign in with the credentials from the previous step. After signing in, go to your profile (top-right corner) and change the default password.

### Step 8: Preview your site

After creating a site and some content in the dashboard, generate it (the "Generate" button). Then open:

```
http://localhost:3000
```

This is your generated site, served by the preview server. Right now it is only accessible from your machine. The next sections explain how to make it public.

---

## Making Your Site Public

The admin dashboard (`localhost:8080`) is restricted to your machine for security. The preview server (`localhost:3000`) is what you expose to the internet. There are two options depending on whether you want a temporary URL for testing or a permanent custom domain.

### Quick Tunnel (temporary URL)

A Cloudflare Quick Tunnel gives you a public `https://something.trycloudflare.com` URL that anyone on the internet can visit. The URL changes every time the container restarts, so this is only for testing or demos.

You do not need a Cloudflare account for this.

#### Step 1: Replace your docker-compose.yml

Replace the entire contents of your `docker-compose.yml` with this:

```yaml
services:
  app:
    image: ghcr.io/cliossg/clio:latest
    container_name: clio-app
    restart: unless-stopped
    environment:
      CLIO_ENV: prod
      CLIO_LOG_LEVEL: ${LOG_LEVEL:-info}
      CLIO_SERVER_ADDR: ":8080"
      CLIO_DATABASE_PATH: /app/data/db/clio.db
      CLIO_SSG_SITES_PATH: /app/data/sites
      CLIO_SSG_PREVIEW_ADDR: ":3000"
      CLIO_AUTH_SESSION_SECRET: ${SESSION_SECRET}
      CLIO_CREDENTIALS_PATH: /app/data/credentials.txt
    volumes:
      - ./data:/app/data
    ports:
      - "${APP_PORT:-8080}:8080"
      - "${PREVIEW_PORT:-3000}:3000"
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:8080/ping"]
      interval: 30s
      timeout: 3s
      retries: 3
      start_period: 10s
    networks:
      - clio-network

  quick-tunnel:
    image: cloudflare/cloudflared:latest
    container_name: clio-quick-tunnel
    restart: unless-stopped
    depends_on:
      app:
        condition: service_healthy
    command: tunnel --url http://app:3000
    networks:
      - clio-network

networks:
  clio-network:
    driver: bridge
```

The new `quick-tunnel` service runs `cloudflared`, a small program from Cloudflare that creates an encrypted tunnel from the internet to your container. The `depends_on` section makes it wait until Clio is healthy before starting. The `command: tunnel --url http://app:3000` tells it to forward internet traffic to the Clio preview server.

Inside Docker, containers on the same network can reach each other by service name. So `http://app:3000` means "port 3000 on the `app` container". This is not `localhost` — it is an internal Docker address.

#### Step 2: Restart

```bash
docker compose down && docker compose up -d
```

Wait about 15 seconds for both containers to start and the tunnel to establish.

#### Step 3: Get your public URL

```bash
docker compose logs quick-tunnel 2>/dev/null | grep -oE "https://[a-z0-9-]+\.trycloudflare\.com" | tail -1
```

This prints a URL like `https://random-words-here.trycloudflare.com`. Open it in your browser (or send it to someone). You should see your generated site.

If nothing is printed, the tunnel may still be starting. Wait a few seconds and try again, or check the full logs:

```bash
docker compose logs quick-tunnel
```

#### Step 4: Verify

1. Open the tunnel URL in your browser — you should see your generated site
2. Open `http://localhost:8080` — the dashboard still works locally
3. The tunnel URL only exposes the preview server (port 3000). The dashboard is never accessible from the internet.

The quick tunnel URL changes every time you restart the containers. If you want a permanent URL with your own domain, continue to [Named Tunnel](#named-tunnel-permanent-custom-domain).

---

### Named Tunnel (permanent custom domain)

A named tunnel gives you a permanent public URL using your own domain (e.g. `myblog.com`). It requires a free Cloudflare account and a domain whose DNS is managed by Cloudflare.

#### Prerequisites

- A domain you own (you can buy one from any registrar)
- The domain's nameservers pointed to Cloudflare (Cloudflare will guide you through this when you add the domain)

#### Step 1: Set up Cloudflare

1. Go to [dash.cloudflare.com](https://dash.cloudflare.com) and create an account (free)
2. Click **Add a site** and enter your domain
3. Select the **Free** plan
4. Cloudflare will give you two nameservers (e.g. `anna.ns.cloudflare.com`, `bob.ns.cloudflare.com`). Go to your domain registrar and update the nameservers to these values. This can take up to 24 hours to propagate, but usually happens within minutes.
5. Wait until Cloudflare shows your domain as **Active**

#### Step 2: Create a tunnel

1. In the Cloudflare dashboard, go to **Zero Trust** (left sidebar) — if asked, pick the free plan and accept
2. Go to **Networks** → **Tunnels**
3. Click **Create a tunnel**
4. Choose **Cloudflared** as the connector type
5. Give the tunnel a name (e.g. "clio")
6. On the "Install and run a connector" page, **do not install anything**. Instead, look for the token. It is displayed in a command like `cloudflared service install <TOKEN>`. Copy just the token (the long string after `install`).
7. Click **Next**
8. Configure the public hostname:
   - **Subdomain**: leave empty to use your root domain, or enter a subdomain like `www`
   - **Domain**: select your domain from the dropdown
   - **Service type**: `HTTP`
   - **URL**: `app:3000`
9. Click **Save tunnel**

#### Step 3: Add the tunnel token to your .env

Open your `.env` file and add the token:

```bash
SESSION_SECRET=your-existing-secret-here
LOG_LEVEL=info
APP_PORT=8080
PREVIEW_PORT=3000
CLOUDFLARE_TUNNEL_TOKEN=eyJhIjoiYb...your-long-token-here
```

#### Step 4: Replace your docker-compose.yml

Replace the entire contents of your `docker-compose.yml` with this:

```yaml
services:
  app:
    image: ghcr.io/cliossg/clio:latest
    container_name: clio-app
    restart: unless-stopped
    environment:
      CLIO_ENV: prod
      CLIO_LOG_LEVEL: ${LOG_LEVEL:-info}
      CLIO_SERVER_ADDR: ":8080"
      CLIO_DATABASE_PATH: /app/data/db/clio.db
      CLIO_SSG_SITES_PATH: /app/data/sites
      CLIO_SSG_PREVIEW_ADDR: ":3000"
      CLIO_AUTH_SESSION_SECRET: ${SESSION_SECRET}
      CLIO_CREDENTIALS_PATH: /app/data/credentials.txt
    volumes:
      - ./data:/app/data
    ports:
      - "${APP_PORT:-8080}:8080"
      - "${PREVIEW_PORT:-3000}:3000"
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:8080/ping"]
      interval: 30s
      timeout: 3s
      retries: 3
      start_period: 10s
    networks:
      - clio-network

  tunnel:
    image: cloudflare/cloudflared:latest
    container_name: clio-tunnel
    restart: unless-stopped
    depends_on:
      app:
        condition: service_healthy
    command: tunnel run
    environment:
      TUNNEL_TOKEN: ${CLOUDFLARE_TUNNEL_TOKEN}
    networks:
      - clio-network

networks:
  clio-network:
    driver: bridge
```

The `tunnel` service is similar to the quick tunnel, but instead of `tunnel --url ...` it uses `tunnel run` with your token. Cloudflare already knows from the dashboard configuration where to route traffic (to `app:3000`). The `TUNNEL_TOKEN` environment variable authenticates this container as your tunnel connector.

#### Step 5: Start

```bash
docker compose down && docker compose up -d
```

#### Step 6: Verify

1. Wait about 30 seconds for the tunnel to connect
2. Check the tunnel status: `docker compose logs tunnel` — look for "Connection registered" or "Registered tunnel connection"
3. Open your domain (e.g. `https://myblog.com`) in a browser — you should see your generated site
4. The dashboard is still at `http://localhost:8080` on your machine

If the site does not load, check:
- Tunnel logs: `docker compose logs tunnel`
- Cloudflare dashboard: **Zero Trust** → **Tunnels** — the tunnel should show as "Healthy"
- DNS: the domain should have a CNAME record pointing to your tunnel (Cloudflare creates this automatically)

Your site is now live. The tunnel routes traffic from the internet to the Clio preview server. This includes both the static site and the [Contact Forms](../forms/index.md) endpoint, so contact forms work without any extra configuration.

---

## Architecture

Clio runs two HTTP servers inside the container:

| Server | Port | Purpose | Access |
|--------|------|---------|--------|
| Admin dashboard | `:8080` | Content management, settings, messages | Localhost only |
| Preview server | `:3000` | Serves the generated site + forms API | Public (via tunnel) |

The admin dashboard is intentionally restricted to localhost. You access it from your browser on the same machine. The preview server is what gets exposed to the internet through the tunnel.

```
Your machine
├── http://localhost:8080  → Admin dashboard (only you)
└── http://localhost:3000  → Preview server (also you, for testing)

Internet (via Cloudflare Tunnel)
└── https://myblog.com     → Preview server (:3000)
    ├── /                  → Your generated static site
    └── /api/v1/forms/     → Contact form submissions
```

---

## Data and Persistence

All data lives in the `./data` directory on your machine, mounted into the container at `/app/data`:

```
data/
├── db/
│   └── clio.db          # SQLite database (content, settings, users)
├── sites/
│   └── your-site/       # Generated HTML, images, workspace
└── credentials.txt      # Auto-generated admin credentials (first run)
```

**Your data is safe if you delete the container.** The container is just the application. Your content, settings, and generated site live in the `data` folder on your machine. As long as you do not delete that folder, you can destroy and recreate the container without losing anything.

**Backup**: Copy the entire `./data` directory. That is your complete backup — database, generated sites, everything.

---

## Updating

When a new version of Clio is released, update with two commands:

```bash
docker compose pull
docker compose up -d
```

The first command downloads the latest image. The second restarts the container with the new image. Database migrations run automatically on startup. Your data is preserved.

**Verify after updating:**

```bash
docker compose ps
```

The container should show `healthy` status within 30 seconds.

---

## Stopping and Starting

**Stop Clio** (keeps data):

```bash
docker compose down
```

**Start again:**

```bash
docker compose up -d
```

**Restart** (stop and start in one command):

```bash
docker compose restart
```

---

## Troubleshooting

### Can't access the dashboard

The dashboard only works from `localhost:8080`. If you are trying to access it from another machine on your network, it will not work — this is intentional for security.

Check the container is running:

```bash
docker compose ps
```

If it shows `Exit` or `Restarting`, check the logs:

```bash
docker compose logs app --tail 50
```

### Tunnel URL not working

Wait about 15-30 seconds after starting for the tunnel to connect.

Check tunnel logs:

```bash
docker compose logs quick-tunnel
# or, for named tunnels:
docker compose logs tunnel
```

Look for error messages. Common issues:
- **"failed to connect"**: The `app` container may not be healthy yet. Wait and try again.
- **Invalid token**: Double-check the `CLOUDFLARE_TUNNEL_TOKEN` in your `.env` file.

Verify the tunnel routes to `app:3000` (not `app:8080` or `localhost:3000`).

### Permission errors on data directory

The container runs as user ID 1000. If the `data` directory was created by root or another user:

```bash
sudo chown -R 1000:1000 ./data
```

### Container keeps restarting

```bash
docker compose logs app --tail 50
```

Common causes:
- **Missing or empty `SESSION_SECRET`**: Check your `.env` file. The `SESSION_SECRET` line must have a value.
- **Port already in use**: Another program is using port 8080 or 3000. Change `APP_PORT` or `PREVIEW_PORT` in `.env` to different numbers (e.g. `APP_PORT=9080`).
- **Database locked**: Another instance of Clio or another program has the database open. Stop any other instances first.

### "image not found" or "manifest unknown"

The image has not been published yet, or there is a typo in the image name. Verify with:

```bash
docker pull ghcr.io/cliossg/clio:latest
```

If this fails, the image may not be published yet. See [Building from Source](#building-from-source) as an alternative.

---

## Building from Source

The instructions above use the pre-built image from GitHub Container Registry. If you prefer to build the image yourself from the source code (for example, if the pre-built image is not available yet, or if you have made local changes):

1. Install [Git](https://git-scm.com/downloads) if you don't have it
2. Clone the repository and enter it:

```bash
git clone https://github.com/cliossg/clio.git
cd clio
```

3. Create the environment file:

```bash
cp .env.example .env
```

4. Generate a session secret and paste it as `SESSION_SECRET` in `.env`:

```bash
openssl rand -base64 32
```

5. Build and start:

```bash
docker compose up -d --build
```

This compiles Clio inside the container. You need Git installed, but not Go or any other build tools. Everything else (dashboard access, tunnels, data persistence, updating) works the same as described above. To add a tunnel, follow the same tunnel instructions but add `--build` to the `docker compose up` commands.
