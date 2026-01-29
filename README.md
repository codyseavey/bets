# LetsBet

Friendly betting pools for groups of friends, family, and coworkers. Create groups, invite people with a code, and run betting pools using a points-based system.

## Features

- **Google OAuth** sign-in
- **Groups** with invite codes, configurable starting points, admin controls
- **Betting pools** with multiple options, one bet per person per pool
- **Proportional payouts** when pools are resolved
- **Points audit trail** tracking every grant, bet, win, and refund
- **Leaderboard** with win/loss records per group
- **Real-time updates** via WebSockets
- **Dark/light/system theme** toggle
- **Mobile-friendly** responsive design

## Quick Start

### Prerequisites
- Go 1.24+
- Node.js 20+
- Google OAuth credentials ([console.cloud.google.com](https://console.cloud.google.com))

### Backend
```bash
cd backend
cp .env.example .env  # edit with your Google OAuth credentials
go mod tidy
go run .
```

### Frontend
```bash
cd frontend
npm install
npm run dev
```

Open `http://localhost:5173`. The Vite dev server proxies API requests to the Go backend on port 8080.

## How It Works

1. Sign in with Google
2. Create a group (you become admin, get starting points)
3. Share the invite code with friends
4. Create a betting pool with 2+ options
5. Members place bets using their points
6. Admin or pool creator resolves the pool by picking the winner
7. Winners split the pot proportionally to their wagers

## Deployment

Hosted at **bets.seavey.dev** via Docker + nginx + Cloudflare.

```bash
# Local Docker build
docker compose -f docker-compose.local.yml up --build
```

See `CLAUDE.md` for full deployment details and required environment variables.

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Frontend | Vue 3, TypeScript, Vite 7, Tailwind CSS 4, Pinia 3 |
| Backend | Go 1.24, Gin, GORM, SQLite (WAL) |
| Auth | Google OAuth 2.0 + JWT |
| Real-time | WebSockets (gorilla/websocket) |
| Deploy | Docker, nginx, GitHub Actions, Cloudflare |
