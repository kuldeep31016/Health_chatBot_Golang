# LangGraph Service (Optional)

This optional service uses **LangGraph itself** and exposes:
- `POST /run` → returns `{ "response": "..." }`
- `GET /health`

## Setup

1. Create a Python environment.
2. Install requirements:
   - `pip install -r requirements.txt`
3. Set env vars:
   - `GEMINI_API_KEY=...`
   - `GEMINI_MODEL=gemini-1.5-flash` (optional)
4. Run:
   - `uvicorn app:app --host 0.0.0.0 --port 8090`

## Connect from Go backend

In root `.env` set:
- `LANGGRAPH_API_URL=http://localhost:8090`

When this is set, the Go backend calls the LangGraph service first and falls back to local Go state machine if unavailable.
