#!/bin/bash

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

echo "Stopping backend and frontend..."
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
