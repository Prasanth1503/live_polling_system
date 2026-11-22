# Live Polling System

A simple full-stack web application that allows users to create polls, share them with others, and collect votes.

I built this project as part of my developer internship preparation to improve my skills in React, Go, MongoDB, and Redis.

The main goal is to make polling easy and allow users to see voting results as they come in.

## Project Overview

The application follows this workflow:

**Create Poll → Share Link → Audience Votes → View Results**

A poll creator can create a question with multiple options. Other users can open the shared poll link and submit their votes.

## Technologies Used 
 React    - Frontend development 
 Go (Gin) - Backend API 
 MongoDB  - Store poll information 
 Redis    - Vote counters and Pub/Sub messaging 

## Features

- Create polls with 2–6 options.
- Display polls stored in MongoDB.
- Share a poll with other users.
- Record vote counts using Redis.
- Save vote-count snapshots in MongoDB.
- Publish voting events through Redis Pub/Sub.
- Validate poll data on the backend.
- Automatically refresh poll results every five seconds.

## Project Structure

```text
Live_Polling_System/
│
├── frontend/
│   ├── src/
│   ├── public/
│   ├── package.json
│   └── ...
│
├── backend/
│   ├── main.go
│   ├── go.mod
│   └── ...
│
├── .gitignore
└── README.md
```

## Requirements

Before running this project, install:

- Node.js and npm
- Go 1.22 or later
- MongoDB
- Redis
- Git

## How to Run the Project

### Step 1: Clone the repository

```bash
git clone YOUR_GITHUB_REPOSITORY_URL
cd Live_Polling_System
```

### Step 2: Start the Backend

Open a terminal:

```powershell
cd backend
go mod tidy
```

Set the environment variables:

```powershell
$env:MONGO_URI="mongodb://localhost:27017"
$env:MONGO_DATABASE="live_polling"
$env:REDIS_ADDR="localhost:6379"
$env:PORT="8080"
```

Start the Go server:

```powershell
go run .
```

Check the backend health endpoint:

http://localhost:8080/api/health

### Step 3: Start the Frontend

Open a second terminal:

```powershell
cd frontend
npm install
npm run dev
```

Open the application:

http://localhost:5173

## Current Development Status

The basic polling functionality is implemented, including poll creation, MongoDB storage, Redis vote counters, and Redis Pub/Sub event publishing.

Some important features are still under development:

- User signup and login.
- One-vote-per-user protection.
- WebSocket integration for live browser updates.
- Improved vote consistency and error handling.
- Automated testing.
- Public deployment.

**Real-time update note:** The current frontend refreshes results every five seconds. WebSocket-based updates are not implemented yet, so the application does not currently meet the internship's no-refresh real-time requirement.

## Future Improvements

I plan to improve the application by adding:

- Secure user authentication.
- Real-time voting results using WebSockets and Redis Pub/Sub.
- Duplicate vote prevention.
- Better UI and mobile responsiveness.
- Automated testing.
- Cloud deployment.

## What I Learned

While working on this project, I gained practical experience with:

- Building frontend applications using React.
- Creating REST APIs using Go and Gin.
- Connecting applications to MongoDB.
- Using Redis for vote counters and messaging.
- Integrating frontend and backend services.
- Debugging and fixing application errors.

## Author

Developed as a developer internship preparation project.
