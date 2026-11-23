
import React, { useEffect, useState } from "react";
import PublicPoll from "./PublicPoll";
import { API, getApiResponse } from "./api";

function Dashboard() {
    // Authentication states
    const [user, setUser] = useState(
        localStorage.getItem("email") || ""
    );

    const [token, setToken] = useState(
        localStorage.getItem("token") || ""
    );

    const [isSignup, setIsSignup] = useState(false);
    const [email, setEmail] = useState("");
    const [password, setPassword] = useState("");

    // Poll states
    const [question, setQuestion] = useState("");
    const [options, setOptions] = useState(["", ""]);
    const [polls, setPolls] = useState([]);

    // Share and message states
    const [shareLink, setShareLink] = useState("");
    const [error, setError] = useState("");
    const [success, setSuccess] = useState("");
    const [loading, setLoading] = useState(false);

// LOGIN / SIGNUP
async function auth() {
    setError("");
    setSuccess("");

    const cleanEmail = email.trim().toLowerCase();

    if (!cleanEmail || !password) {
        setError("Please enter email and password.");
        return;
    }

    if (isSignup && password.length < 8) {
        setError("Password must contain at least 8 characters.");
        return;
    }

    const endpoint = isSignup ? "signup" : "login";
    const url = `${API}/auth/${endpoint}`;

    try {
        setLoading(true);

        console.log("Authentication URL:", url);

        const response = await fetch(url, {
            method: "POST",
            headers: {
                "Content-Type": "application/json",
                "Accept": "application/json"
            },
            body: JSON.stringify({
                email: cleanEmail,
                password
            })
        });

        const data = await getApiResponse(response);

        console.log("Authentication response:", response.status, data);

        if (!response.ok) {
            throw new Error(
                data.error || data.message || `Request failed: ${response.status}`
            );
        }

        if (!data.token) {
            throw new Error("Backend did not return a JWT token.");
        }

        localStorage.setItem("token", data.token);
        localStorage.setItem("email", data.user?.email || cleanEmail);

        setToken(data.token);
        setUser(data.user?.email || cleanEmail);
        setPassword("");

        setSuccess(
            isSignup
                ? "Account created successfully!"
                : "Login successful!"
        );

    } catch (err) {
        console.error("Authentication error:", err);

        if (err instanceof TypeError) {
            setError(
                "Cannot connect to backend. Check the API URL, backend server and CORS settings."
            );
        } else {
            setError(err.message || "Authentication failed.");
        }
    } finally {
        setLoading(false);
    }
}


    // LOAD POLLS
    async function loadPolls() {
        if (!token) return;

        try {
            const response = await fetch(`${API}/polls`, {
                method: "GET",
                headers: {
                    Authorization: `Bearer ${token}`
                }
            });

            const data = await getApiResponse(response);

            if (!response.ok) {
                throw new Error(
                    data.error || "Unable to load polls."
                );
            }

            setPolls(
                Array.isArray(data.polls)
                    ? data.polls
                    : []
            );

        } catch (err) {
            setError(err.message);
        }
    }

    useEffect(() => {
        loadPolls();
    }, [token]);

    // UPDATE OPTION
    function updateOption(index, value) {
        setOptions(prev =>
            prev.map((item, i) =>
                i === index ? value : item
            )
        );
    }

    // ADD OPTION
    function addOption() {
        setOptions(prev => [...prev, ""]);
    }

    // REMOVE OPTION
    function removeOption(index) {
        setOptions(prev =>
            prev.filter((_, i) => i !== index)
        );
    }

    // CREATE POLL
    async function createPoll() {
        setError("");
        setSuccess("");
        setShareLink("");

        const cleanQuestion = question.trim();

        const cleanOptions = options
            .map(option => option.trim())
            .filter(Boolean);

        if (!cleanQuestion) {
            setError("Please enter a poll question.");
            return;
        }

        if (cleanOptions.length < 2) {
            setError("Enter at least two poll options.");
            return;
        }

        try {
            setLoading(true);

            const response = await fetch(`${API}/polls`, {
                method: "POST",
                headers: {
                    "Content-Type": "application/json",
                    Authorization: `Bearer ${token}`
                },
                body: JSON.stringify({
                    question: cleanQuestion,
                    options: cleanOptions
                })
            });

            const data = await getApiResponse(response);

            if (!response.ok) {
                throw new Error(
                    data.error || "Poll creation failed."
                );
            }

            const poll = data.poll || data;
            const id = poll.id || poll._id;

            if (!id) {
                throw new Error(
                    "Poll created, but backend did not return poll ID."
                );
            }

            const link =
                `${window.location.origin}/poll/${encodeURIComponent(id)}`;

            setShareLink(link);

            setQuestion("");
            setOptions(["", ""]);

            setSuccess("Poll created successfully!");

            await loadPolls();

        } catch (err) {
            setError(err.message);
        } finally {
            setLoading(false);
        }
    }

    // SHARE POLL
    async function sharePoll(link) {
        if (!link) {
            setError("Share link is not available.");
            return;
        }

        setError("");
        setSuccess("");

        try {
            if (navigator.share) {
                await navigator.share({
                    title: "Live Poll",
                    text: "Participate in this poll!",
                    url: link
                });

                setSuccess("Poll shared successfully!");

            } else {
                const whatsappUrl =
                    `https://wa.me/?text=${encodeURIComponent(
                        "Vote in this poll: " + link
                    )}`;

                window.open(
                    whatsappUrl,
                    "_blank",
                    "noopener,noreferrer"
                );

                setSuccess("WhatsApp sharing opened!");
            }

        } catch (err) {
            if (err.name !== "AbortError") {
                setError("Unable to share poll. Please try again.");
            }
        }
    }

    // LOGOUT
    function logout() {
        localStorage.removeItem("token");
        localStorage.removeItem("email");

        setToken("");
        setUser("");
        setEmail("");
        setPassword("");
        setPolls([]);
        setShareLink("");
        setError("");
        setSuccess("");
    }

    // LOGIN / SIGNUP SCREEN
    if (!token) {
        return (
            <div className="container">
                <div className="panel auth-panel">

                    <p className="eyebrow">
                        LIVE POLLING SYSTEM
                    </p>

                    <h1>
                        {isSignup
                            ? "Create Account"
                            : "Welcome Back"}
                    </h1>

                    <p className="sub">
                        {isSignup
                            ? "Register to create and manage polls."
                            : "Login to manage your polls."}
                    </p>

                    {error && (
                        <div className="error">{error}</div>
                    )}

                    {success && (
                        <div className="success">{success}</div>
                    )}

                    <label>Email</label>

                    <input
                        type="email"
                        placeholder="Enter your email"
                        value={email}
                        onChange={e => setEmail(e.target.value)}
                    />

                    <label>Password</label>

                    <input
                        type="password"
                        placeholder="Enter password"
                        value={password}
                        onChange={e => setPassword(e.target.value)}
                    />

                    <button
                        type="button"
                        onClick={auth}
                        disabled={loading}
                    >
                        {loading
                            ? "Please wait..."
                            : isSignup
                                ? "Sign Up"
                                : "Login"}
                    </button>

                    <p className="sub">
                        {isSignup
                            ? "Already have an account?"
                            : "Don't have an account?"}
                    </p>

                    <button
                        type="button"
                        className="secondary"
                        onClick={() => {
                            setIsSignup(!isSignup);
                            setError("");
                            setSuccess("");
                        }}
                    >
                        {isSignup ? "Login" : "Create Account"}
                    </button>

                </div>
            </div>
        );
    }

    // DASHBOARD
    return (
        <div className="container">

            <header>
                <p className="eyebrow">
                    LIVE POLLING SYSTEM
                </p>

                <h1>Poll Dashboard</h1>

                <p className="sub">
                    Welcome, {user}
                </p>

                <button
                    type="button"
                    className="logout"
                    onClick={logout}
                >
                    Logout
                </button>
            </header>

            {error && (
                <div className="error">{error}</div>
            )}

            {success && (
                <div className="success">{success}</div>
            )}

            {/* CREATE NEW POLL */}

            <section className="panel">

                <h2>Create New Poll</h2>

                <label>Poll Question</label>

                <input
                    type="text"
                    placeholder="Enter your poll question"
                    value={question}
                    onChange={e => setQuestion(e.target.value)}
                />

                <label>Poll Options</label>

                {options.map((option, index) => (
                    <div
                        className="option-input"
                        key={index}
                    >
                        <input
                            type="text"
                            placeholder={`Option ${index + 1}`}
                            value={option}
                            onChange={e =>
                                updateOption(index, e.target.value)
                            }
                        />

                        {options.length > 2 && (
                            <button
                                type="button"
                                className="secondary"
                                onClick={() => removeOption(index)}
                            >
                                Remove
                            </button>
                        )}
                    </div>
                ))}

                <button
                    type="button"
                    className="secondary"
                    onClick={addOption}
                >
                    + Add Option
                </button>

                <button
                    type="button"
                    onClick={createPoll}
                    disabled={loading}
                >
                    {loading ? "Creating..." : "Create Poll"}
                </button>

                {/* NEW POLL SHARE SECTION */}

                {shareLink && (
                    <div className="share-box">

                        <h3>Poll Created Successfully!</h3>

                        <label>Public Poll Link</label>

                        <div className="copy-container">

                            <input
                                type="text"
                                value={shareLink}
                                readOnly
                                onClick={e => e.target.select()}
                            />

                            <button
                                type="button"
                                className="share-btn"
                                onClick={() => sharePoll(shareLink)}
                            >
                                ↗ Share Poll
                            </button>

                        </div>

                        <a
                            href={shareLink}
                            target="_blank"
                            rel="noreferrer"
                            className="view-link"
                        >
                            View Public Poll
                        </a>

                    </div>
                )}

            </section>

            {/* MY POLLS */}

            <section className="poll-section">

                <h2>My Polls</h2>

                {polls.length === 0 ? (
                    <p className="sub">
                        No polls created yet.
                    </p>
                ) : (

                    <div className="poll-grid">

                        {polls.map(poll => {

                            const id = poll.id || poll._id;

                            if (!id) return null;

                            const link =
                                `${window.location.origin}/poll/${encodeURIComponent(id)}`;

                            const totalVotes =
                                poll.totalVotes || 0;

                            return (
                                <div
                                    className="poll-card"
                                    key={id}
                                >

                                    <h3>{poll.question}</h3>

                                    <p className="muted">
                                        Total Votes: {totalVotes}
                                    </p>

                                    {poll.options?.map(
                                        (option, index) => {

                                            const count =
                                                poll.counts?.[index] || 0;

                                            const percentage =
                                                totalVotes > 0
                                                    ? (count / totalVotes) * 100
                                                    : 0;

                                            return (
                                                <div
                                                    className="option"
                                                    key={index}
                                                >

                                                    <div className="option-line">

                                                        <span>{option}</span>

                                                        <span>
                                                            {count} votes
                                                            ({percentage.toFixed(1)}%)
                                                        </span>

                                                    </div>

                                                    <div className="bar">
                                                        <span
                                                            style={{
                                                                width: `${percentage}%`
                                                            }}
                                                        />
                                                    </div>

                                                </div>
                                            );
                                        }
                                    )}

                                    {/* POLL ACTIONS */}

                                    <div className="poll-actions">

                                        <button
                                            type="button"
                                            className="share-btn"
                                            onClick={() => sharePoll(link)}
                                        >
                                            ↗ Share Poll
                                        </button>

                                        <a
                                            href={`/poll/${encodeURIComponent(id)}`}
                                            target="_blank"
                                            rel="noreferrer"
                                            className="view-link"
                                        >
                                            View Public Poll
                                        </a>

                                    </div>

                                </div>
                            );
                        })}

                    </div>
                )}

            </section>

            <footer>
                Live Polling System © 2026
            </footer>

        </div>
    );
}

// APP ROUTING
export default function App() {
    if (window.location.pathname.startsWith("/poll/")) {
        return <PublicPoll />;
    }

    return <Dashboard />;
}