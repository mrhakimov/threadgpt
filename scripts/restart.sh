#!/bin/bash

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

kill_listener() {
  local port="$1"
  local pids

  pids="$(lsof -tiTCP:${port} -sTCP:LISTEN 2>/dev/null || true)"
  if [[ -n "$pids" ]]; then
    echo "$pids" | xargs kill
  fi
}

echo "Stopping backend and frontend..."
kill_listener 8000
kill_listener 3001
kill_listener 3000
pkill -f "$PROJECT_ROOT/backend" 2>/dev/null
pkill -f "$PROJECT_ROOT/frontend/node_modules/.bin/next" 2>/dev/null
pkill -f "npm run dev" 2>/dev/null
sleep 1

echo "Starting backend..."
cd "$PROJECT_ROOT/backend"
go run . &
BACKEND_PID=$!
echo "Backend started (PID $BACKEND_PID)"

sleep 2

echo "Starting frontend..."
cd "$PROJECT_ROOT/frontend"
npm run dev -- --port 3001 &
FRONTEND_PID=$!
echo "Frontend started (PID $FRONTEND_PID)"

echo "Both servers running. Backend: $BACKEND_PID, Frontend: $FRONTEND_PID"
