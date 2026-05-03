# ThreadGPT iOS

Native SwiftUI client for the ThreadGPT backend.

## Open

Open `ThreadGPT.xcodeproj` in Xcode, select the `ThreadGPT` scheme, and run on an iOS simulator.

The app defaults to `http://localhost:8000`, which works from the iOS simulator when the Go backend is running on the same Mac. For a physical device, enter your Mac's LAN URL in the sign-in screen or Settings, for example `http://192.168.1.25:8000`.

## Architecture

- `API/`: URLSession client, JSON requests, SSE streaming, backend error parsing
- `Auth/`: Keychain-backed bearer token storage
- `ViewModels/`: MVVM state and user intents
- `Views/`: SwiftUI rendering only

## Current Scope

- OpenAI API-key sign in against the existing backend
- Bearer-token auth with Keychain persistence
- Conversation list with refresh, rename, and delete
- New conversation flow where the first message sets instructions
- Main chat streaming via the backend SSE API
- Per-answer thread sheet with streaming follow-ups
- Settings for backend URL, system prompt editing, and sign out
