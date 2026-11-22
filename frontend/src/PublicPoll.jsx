
import React, { useEffect, useState } from "react";

const API = import.meta.env.VITE_API_URL || "http://localhost:8080/api";

export default function PublicPoll() {
    const pollId = window.location.pathname.split("/").pop();

    const [poll, setPoll] = useState(null);
    const [selected, setSelected] = useState(null);
    const [message, setMessage] = useState("");
    const [loading, setLoading] = useState(true);
    const [voted, setVoted] = useState(false);

    async function loadPoll() {
        try {
            const response = await fetch(`${API}/polls/${pollId}`);
            const data = await response.json();

            if (!response.ok) {
                throw new Error(data.error || "Poll not found");
            }

            setPoll(data.poll || data);
        } catch (err) {
            setMessage(err.message);
        } finally {
            setLoading(false);
        }
    }

    useEffect(() => {
        loadPoll();

        const timer = setInterval(loadPoll, 2000);

        return () => clearInterval(timer);
    }, [pollId]);

    async function submitVote() {
        if (selected === null || voted) return;

        try {
            const response = await fetch(
                `${API}/polls/${pollId}/vote`,
                {
                    method: "POST",
                    headers: {
                        "Content-Type": "application/json"
                    },
                    body: JSON.stringify({
                        optionIndex: selected
                    })
                }
            );

            const data = await response.json();

            if (!response.ok) {
                throw new Error(data.error || "Voting failed");
            }

            setVoted(true);
            setMessage("Your vote was submitted successfully!");

            await loadPoll();

        } catch (err) {
            setMessage(err.message);
        }
    }

    if (loading) {
        return <div className="public-poll">Loading poll...</div>;
    }

    if (!poll) {
        return (
            <div className="public-poll">
                <h2>Poll Not Found</h2>
                <p>{message}</p>
            </div>
        );
    }

    return (
        <div className="public-poll">
            <p className="eyebrow">PUBLIC POLL</p>

            <h1>Vote Now</h1>

            <h2>{poll.question}</h2>

            <p className="muted">
                Total Votes: {poll.totalVotes || 0}
            </p>

            {!voted && (
                <>
                    {poll.options?.map((option, index) => (
                        <label className="poll-option" key={index}>
                            <input
                                type="radio"
                                name="poll"
                                checked={selected === index}
                                onChange={() => setSelected(index)}
                            />
                            {option}
                        </label>
                    ))}

                    <button
                        onClick={submitVote}
                        disabled={selected === null}
                    >
                        Submit Vote
                    </button>
                </>
            )}

            {message && (
                <div className="success">{message}</div>
            )}

            <h2>Live Results</h2>

            {poll.options?.map((option, index) => {
                const count = poll.counts?.[index] || 0;

                const percentage = poll.totalVotes
                    ? (count / poll.totalVotes) * 100
                    : 0;

                return (
                    <div className="result" key={index}>
                        <p>
                            {option} — {count} votes
                            ({percentage.toFixed(1)}%)
                        </p>

                        <div className="progress">
                            <div
                                className="progress-fill"
                                style={{ width: `${percentage}%` }}
                            />
                        </div>
                    </div>
                );
            })}
        </div>
    );
}
