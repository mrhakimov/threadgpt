# ThreadGPT iOS V2

A fresh SwiftUI implementation of ThreadGPT that mirrors the current web client while staying independent from the existing `ios/` folder.

## Structure

- `Domain`: entities, repository protocols, app errors, and pure rendering/domain helpers.
- `Data`: URLSession API client, bearer-token auth, SSE streaming, Keychain token storage, and remote repositories.
- `Application`: use-case services that mirror the web client behavior.
- `Presentation`: MVVM view models, SwiftUI screens, reusable components, and the ThreadGPT palette.

## Features

- API-key login, auth checking, auth expiry display, and logout.
- Conversation list with pagination, rename, delete, and new conversation flow.
- Main chat view with first-message-as-context behavior, streaming assistant responses, older-message loading, copy actions, and editable system prompt.
- Subthreads for assistant messages with follow-up history, streaming replies, and local reply-count updates.
- Settings for theme, session status, server URL, and logout.

The default backend URL is `http://localhost:8000`, matching local development. Change it on the login screen or in Settings when testing on another device. The app allows cleartext HTTP so local ThreadGPT servers work during development; use HTTPS for any non-local deployment.
