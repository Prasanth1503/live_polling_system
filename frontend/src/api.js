export const API = (
    import.meta.env.VITE_API_URL || "/api"
).replace(/\/$/, "");

export async function getApiResponse(response) {
    const text = await response.text();

    try {
        return text ? JSON.parse(text) : {};
    } catch {
        if (text.trimStart().startsWith("<")) {
            throw new Error(
                "The API returned a web page instead of JSON. Deploy the backend and set VITE_API_URL to its API URL."
            );
        }

        throw new Error("The backend returned an invalid response.");
    }
}
