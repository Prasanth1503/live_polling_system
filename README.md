# Live Polling System

A starter full-stack polling application using React, Go (Gin), MongoDB and Redis.

## Prerequisites
- Node.js and npm
- Go 1.22+
- MongoDB running locally
- Redis running locally

## Run the backend
Open a terminal:

```powershell
cd backend
go mod tidy
$env:MONGO_URI="mongodb://localhost:27017"
$env:MONGO_DATABASE="live_polling"
$env:REDIS_ADDR="localhost:6379"
$env:PORT="8080"
go run .
```

Check `http://localhost:8080/api/health`.

## Run the frontend
Open a second terminal:

```powershell
cd frontend
npm install
npm run dev
```

Open `http://localhost:5173`.

## Current functionality
- Create polls with 2–6 options.
- List polls from MongoDB.
- Record vote counters in Redis.
- Persist a vote-count snapshot in MongoDB.
- Publish vote events to a Redis Pub/Sub channel.
- Basic backend validation.

## Important work still required before internship submission
This is a learning starter, not a finished assignment. It does not yet include signup/login, one-vote-per-user protection, WebSocket delivery to browsers, production-grade vote consistency, automated tests, deployment, or a submission demo video. The frontend currently polls the API every five seconds; this is NOT the required real-time behavior. Implement WebSocket broadcasting backed by Redis Pub/Sub before submitting.

Do not deploy this starter as-is. Add authentication, rate limiting, secure configuration, tests, and proper error handling first.
