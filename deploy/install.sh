#!/usr/bin/env bash
set -euo pipefail

REPO="cliossg/clio"
BRANCH="main"
BASE_URL="https://raw.githubusercontent.com/${REPO}/${BRANCH}/deploy"
INSTALL_DIR="${CLIO_DIR:-clio}"

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

mkdir -p "${INSTALL_DIR}"
cd "${INSTALL_DIR}"

echo "Downloading docker-compose.yml..."
curl -fsSL "${BASE_URL}/docker-compose.yml" -o docker-compose.yml

SECRET=$(openssl rand -base64 32 2>/dev/null || head -c 32 /dev/urandom | base64)

cat > .env <<EOF
SESSION_SECRET=${SECRET}
LOG_LEVEL=info
APP_PORT=8080
PREVIEW_PORT=3000
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
echo "Dashboard: http://localhost:8080"
if [ -f ./data/credentials.txt ]; then
    echo ""
    cat ./data/credentials.txt
    echo ""
    echo "Sign in and change the default password."
else
    echo "Credentials: cat ${INSTALL_DIR}/data/credentials.txt"
fi

echo ""
echo "Preview server: http://localhost:3000"
echo "Data directory:  ${INSTALL_DIR}/data/"
echo ""
echo "To stop:   cd ${INSTALL_DIR} && docker compose down"
echo "To update: cd ${INSTALL_DIR} && docker compose pull && docker compose up -d"
