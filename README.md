# Project Insights Platform

Backend and client applications for consuming time data exported by the Project Insights ecosystem.

Current components:
- `services/dashboard-backend/` — Go API service for loading and aggregating time entries.
- `apps/dashboard-frontend/` — React dashboard that visualizes API data.
- `docs/` — Data contracts, aggregation rules, and platform architecture notes.

## Direction

This repository is intentionally API-first. The web dashboard is one consumer, but the backend API should remain stable and reusable for future clients, including:
- Android application
- iOS application
- macOS widgets
- iPhone widgets

The backend should avoid dashboard-specific assumptions in route design and response models.

## Repository principles

- Treat `docs/backend-contract.md` as the source of truth for entry schema and API boundaries.
- Keep aggregation rules deterministic and documented (`docs/aggregation-spec.md`).
- Keep client-specific presentation logic in each client app, not in shared API contracts.
- Prefer additive API evolution to avoid breaking future mobile/widget clients.

## Quick start

Backend:

```bash
cd services/dashboard-backend
go test ./...
go run .
```

Frontend:

```bash
cd apps/dashboard-frontend
npm install
npm test
npm run build
npm run dev
```
