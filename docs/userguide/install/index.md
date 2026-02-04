# Installation

There are three ways to install Clio. Pick the one that fits your situation:

| I want to... | Method |
|---|---|
| Get running as fast as possible | [Quick Install](#quick-install) (single command, uses Docker) |
| Clone the repo and run from source | [Clone and Run](#clone-and-run) (uses Docker to build locally) |
| Build and run a native binary | [Build from Source](#build-from-source) (no Docker needed) |

The first two methods use Docker. The third produces a standalone binary and only requires Go and a C compiler.

---

## Quick Install

This is the fastest path. A single command downloads the configuration, pulls the Docker image, and starts Clio.

```bash
bash <(curl -fsSL https://raw.githubusercontent.com/cliossg/clio/main/deploy/install.sh)
```

When it finishes, it prints your dashboard URL and login credentials. Open `http://localhost:8080` and sign in.

You can customize the ports and directory:

```bash
bash <(curl -fsSL https://raw.githubusercontent.com/cliossg/clio/main/deploy/install.sh) \
  --app-port 9090 \
  --preview-port 9091 \
  --dir myblog
```

All parameters are optional and can be combined.

**What happens next**: Clio is running locally. If you want to expose it to the internet, configure tunnels, or understand the Docker setup in detail, see the [Self-Hosting with Docker](../docker/index.md) guide.

---

## Clone and Run

This method clones the repository and builds the Docker image locally from source. Useful if you want to inspect the code, make changes, or if the pre-built image is not available.

### Prerequisites

- [Git](https://git-scm.com/downloads)
- [Docker](https://www.docker.com/products/docker-desktop/) (includes Docker Compose)
- Make (pre-installed on Mac and Linux; on Windows use [WSL](https://learn.microsoft.com/en-us/windows/wsl/install))

### Step 1: Clone the repository

```bash
git clone https://github.com/cliossg/clio.git
cd clio
```

All commands from this point forward assume you are inside this folder.

### Step 2: Initialize the environment

```bash
make docker-init
```

This creates a `.env` file from the template and generates a random session secret. If `.env` already exists, it does nothing.

### Step 3: Start Clio

```bash
make docker-up
```

This builds the Docker image from source and starts the container. The first build downloads Go dependencies, so it may take a moment. Subsequent starts are fast.

### Step 4: Get your login credentials

On first run, Clio creates an admin account and writes the credentials to a file:

```bash
cat data/credentials.txt
```

### Step 5: Open the dashboard

Go to `http://localhost:8080`, sign in with the credentials from the previous step, and change the default password.

### Managing the containers

| Task | Command |
|---|---|
| Stop Clio | `make docker-down` |
| Start again | `make docker-up` |
| View logs | `make docker-logs` |
| Container status | `make docker-ps` |
| Stop and delete all data | `make docker-reset` |

### Start with a public URL

If you want a temporary public URL for testing:

```bash
make docker-up-quick-tunnel
```

This starts Clio and a Cloudflare tunnel that gives you a public `https://something.trycloudflare.com` URL. The URL changes on every restart.

For a permanent URL with your own domain, see the [Named Tunnel](../docker/index.md#named-tunnel-permanent-custom-domain) section in the Docker guide.

---

## Build from Source

This method compiles Clio into a native binary and runs it directly on your machine without Docker. This is the typical setup for development or for running Clio on machines where Docker is not available.

### Prerequisites

- [Git](https://git-scm.com/downloads)
- [Go](https://go.dev/dl/) 1.23 or later
- A C compiler (GCC or Clang). Clio uses CGO for SQLite.
- Make

**Mac**: Xcode Command Line Tools provides both Make and a C compiler. Install with `xcode-select --install`. Install Go from [go.dev](https://go.dev/dl/) or with `brew install go`.

**Linux (Debian/Ubuntu)**:

```bash
sudo apt install build-essential golang
```

**Linux (Fedora)**:

```bash
sudo dnf install gcc make golang
```

Verify your Go version is 1.23 or later:

```bash
go version
```

### Step 1: Clone the repository

```bash
git clone https://github.com/cliossg/clio.git
cd clio
```

### Step 2: Build

```bash
make build
```

This compiles the binary to `build/clio`.

### Step 3: Run

For development (verbose logging, auto-creates workspace in `_workspace/`):

```bash
make run
```

For production:

```bash
make run-prod
```

Both commands build and start Clio in one step. The dashboard is at `http://localhost:8080` and the preview server at `http://localhost:3000`.

On first run, Clio prints the admin credentials to the terminal. Sign in and change the default password.

### Useful commands

| Task | Command |
|---|---|
| Build without running | `make build` |
| Run in dev mode | `make run` |
| Run in prod mode | `make run-prod` |
| Stop | `make kill` |
| Run tests | `make test` |
| Run all quality checks | `make check` |
| Reset dev database | `make dev-db-reset` |

### Data location

When running natively, Clio stores its database and generated sites in `_workspace/` (dev mode) or the configured paths (prod mode). See the environment variables below to customize this.

### Environment variables

You can configure Clio by setting environment variables before running:

| Variable | Default | Description |
|---|---|---|
| `CLIO_ENV` | `dev` | `dev` or `prod` |
| `CLIO_LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error` |
| `CLIO_SERVER_ADDR` | `:8080` | Dashboard listen address |
| `CLIO_DATABASE_PATH` | (auto) | Path to SQLite database file |
| `CLIO_SSG_SITES_PATH` | (auto) | Path to generated sites directory |
| `CLIO_SSG_PREVIEW_ADDR` | `:3000` | Preview server listen address |
| `CLIO_AUTH_SESSION_SECRET` | (auto in dev) | Secret for signing session cookies |

---

## What's Next

Once Clio is running:

1. Create a site in the dashboard
2. Add sections and write content
3. Generate your static site
4. Make it public. See [Self-Hosting with Docker](../docker/index.md) for tunneling options.

For importing existing Markdown files, see the [Import](../import/index.md) guide.
