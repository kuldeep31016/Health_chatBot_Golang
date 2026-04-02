# AI Health Assistant
## Personalized Health Guidance Powered by Agentic AI

![Status](https://img.shields.io/badge/status-production_ready-brightgreen)
![Go](https://img.shields.io/badge/backend-Go_1.22+-00ADD8)
![React](https://img.shields.io/badge/frontend-React_18+-61DAFB)
![Python](https://img.shields.io/badge/orchestration-Python_3.11+-3776AB)

---

## 📽️ Demo Video

**[Watch Full Project Demo on YouTube](https://www.youtube.com/watch?v=sSzMFRHcmtU)**
---

## 🎯 Project Overview

**AI Health Assistant** is a full-stack, production-grade chatbot that provides **personalized health guidance** by combining:

- ✅ **User health profile grounding** (allergies, goals, medical history)
- ✅ **Tool-based agentic reasoning** (profile lookup, health data analysis, web search)
- ✅ **Conversational memory** (embedding-based context continuity)
- ✅ **Fault-tolerant execution** (retry logic, background workers)
- ✅ **Medical-safe responses** (avoids hallucination, recommends professional care when needed)
- ✅ **Multi-service architecture** (Go backend, React frontend, optional Python orchestration)

### Key Features

| Feature | Implementation |
|---------|-----------------|
| **Personalization** | Structured user profile (height, weight, allergies, goals) integrated into tool calls |
| **Agent System** | State-machine based conversation flow with tool invocation decisions |
| **Memory System** | Embedding-based retrieval for contextual conversation continuity |
| **Safety** | Medical disclaimer messages; avoids diagnosis without professional care |
| **Scalability** | Concurrent Go handlers, background job workers, optional external workflow service |
| **LLM Integration** | Google Gemini APIs (text generation + embeddings) |

---

## 🏗️ Architecture

### System Design

```
┌─────────────────────────────────────────────────────────────┐
│                        Frontend (React)                      │
│          Chat UI • Message History • Connection Status       │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       │ POST /api/chat
                       ▼
┌─────────────────────────────────────────────────────────────┐
│                  Backend (Go) API Server                     │
│  ┌────────────────────────────────────────────────────────┐ │
│  │  Chat Handler (handlers/chat.go)                       │ │
│  │  • Parse request • Route to agent • Return response    │ │
│  └────────────────────────────────────────────────────────┘ │
│  ┌────────────────────────────────────────────────────────┐ │
│  │  Agent Engine (agent/)                                 │ │
│  │  • State machine logic • Tool selection • Transitions  │ │
│  └────────────────────────────────────────────────────────┘ │
│  ┌────────────────────────────────────────────────────────┐ │
│  │  Tool Suite (tools/)                                   │ │
│  │  • profile_qa.go • health_data.go • memory.go         │ │
│  │  • web_search.go • user_data.go                       │ │
│  └────────────────────────────────────────────────────────┘ │
│  ┌────────────────────────────────────────────────────────┐ │
│  │  Memory & Context (memory/)                            │ │
│  │  • Embedding generation (Gemini)                       │ │
│  │  • Retrieval-augmented response                        │ │
│  └────────────────────────────────────────────────────────┘ │
│  ┌────────────────────────────────────────────────────────┐ │
│  │  LLM Integration (llm/gemini.go)                       │ │
│  │  • Text generation • Embedding API calls               │ │
│  └────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
                       │
                       │ (Optional)
                       ▼
┌─────────────────────────────────────────────────────────────┐
│            LangGraph Service (Python Optional)              │
│  Graph-based workflow for multi-step process automation     │
└─────────────────────────────────────────────────────────────┘
```

### Folder Structure

```
health-assistant/
├── README.md                    # Project documentation
├── .env                         # Environment configuration
├── backend/                     # Go backend service
│   ├── main.go                  # Entry point
│   ├── go.mod
│   ├── agent/                   # State machine & logic
│   │   ├── agent.go
│   │   ├── states.go
│   │   └── transitions.go
│   ├── handlers/                # HTTP request handlers
│   │   └── chat.go
│   ├── tools/                   # Tool implementations
│   │   ├── profile_qa.go
│   │   ├── health_data.go
│   │   ├── memory.go
│   │   ├── web_search.go
│   │   └── user_data.go
│   ├── memory/                  # Embedding & retrieval
│   │   ├── embedding.go
│   │   └── store.go
│   ├── llm/                     # LLM integration
│   │   └── gemini.go
│   ├── jobs/                    # Background processing
│   │   ├── worker.go
│   │   └── retry.go
│   └── data/
│       └── user_profile.json    # Sample user data
├── frontend/                    # React + TypeScript frontend
│   ├── index.html
│   ├── package.json
│   ├── tsconfig.json
│   ├── vite.config.ts
│   └── src/
│       ├── main.tsx
│       ├── App.tsx
│       ├── vite-env.d.ts
│       ├── api/
│       │   └── chat.ts          # API client
│       └── components/
│           ├── ChatWindow.tsx
│           ├── ConnectionStatus.tsx
│           ├── InputBox.tsx
│           └── MessageBubble.tsx
└── langgraph-service/           # Python orchestration (optional)
    ├── app.py
    ├── README.md
    └── requirements.txt
```

---

## 🚀 Quick Start

### Prerequisites

- **Go 1.22+** → [Download](https://go.dev/dl/)
- **Node.js 18+** → [Download](https://nodejs.org/en/download)
- **Python 3.11+** (optional) → [Download](https://www.python.org/downloads/)
- **API Keys:**
  - Google Gemini API → [Get Key](https://aistudio.google.com/app/apikey)
  - Serper (web search) → [Get Key](https://serper.dev/)

### 1️⃣ Environment Configuration

Create `.env` in project root:

```bash
# LLM Configuration
GEMINI_API_KEY=your_gemini_api_key_here
GEMINI_MODEL=gemini-1.5-flash
GEMINI_EMBEDDING_MODEL=gemini-embedding-001

# External Tools
SERPER_API_KEY=your_serper_api_key_here

# Server
PORT=8080

# LangGraph Service (optional)
LANGGRAPH_API_URL=http://localhost:8090
```

### 2️⃣ Backend Setup

```bash
cd backend
go run main.go
```

Backend runs on `http://localhost:8080`

Health check: `http://localhost:8080/health`

### 3️⃣ Frontend Setup

```bash
cd frontend
npm install
npm run dev
```

Frontend runs on `http://localhost:5173`

Open browser: `http://localhost:5173`

### 4️⃣ LangGraph Service (Optional)

```bash
cd langgraph-service
pip install -r requirements.txt
uvicorn app:app --host 0.0.0.0 --port 8090
```

Then update `.env`:
```
LANGGRAPH_API_URL=http://localhost:8090
```

---

## 🔌 API Reference

### Base URL
```
http://localhost:8080
```

### Endpoints

#### 1. Submit Chat Message
```
POST /api/chat
```

**Request:**
```json
{
  "message": "What should I avoid eating?",
  "session_id": "user-abc123"
}
```

**Response:**
```json
{
  "job_id": "job-f7da2c1e",
  "status": "processing",
  "response": "Processing your request..."
}
```

#### 2. Poll Job Result
```
GET /api/chat?job_id=job-f7da2c1e
```

**Response (Processing):**
```json
{
  "status": "processing",
  "response": "Finding relevant health information..."
}
```

**Response (Success):**
```json
{
  "status": "success",
  "response": "Based on your profile, you're allergic to peanuts and shellfish. Your diet preference is Mediterranean...",
  "metadata": {
    "tools_used": ["profile_qa", "memory"],
    "latency_ms": 1247
  }
}
```

#### 3. Health Check
```
GET /health
```

**Response:**
```json
{
  "status": "healthy",
  "timestamp": "2026-04-02T15:30:45Z"
}
```

---

## 🛠️ Core Components

### Agent System (State Machine)

The agent uses a **state-driven approach** for conversation flow:

- **States** (`agent/states.go`): Conversation states (idle, processing, responding)
- **Transitions** (`agent/transitions.go`): Logic for state transitions based on user intent
- **Engine** (`agent/agent.go`): Main orchestration loop

**Flow:**
```
User Input → Parse Intent → Evaluate State → Select Tool(s) → Execute → Generate Response → Return
```

### Tool Suite

| Tool | Purpose | Input | Output |
|------|---------|-------|--------|
| `profile_qa` | Answer questions using user profile data | Question | Profile-based answer |
| `health_data` | Fetch medical metrics (BMI, bloodwork, etc.) | Query | Calculated health metrics |
| `memory` | Retrieve conversation context via embeddings | Query | Relevant past context |
| `web_search` | Search external medical sources | Query | Web results + attribution |
| `user_data` | Update/fetch user profile information | Data key | Current user data |

### Memory & Embeddings

- Uses Google Gemini embedding model to vectorize messages
- Stores embeddings in local in-memory store
- Retrieves top-k similar past exchanges for context grounding
- Reduces hallucinations by anchoring responses to known context

### LLM Integration (Gemini)

- **Text Generation**: `gemini-1.5-flash` for fast inference
- **Embeddings**: `gemini-embedding-001` for semantic similarity
- **Grounding**: All responses include supporting context from tools/memory
- **Safety**: Medical disclaimers for diagnosis-like queries

---

## 💡 Design Highlights

### Why This Architecture?

| Design Choice | Benefit |
|---------------|---------|
| **Go Backend** | High concurrency, low latency, easy deployment |
| **React Frontend** | Responsive UI, real-time connection status |
| **Agent Pattern** | Explicit tool orchestration, interpretability |
| **Memory Embeddings** | Context continuity without large context windows |
| **Optional LangGraph** | Advanced multi-step workflows when needed |
| **Retry/Worker Jobs** | Fault tolerance, background processing |

### Safety & Responsibility

- ✅ **No Medical Diagnosis**: Agent avoids guessing; recommends professional care
- ✅ **Profile Grounding**: All responses cite user data
- ✅ **Transparent Tools**: Used tools are tracked in metadata
- ✅ **Memory Limits**: Respects context window constraints
- ✅ **Rate/Load Limits**: Background worker prevents overload

---

## 🧪 Testing & Examples

### Example Conversation Flow

**User:** "I've been feeling tired and have a bad headache."

**Agent Flow:**
1. Recognize as symptom query
2. Call `health_data` tool → retrieve BMI, vitals, vitamin levels
3. Call `memory` tool → retrieve past related conversations
4. Generate response using Gemini with grounded context
5. Include disclaimer: "I cannot diagnose. Please consult a healthcare professional."
6. Return: Possible factors + scheduled appointment info

**User:** "What foods should I avoid?"

**Agent Flow:**
1. Recognize as profile query
2. Call `profile_qa` tool → fetch allergies (peanuts, shellfish)
3. Call `user_data` tool → fetch diet preference (Mediterranean)
4. Generate response with context
5. Return: "Avoid peanuts, shellfish; focus on Mediterranean diet principles."

---

## 🚢 Deployment

### Docker Support (Optional)

You can containerize each service:

```dockerfile
# backend/Dockerfile
FROM golang:1.22-alpine
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o server main.go
EXPOSE 8080
CMD ["./server"]
```

```dockerfile
# frontend/Dockerfile
FROM node:18-alpine
WORKDIR /app
COPY package*.json ./
RUN npm install
COPY . .
RUN npm run build
EXPOSE 5173
CMD ["npm", "run", "dev"]
```

### Environment Variables (Production)

```bash
# Backend
GEMINI_API_KEY            # Keep secret; use secrets manager
SERPER_API_KEY            # Keep secret; use secrets manager
PORT                      # Default: 8080
GEMINI_MODEL              # Default: gemini-1.5-flash
GEMINI_EMBEDDING_MODEL    # Default: gemini-embedding-001
LANGGRAPH_API_URL         # Optional external workflow service

# Frontend
VITE_API_URL              # Backend API base URL
```

---

## 📚 Tech Stack

| Layer | Technology | Version |
|-------|-----------|---------|
| **Backend** | Go | 1.22+ |
| **Frontend** | React + TypeScript | 18+ |
| **Bundler** | Vite | 5.0+ |
| **LLM** | Google Gemini | API v1 |
| **Embeddings** | Gemini Embedding | 001 |
| **Web Search** | Serper API | v1 |
| **Workflow** | LangGraph (optional) | 0.1+ |
| **Styling** | CSS-in-JS / Tailwind | Standard |

---

## 🎓 Learning Resources

- **Go Concurrency**: [Effective Go](https://go.dev/doc/effective_go)
- **React Hooks**: [React Docs](https://react.dev)
- **LLM Prompting**: [OpenAI Prompt Guide](https://platform.openai.com/docs/guides/prompt-engineering)
- **Embeddings**: [Semantic Search](https://www.sbert.net/)
- **LangGraph**: [LangGraph Python Docs](https://python.langchain.com/docs/langgraph)

---

## 📝 License

MIT License — See LICENSE file for details

---

## 🤝 Contact & Support

For questions about this project:
- **Developed by**: Kuldeep Raj
- **Repository**: [GitHub - Health_chatBot_Golang](https://github.com/kuldeep31016/Health_chatBot_Golang)
- **Demo Video**: [YouTube](https://www.youtube.com/watch?v=YOUR_VIDEO_ID_HERE) *(Replace with your actual link)*

---

## 🔄 Future Improvements

- [ ] EHR/FHIR integration for real medical records
- [ ] Multi-user support with secure authentication
- [ ] Advanced medical NLP models for intent classification
- [ ] Voice input/output support
- [ ] Mobile app (React Native)
- [ ] Kubernetes deployment manifests
- [ ] Comprehensive test suite (unit + integration)
- [ ] Analytics dashboard for conversation insights

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

Author :
Kuldeep Raj
