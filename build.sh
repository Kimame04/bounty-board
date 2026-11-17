#!/usr/bin/env bash
set -e

echo "⚔️ [1/2] Building React TypeScript frontend..."
cd frontend
export PATH="/usr/local/bin:$PATH"
npm run build
cd ..

echo "🛡️ [2/2] Compiling standalone Go binary with embedded assets..."
cd backend
export PATH="/opt/homebrew/bin:$PATH"
go build -ldflags="-s -w" -o ../bounty-board-app ./cmd/server/main.go
cd ..

echo "✨ Done! Single-binary executable created: ./bounty-board-app"
ls -lh ./bounty-board-app
