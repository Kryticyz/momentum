#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")" && pwd)"

echo "==> Installing frontend dependencies..."
cd "$REPO_ROOT/apps/dashboard-frontend"
bun install --frozen-lockfile

echo "==> Building frontend (TypeScript + Vite)..."
bun run build

echo "==> Building backend (Go)..."
cd "$REPO_ROOT/services/dashboard-backend"
go build -o momentum-dashboard .

echo ""
echo "Build complete."
echo "  Frontend -> services/dashboard-backend/frontend/dist/"
echo "  Backend  -> services/dashboard-backend/momentum-dashboard"
