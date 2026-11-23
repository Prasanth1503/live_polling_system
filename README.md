# Live Polling System

## Deploying the frontend and backend

The React frontend is a static site; the Go API must run as a separate web
service. The recommended setup is Vercel for the frontend and Render for the
API. `render.yaml` defines the Render service, and `frontend/vercel.json`
supports client-side poll routes.

1. Deploy the repository's `render.yaml` as a Render Blueprint. Set the
   service's `MONGO_URI` to your MongoDB connection string and `JWT_SECRET` to
   a private value of at least 32 characters. Keep both values in Render's
   environment settings; do not commit them.
2. Deploy the repository to Vercel with the project root set to `frontend`
   and the Vite framework preset. Set the Vercel environment variable
   `VITE_API_URL` to the Render service's public URL followed by `/api`, for
   example `https://your-service.onrender.com/api`.
3. Set the Render service's `FRONTEND_URL` to the public Vercel site URL,
   without a trailing slash, then redeploy the Render service. This allows the
   backend's CORS policy to accept requests from the frontend.
4. Redeploy Vercel after setting `VITE_API_URL`. Verify the backend at
   `https://your-service.onrender.com/api/health`; it should return JSON with
   a successful status.

For local development, Vite's `/api` proxy reaches `http://localhost:8080`.
Netlify can also host the static frontend; its build uses `netlify.toml` and
requires the same `VITE_API_URL` environment variable.
