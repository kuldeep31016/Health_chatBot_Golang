# AI Health Assistant (Go + React + Gemini)

## Project overview

This project is a full-stack AI Health Assistant with:
- **Backend (Go)**: state-machine agent, retry/fault-tolerance, in-memory conversation embeddings
- **Frontend (React + TypeScript)**: chat UI with connection and reconnecting status
- **LLM/Embeddings**: Google Gemini APIs
- **Grounding strategy**: responses are built only from selective user profile and memory context
- **LangGraph workflow**: explicit `process -> tasks -> action -> success/fail` with recursive task execution
- **Execution metadata**: LangGraph returns executed task trace, source links (when web search runs), and latency

## Folder structure

- `backend/` — Go API, state machine agent, tools, memory, jobs
- `frontend/` — React app and chat components
- `langgraph-service/` — optional Python LangGraph runtime (`/run`)
- `.env` — environment variables

## Setup

## End-to-end quick start (your side)

Follow these in order.

### Prerequisites

- Go 1.22+ (for backend): https://go.dev/dl/
- Node.js 18+ (for frontend): https://nodejs.org/en/download
- Python 3.11+ (for optional LangGraph service): https://www.python.org/downloads/
- Gemini API key: https://aistudio.google.com/app/apikey

### 0) Configure environment variables

Edit root `.env` in this project:

- `GEMINI_API_KEY=your_real_key`
- `SERPER_API_KEY=your_serper_key` (required for web search tool)
- `PORT=8080`
- `GEMINI_MODEL=gemini-1.5-flash`
- `GEMINI_EMBEDDING_MODEL=gemini-embedding-001`
- `LANGGRAPH_API_URL=http://localhost:8090` (optional, only if using LangGraph service)

### 1) Configure environment

Set in `.env`:
- `GEMINI_API_KEY=your_key_here`
- `SERPER_API_KEY=your_serper_key_here` (needed for web search)
- `PORT=8080`
- `GEMINI_MODEL=gemini-1.5-flash` (or `gemini-1.5-pro`)
- `GEMINI_EMBEDDING_MODEL=gemini-embedding-001`
- `LANGGRAPH_API_URL=` (optional; when set, backend calls a real LangGraph service at `POST {LANGGRAPH_API_URL}/run`)

### 2) Run backend

From `backend/`:
- Initialize module (if needed): `go mod init health-assistant/backend`
- Start server: `go run main.go`

Backend starts on `http://localhost:8080`.

Health check link:
- http://localhost:8080/health

### 3) Run frontend

From `frontend/`:
- Install dependencies: `npm install`
- Start dev server: `npm run dev`

Frontend starts on `http://localhost:5173`.

Open app link:
- http://localhost:5173

### 4) (Optional) Run LangGraph service

From `langgraph-service/`:
- Install dependencies: `pip install -r requirements.txt`
- Start service: `uvicorn app:app --host 0.0.0.0 --port 8090`

Then set in root `.env`:
- `LANGGRAPH_API_URL=http://localhost:8090`

LangGraph health link:
- http://localhost:8090/health

---

## API endpoints (from your side)

Base backend URL:
- `http://localhost:8080`

1) Submit chat job
- `POST /api/chat`
- body:
```json
{ "message": "I feel dizzy", "session_id": "abc123" }
```
- returns:
```json
{ "job_id": "job-...", "response": "processing", "status": "processing" }
```

2) Poll job result
- `GET /api/chat?job_id=job-...`
- returns one of:
  - `processing`
  - `success` with final `response`
  - `fail` with safe fallback response

3) Backend health
- `GET /health`

---

## How to run everything together (recommended)

Open 3 terminals:

Terminal A (optional LangGraph):
- `cd langgraph-service`
- `pip install -r requirements.txt`
- `uvicorn app:app --host 0.0.0.0 --port 8090`

Terminal B (Go backend):
- `cd backend`
- `go run main.go`

Terminal C (React frontend):
- `cd frontend`
- `npm install`
- `npm run dev`

## API

### POST `/api/chat`
Request:
```json
{ "message": "I feel dizzy", "session_id": "abc123" }
```
Response:
```json
{ "job_id": "job-123", "response": "processing", "status": "processing" }
```

### GET `/api/chat?job_id=...`
Poll job status until completion.

Responses:
```json
{ "job_id": "job-123", "response": "", "status": "processing" }
```

```json
{ "job_id": "job-123", "response": "...assistant reply...", "status": "success" }
```

```json
{ "job_id": "job-123", "response": "I'm having trouble right now. Please try again in a moment.", "status": "fail" }
```

### GET `/health`
Returns `200 OK` when backend is alive.

## Sample test queries

- `I feel dizzy`
- `Suggest exercise for me`
- `What did we discuss earlier?`
- `Do I have any upcoming appointment?`

## Retry and resilience design

### Backend retry
- All external/API operations use retry with up to 3 attempts and 2s delay:
  - Gemini generation
  - Gemini embeddings
  - user profile load and optional web search
- Agent includes explicit state `retry` and transitions to `fail` after max retries.
- Retry path uses goroutines so retry handling is background-friendly and fault-tolerant.
- If failures persist, backend returns a safe fallback:
  - `I'm having trouble right now. Please try again in a moment.`

### Worker pool
- Goroutine-based worker pool executes:
  - tool calls
  - Gemini generation jobs
  - async memory embedding storage
  - async response jobs for `/api/chat` (request returns quickly with `job_id`)

### Frontend retry
- Chat send has retry with exponential backoff: **2s → 4s → 6s**
- Connection status behavior:
  - ✅ Connected
  - 🟡 Reconnecting...
  - 🔴 Disconnected

## Grounding + anti-hallucination rules

- Health JSON is **never sent in full**.
- Only keyword-relevant sections are extracted per query.
- LLM prompt includes only:
  - selected user profile fields
  - relevant health subset
  - top memory matches
  - current user query
- Assistant explicitly states when data is unavailable.

## User profile schema notes

The sample profile includes general attributes (in addition to medical context), including:
- `height_cm`
- `weight_kg`
- `hair_color`
- `eye_color`
- `age`
# Health_chatBot_Golang
# Health_chatBot_Golang
