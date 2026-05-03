# Development Workflow

After completing any feature or significant change on the backend or frontend (not iOS app), restart both servers:

1. Kill the backend and frontend processes
2. Start the backend first (`cd backend && go run .`)
3. Then start the frontend (`cd frontend && npm run dev`)
