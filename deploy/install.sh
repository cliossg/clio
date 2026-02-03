#!/usr/bin/env bash
set -euo pipefail

REPO="cliossg/clio"
BRANCH="main"
BASE_URL="https://raw.githubusercontent.com/${REPO}/${BRANCH}/deploy"
INSTALL_DIR="${CLIO_DIR:-clio}"
APP_PORT=8080
PREVIEW_PORT=3000

while [[ $# -gt 0 ]]; do
    case $1 in
        --app-port) APP_PORT="$2"; shift 2 ;;
        --preview-port) PREVIEW_PORT="$2"; shift 2 ;;
        --dir) INSTALL_DIR="$2"; shift 2 ;;
        *) echo "Unknown option: $1"; exit 1 ;;
    esac
done

echo "Installing Clio into ./${INSTALL_DIR}"
echo ""

if ! command -v docker &> /dev/null; then
    echo "Error: Docker is not installed."
    echo "Install it from https://docs.docker.com/get-docker/"
    exit 1
fi

if ! docker info &> /dev/null 2>&1; then
    echo "Error: Docker is not running."
    echo "Start Docker Desktop or run: sudo systemctl start docker"
    exit 1
fi

mkdir -p "${INSTALL_DIR}/data/db" "${INSTALL_DIR}/data/sites"
cd "${INSTALL_DIR}"

echo "Downloading docker-compose.yml..."
curl -fsSL "${BASE_URL}/docker-compose.yml" -o docker-compose.yml

SECRET=$(openssl rand -base64 32 2>/dev/null || head -c 32 /dev/urandom | base64)

cat > .env <<EOF
SESSION_SECRET=${SECRET}
LOG_LEVEL=info
APP_PORT=${APP_PORT}
PREVIEW_PORT=${PREVIEW_PORT}
CLIO_UID=$(id -u)
CLIO_GID=$(id -g)
EOF

echo "Starting Clio..."
docker compose up -d

echo ""
echo "Waiting for Clio to start..."
for i in $(seq 1 30); do
    if [ -f ./data/credentials.txt ]; then
        break
    fi
    sleep 1
done

echo ""
echo "Dashboard: http://localhost:${APP_PORT}"
if [ -f ./data/credentials.txt ]; then
    echo ""
    cat ./data/credentials.txt
    echo ""
    echo "Sign in and change the default password."
else
    echo "Credentials: cat ${INSTALL_DIR}/data/credentials.txt"
fi

echo ""
echo "Preview server: http://localhost:${PREVIEW_PORT}"
echo "Data directory:  ${INSTALL_DIR}/data/"
echo ""
echo "To stop:   cd ${INSTALL_DIR} && docker compose down"
echo "To update: cd ${INSTALL_DIR} && docker compose pull && docker compose up -d"
