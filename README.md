# Live Polling System

## About the Project

The Live Polling System is a web-based application that allows users to create polls, share them with others, and collect votes in real time.

I developed this project to make online voting simple and interactive. Users can create a question with multiple options, share the poll link with others, and view the voting results as people participate.

## Features

- User Registration and Login
- Create polls with multiple options
- Share polls using a public link
- Allow users to vote
- Display live voting results
- View poll details
- Simple and user-friendly interface

## Technologies Used

**Frontend**
- React.js
- JavaScript
- HTML
- CSS

**Backend**
- Go (Golang)
- Gin Framework
- MongoDB
- Redis

## Project Structure

```text
Live_Polling_System/
├── frontend/
│   ├── src/
│   └── package.json
├── backend/
│   ├── main.go
│   └── ...
└── README.md
```

## How to Run the Project

### 1. Clone the Repository

```bash
git clone YOUR_GITHUB_REPOSITORY_URL
```

### 2. Start the Backend

```bash
cd backend
go mod tidy
go run .
```

### 3. Start the Frontend

Open another terminal:

```bash
cd frontend
npm install
npm run dev
```

### 4. Open the Application

Open the local URL displayed in your terminal, usually:

```text
http://localhost:5173
```

## Project Goal

The main goal of this project is to provide an easy way to conduct online polls, collect responses, and display voting results instantly.

## Future Improvements

- Poll expiration
- Admin dashboard
- Improved security
- Voting analytics
- Mobile-friendly design

## Author

Developed as a learning project to improve my skills in full-stack web development.
