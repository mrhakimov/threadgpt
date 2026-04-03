# ThreadGPT

ThreadGPT is a chat app for people who want a reusable AI workspace without dragging every past message into every future answer.

Most chat products treat one conversation like one growing memory. That works for some workflows, but it breaks down for repeated micro-tasks like translation, rewriting, grammar checks, snippet explanations, and other "same job, many independent requests" use cases. You do not want to create a brand-new chat for every word you want to translate, but you also do not want every follow-up about one word to affect the next one.

ThreadGPT takes a different approach:

- The first message defines the conversation context.
- Each new top-level message is treated as its own independent turn.
- If you want to go deeper on one answer, you open a thread on that message.
- The follow-up context stays local to that thread instead of leaking into everything that comes after.

In short: one standing tool, many independent requests, optional local deep-dives.

## Why This Exists

This project was built to solve two problems with normal chat interfaces:

1. Context bloat  
   Every extra message increases the amount of conversation history hanging around in the background.

2. Context pollution  
   A follow-up meant for one specific answer can silently affect unrelated future answers.

That is especially awkward in workflows where people naturally want one persistent chat, but do not want one shared evolving memory. A language-learning assistant is a good example: you may want one "translator" workspace for dozens of words or phrases, while still being able to ask deep follow-up questions about a single item without changing how the next item is handled.

## What It Solves

ThreadGPT is useful when:

- you want one reusable assistant for a repeated task
- each new input should mostly be judged on its own
- you still need local follow-ups on individual results
- you want tighter control over what actually gets sent to the model

Examples:

- translation and language learning
- rewriting messages and emails
- grammar explanation
- explaining short code snippets
- recurring structured prompts where follow-ups should stay local

## Current Product Behavior

- Your first message becomes the conversation-level instruction set.
- Main-chat messages are top-level turns.
- Assistant answers can be opened in a side thread for local follow-up.
- Thread replies use the parent answer plus that thread's own history.
- The system prompt can be edited later for the whole conversation.
- Conversations can be renamed, revisited, and deleted.

## Stack

- Frontend: Next.js 15, React 19, TypeScript, Tailwind CSS
- Backend: Go
- Storage: Supabase
- Model access: OpenAI API via a user-provided API key

## Running It Yourself

### Requirements

- Go `1.25.0`
- Node.js `18+` should be fine, `20+` recommended
- npm
- A Supabase project
- An OpenAI API key for signing into the app

### 1. Create the database schema

Create a Supabase project, then run [`20260403_1410_initial_schema.sql`](/Users/john/Documents/projects/threadgpt/migrations/20260403_1410_initial_schema.sql) in the Supabase SQL editor.

### 2. Configure the backend

Create [`backend/.env`](/Users/john/Documents/projects/threadgpt/backend/.env):

```env
SUPABASE_URL=https://YOUR_PROJECT.supabase.co
SUPABASE_SERVICE_KEY=YOUR_SUPABASE_SERVICE_ROLE_KEY
ALLOWED_ORIGIN=http://localhost:3000
TOKEN_ENCRYPTION_KEY=YOUR_64_CHAR_HEX_KEY
PORT=8000
```

Notes:

- `SUPABASE_SECRET_KEY` also works instead of `SUPABASE_SERVICE_KEY`.
- `TOKEN_ENCRYPTION_KEY` must be exactly 64 hex characters.
- If `TOKEN_ENCRYPTION_KEY` is missing, the backend will generate an ephemeral key and login sessions will not survive restart.
- For local development, `ALLOWED_ORIGIN=http://localhost:3000` is the simplest setup.

Generate a random encryption key with:

```bash
openssl rand -hex 32
```

### 3. Configure the frontend

Create [`frontend/.env.local`](/Users/john/Documents/projects/threadgpt/frontend/.env.local):

```env
NEXT_PUBLIC_API_URL=http://localhost:8000
```

### 4. Install frontend dependencies

```bash
cd frontend
npm install
```

### 5. Start the backend

```bash
cd backend
go run .
```

The backend listens on `http://localhost:8000` by default.

### 6. Start the frontend

In a second terminal:

```bash
cd frontend
npm run dev
```

Then open `http://localhost:3000`.

## Local Development Notes

- The app asks the user for their OpenAI API key and stores encrypted session data server-side for authentication.
- When `TOKEN_ENCRYPTION_KEY` is set, the token store is also persisted to [`backend/.token_store.json`](/Users/john/Documents/projects/threadgpt/backend/.token_store.json).
- The raw API key is never stored in the database.
- The frontend defaults to `localhost:3000`.
- There is also a helper script at [`scripts/restart.sh`](/Users/john/Documents/projects/threadgpt/scripts/restart.sh) that restarts both servers and starts the frontend on port `3000`.

## Development Workflow

Per the repo instructions, after any significant change:

1. Kill the backend and frontend processes.
2. Start the backend first with `cd backend && go run .`.
3. Start the frontend second with `cd frontend && npm run dev`.

## Project Structure

- [`backend`](/Users/john/Documents/projects/threadgpt/backend): Go API, auth, chat/thread handling, storage integration
- [`frontend`](/Users/john/Documents/projects/threadgpt/frontend): Next.js app and UI
- [`migrations/20260403_1410_initial_schema.sql`](/Users/john/Documents/projects/threadgpt/migrations/20260403_1410_initial_schema.sql): Supabase schema
- [`scripts/restart.sh`](/Users/john/Documents/projects/threadgpt/scripts/restart.sh): local restart helper

## Roadmap Direction

The most interesting direction for this project is not "more memory." It is better scope control:

- clearer per-turn context controls
- better visibility into what was sent to the model
- temporary one-off instructions
- stronger thread summaries and navigation
- support for multiple model providers

## Contact

Built by [@omtiness](https://x.com/omtiness).
