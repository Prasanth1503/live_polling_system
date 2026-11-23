
import React, { useEffect, useState, useCallback } from "react";
import { API, getApiResponse } from "./api";

export default function PublicPoll() {
    const pollId = window.location.pathname.split("/").pop();

    const [poll, setPoll] = useState(null);
    const [selected, setSelected] = useState(null);
    const [message, setMessage] = useState("");
    const [loading, setLoading] = useState(true);
    const [voted, setVoted] = useState(false);
    const [submitting, setSubmitting] = useState(false);

    const loadPoll = useCallback(async () => {
        try {
            const response = await fetch(`${API}/polls/${pollId}`);

            const data = await getApiResponse(response);

            if (!response.ok) {
                throw new Error(data.error || "Unable to load poll");
            }

            const pollData = data.poll || data;

            setPoll(pollData);
            setMessage("");
        } catch (error) {
            setMessage(error.message || "Failed to fetch poll");
        } finally {
            setLoading(false);
        }
    }, [pollId]);

    useEffect(() => {
        loadPoll();

        const timer = setInterval(loadPoll, 2000);

        return () => clearInterval(timer);
    }, [loadPoll]);

    async function submitVote() {
        if (selected === null || voted || submitting) {
            return;
        }

        setSubmitting(true);
        setMessage("");

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

            const data = await getApiResponse(response);

            if (!response.ok) {
                throw new Error(data.error || "Voting failed");
            }

            setVoted(true);
            setMessage("Your vote was submitted successfully!");

            await loadPoll();
        } catch (error) {
            setMessage(error.message || "Unable to submit vote");
        } finally {
            setSubmitting(false);
        }
    }

    if (loading) {
        return (
            <div className="public-poll">
                <h2>Loading poll...</h2>
            </div>
        );
    }

    if (!poll) {
        return (
            <div className="public-poll">
                <h2>Poll Not Found</h2>
                <p>{message}</p>
                <button onClick={loadPoll}>Try Again</button>
            </div>
        );
    }

    const options = Array.isArray(poll.options)
        ? poll.options
        : [];

    const totalVotes = Number(poll.totalVotes || 0);

    const counts = Array.isArray(poll.counts)
        ? poll.counts
        : [];

    return (
        <div className="public-poll">
            <p className="eyebrow">PUBLIC POLL</p>

            <h1>Vote Now</h1>

            <h2>{poll.question}</h2>

            <p className="muted">
                Total Votes: {totalVotes}
            </p>

            {!voted && (
                <>
                    {options.map((option, index) => (
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
                        disabled={selected === null || submitting}
                    >
                        {submitting ? "Submitting..." : "Submit Vote"}
                    </button>
                </>
            )}

            {message && (
                <div className="success">
                    {message}
                </div>
            )}

            <h2>Live Results</h2>

            {options.map((option, index) => {
                const count = Number(counts[index] || 0);

                const percentage = totalVotes > 0
                    ? (count / totalVotes) * 100
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
                                style={{
                                    width: `${percentage}%`
                                }}
                            />
                        </div>
                    </div>
                );
            })}
        </div>
    );
}
