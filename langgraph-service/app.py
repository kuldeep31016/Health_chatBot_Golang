import os
import json
import time
from pathlib import Path
from typing import Any, Dict, List, TypedDict, Optional
from urllib import parse, request, error

from fastapi import FastAPI
from langgraph.graph import END, StateGraph
from langchain_google_genai import ChatGoogleGenerativeAI


class GraphState(TypedDict):
    query: str
    history: List[Dict[str, Any]]
    context: Dict[str, Any]
    tasks: List[Dict[str, Any]]
    action_trace: List[str]
    status: str
    error: str
    started_at: float
    response: str


SYSTEM_PROMPT = (
    "You are a personal AI health assistant.\n"
    "You are given structured user data in 'Context'.\n"
    "You MUST treat this context as the user's real data.\n"
    "Always answer using this context when relevant.\n"
    "Never say you don't have access to user data.\n"
    "Do not hallucinate. If context is missing, say clearly.\n"
)

ROOT_DIR = Path(__file__).resolve().parent.parent
PROFILE_PATH = Path(os.getenv("USER_PROFILE_PATH", str(ROOT_DIR / "backend" / "data" / "user_profile.json")))
MAX_CONTEXT_CHARS = int(os.getenv("MAX_CONTEXT_CHARS", "7000"))

_PROFILE_CACHE: Dict[str, Any] = {
    "mtime": None,
    "data": None,
}
_WEB_CACHE: Dict[str, Dict[str, Any]] = {}


def _load_root_env() -> None:
    env_path = Path(__file__).resolve().parent.parent / ".env"
    if not env_path.exists():
        return

    for line in env_path.read_text(encoding="utf-8").splitlines():
        line = line.strip()
        if not line or line.startswith("#") or "=" not in line:
            continue
        key, value = line.split("=", 1)
        os.environ.setdefault(key.strip(), value.strip())


_load_root_env()


def _create_llm() -> ChatGoogleGenerativeAI:
    api_key = os.getenv("GEMINI_API_KEY", "").strip()
    if not api_key:
        raise ValueError("GEMINI_API_KEY is missing. Set it in project .env or shell env.")

    return ChatGoogleGenerativeAI(
        model=os.getenv("GEMINI_MODEL", "gemini-2.5-flash"),
        google_api_key=api_key,
        temperature=0.2,
    )


def _get_profile_data() -> Dict[str, Any]:
    if not PROFILE_PATH.exists():
        return {}

    mtime = PROFILE_PATH.stat().st_mtime
    cached_mtime = _PROFILE_CACHE.get("mtime")
    cached_data = _PROFILE_CACHE.get("data")

    if cached_data is not None and cached_mtime == mtime:
        return cached_data

    try:
        parsed = json.loads(PROFILE_PATH.read_text(encoding="utf-8"))
        if isinstance(parsed, dict):
            _PROFILE_CACHE["mtime"] = mtime
            _PROFILE_CACHE["data"] = parsed
            return parsed
    except Exception:
        return {}

    return {}


def _trim_context_for_prompt(context: Dict[str, Any], max_chars: int) -> Dict[str, Any]:
    raw = json.dumps(context, ensure_ascii=False)
    if len(raw) <= max_chars:
        return context

    trimmed = dict(context)
    if isinstance(trimmed.get("web"), dict):
        web = dict(trimmed["web"])
        results = web.get("top_results", [])
        if isinstance(results, list):
            web["top_results"] = results[:2]
        trimmed["web"] = web

    if isinstance(trimmed.get("memory"), dict):
        mem = dict(trimmed["memory"])
        items = mem.get("items", [])
        if isinstance(items, list):
            mem["items"] = items[:2]
        trimmed["memory"] = mem

    raw_trimmed = json.dumps(trimmed, ensure_ascii=False)
    if len(raw_trimmed) <= max_chars:
        return trimmed

    return {
        "notice": f"Context was too large and was summarized to fit {max_chars} chars.",
        "user": trimmed.get("user", {}),
        "health": trimmed.get("health", {}),
    }


def _select_user_basics(profile: Dict[str, Any]) -> Dict[str, Any]:
    keys = [
        "name",
        "age",
        "height_cm",
        "weight_kg",
        "hair_color",
        "eye_color",
        "gender",
        "blood_group",
    ]
    return {k: profile[k] for k in keys if k in profile}


def _select_health_sections(profile: Dict[str, Any], query: str) -> Dict[str, Any]:
    q = query.lower()
    out: Dict[str, Any] = {}

    if any(t in q for t in ["heart", "hrv", "bp", "blood pressure"]):
        if "cardiovascular_matrix" in profile:
            out["cardiovascular_matrix"] = profile["cardiovascular_matrix"]

    if any(t in q for t in ["glucose", "metabolic", "dizzy", "weak", "fatigue"]):
        if "metabolic" in profile:
            out["metabolic"] = profile["metabolic"]
        if "biomarkers" in profile:
            out["biomarkers"] = profile["biomarkers"]

    if any(t in q for t in ["exercise", "workout", "fitness", "running", "run"]):
        if "fitness_milestones" in profile:
            out["fitness_milestones"] = profile["fitness_milestones"]
        if "workout_preferences" in profile:
            out["workout_preferences"] = profile["workout_preferences"]

    if any(t in q for t in ["diet", "hydrate", "hydration", "nutrition"]):
        if "diet_preferences" in profile:
            out["diet_preferences"] = profile["diet_preferences"]

    if any(t in q for t in ["allergy", "allergies"]):
        if "allergies" in profile:
            out["allergies"] = profile["allergies"]

    if any(t in q for t in ["appointment", "schedule", "visit"]):
        if "scheduled_appointments" in profile:
            out["scheduled_appointments"] = profile["scheduled_appointments"]

    return out


def _web_search(query: str) -> Dict[str, Any]:
    cache_key = query.strip().lower()
    if cache_key in _WEB_CACHE:
        return _WEB_CACHE[cache_key]

    api_key = os.getenv("SERPER_API_KEY", "").strip()
    if not api_key:
        return {"error": "SERPER_API_KEY is missing"}

    endpoint = f"https://google.serper.dev/search?q={parse.quote(query)}"
    last_error = None

    for _ in range(3):
        try:
            req = request.Request(endpoint, method="GET")
            req.add_header("X-API-KEY", api_key)
            with request.urlopen(req, timeout=12) as resp:
                body = resp.read().decode("utf-8")
                if resp.status >= 300:
                    raise RuntimeError(f"web search status {resp.status}: {body}")
                parsed = json.loads(body)

                organic = parsed.get("organic", [])
                top_results: List[Dict[str, str]] = []
                for item in organic[:3]:
                    if not isinstance(item, dict):
                        continue
                    top_results.append(
                        {
                            "title": str(item.get("title", "")),
                            "link": str(item.get("link", "")),
                            "snippet": str(item.get("snippet", "")),
                        }
                    )

                result = {
                    "query": query,
                    "top_results": top_results,
                }
                _WEB_CACHE[cache_key] = result
                return result
        except (error.URLError, TimeoutError, json.JSONDecodeError, RuntimeError) as exc:
            last_error = exc
            time.sleep(1.5)

    result = {"error": f"web search failed: {last_error}"}
    _WEB_CACHE[cache_key] = result
    return result


def build_graph() -> Any:
    def process_node(state: GraphState) -> GraphState:
        # Normalize incoming state to keep graph execution robust.
        state["query"] = str(state.get("query", "")).strip()

        history = state.get("history", [])
        state["history"] = history if isinstance(history, list) else []

        context = state.get("context", {})
        state["context"] = context if isinstance(context, dict) else {}
        tasks = state.get("tasks", [])
        state["tasks"] = tasks if isinstance(tasks, list) else []
        action_trace = state.get("action_trace", [])
        state["action_trace"] = action_trace if isinstance(action_trace, list) else []
        state["status"] = "process"
        state["error"] = ""
        started = state.get("started_at", 0.0)
        state["started_at"] = float(started) if isinstance(started, (int, float)) else time.time()

        return state

    def tasks_node(state: GraphState) -> GraphState:
        query = state.get("query", "").lower()
        context = state.get("context", {})
        tasks: List[Dict[str, Any]] = []

        wants_profile = any(token in query for token in ["age", "weight", "height", "hair", "eye", "profile", "who am i"])
        wants_health = any(token in query for token in ["exercise", "workout", "fitness", "health", "hrv", "glucose", "bp", "heart", "diet", "hydration", "appointment"])
        wants_memory = any(token in query for token in ["earlier", "remember", "before", "history", "last time"])
        wants_web = any(token in query for token in ["latest", "news", "research", "web", "search", "what is", "guideline"])

        if wants_profile and not isinstance(context.get("user"), dict):
            tasks.append({"name": "fetch_user_data", "reason": "profile query trigger"})
        if wants_health and not isinstance(context.get("health"), dict):
            tasks.append({"name": "fetch_health_data", "reason": "health query trigger"})
        if wants_memory and "memory" not in context:
            tasks.append({"name": "prepare_memory_context", "reason": "memory query trigger"})
        if wants_web:
            tasks.append({"name": "web_search", "reason": "web query trigger"})

        if not tasks:
            tasks.append({"name": "no_op", "reason": "no external context needed"})

        state["tasks"] = tasks
        state["status"] = "tasks"
        return state

    def action_node(state: GraphState) -> GraphState:
        context = dict(state.get("context", {}))
        tasks = list(state.get("tasks", []))
        if not tasks:
            state["context"] = context
            state["status"] = "success"
            return state

        current_task = tasks.pop(0)
        task_name = str(current_task.get("name", ""))
        state.setdefault("action_trace", []).append(task_name)

        try:
            profile = _get_profile_data()
            query = state.get("query", "")

            if task_name == "fetch_user_data":
                if not isinstance(context.get("user"), dict):
                    context["user"] = _select_user_basics(profile)

            elif task_name == "fetch_health_data":
                if not isinstance(context.get("health"), dict):
                    context["health"] = _select_health_sections(profile, query)

            elif task_name == "prepare_memory_context":
                # Uses already-available context/history (trigger-based, no unnecessary fetch).
                if "memory" not in context:
                    tail = state.get("history", [])[-6:]
                    context["memory"] = {"items": tail}

            elif task_name == "web_search":
                context["web"] = _web_search(query)

            elif task_name == "no_op":
                pass

            state["status"] = "action"
        except Exception as exc:
            state["error"] = str(exc)
            state["status"] = "fail"

        state["context"] = context
        state["tasks"] = tasks
        return state

    def route_after_action(state: GraphState) -> str:
        if state.get("status") == "fail":
            return "fail"
        pending = state.get("tasks", [])
        return "action" if pending else "success"

    def success_node(state: GraphState) -> GraphState:
        safe_context = _trim_context_for_prompt(state.get("context", {}), MAX_CONTEXT_CHARS)
        prompt = (
            f"{SYSTEM_PROMPT}\n\n"
            f"Executed Tasks: {state.get('action_trace', [])}\n"
            f"Context: {safe_context}\n"
            f"History: {state.get('history', [])}\n"
            f"User Query: {state.get('query', '')}"
        )
        try:
            llm = _create_llm()
            msg = llm.invoke(prompt)
            state["response"] = msg.content if hasattr(msg, "content") else str(msg)
            state["status"] = "success"
        except Exception as exc:
            state["error"] = str(exc)
            state["status"] = "fail"
        return state

    def fail_node(state: GraphState) -> GraphState:
        detail = state.get("error", "unknown error")
        state["response"] = (
            "I'm having trouble right now. Please try again in a moment. "
            f"(LangGraph detail: {detail})"
        )
        state["status"] = "fail"
        return state

    graph = StateGraph(GraphState)
    graph.add_node("process", process_node)
    graph.add_node("tasks", tasks_node)
    graph.add_node("action", action_node)
    graph.add_node("success", success_node)
    graph.add_node("fail", fail_node)

    graph.set_entry_point("process")
    graph.add_edge("process", "tasks")
    graph.add_edge("tasks", "action")
    graph.add_conditional_edges("action", route_after_action, {"action": "action", "success": "success", "fail": "fail"})
    graph.add_edge("success", END)
    graph.add_edge("fail", END)

    return graph.compile()


app = FastAPI(title="LangGraph Health Agent")
chain = build_graph()


@app.get("/health")
def health() -> Dict[str, str]:
    return {"status": "ok"}


@app.post("/run")
def run(payload: Dict[str, Any]) -> Dict[str, Any]:
    query = payload.get("query", "")
    history = payload.get("history", [])
    context = payload.get("context", {})
    started = time.time()

    out = chain.invoke(
        {
            "query": query,
            "history": history,
            "context": context,
            "tasks": [],
            "action_trace": [],
            "status": "process",
            "error": "",
            "started_at": started,
            "response": "",
        }
    )

    web_ctx: Optional[Dict[str, Any]] = out.get("context", {}).get("web") if isinstance(out.get("context"), dict) else None
    sources = []
    if isinstance(web_ctx, dict):
        for item in web_ctx.get("top_results", [])[:3]:
            if isinstance(item, dict) and item.get("link"):
                sources.append(item.get("link"))

    return {
        "response": out.get("response", "I'm having trouble right now. Please try again in a moment."),
        "status": out.get("status", "unknown"),
        "meta": {
            "executed_tasks": out.get("action_trace", []),
            "latency_ms": int((time.time() - started) * 1000),
            "sources": sources,
        },
    }
