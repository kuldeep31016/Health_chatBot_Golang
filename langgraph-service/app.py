import os
import json
import time
import logging
import re
import warnings
from pathlib import Path
from typing import Any, Dict, List, TypedDict, Optional
from urllib import parse, request, error

from fastapi import FastAPI

warnings.filterwarnings("ignore", category=FutureWarning, module=r"langchain_google_genai\..*")
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
    "Use plain simple text only.\n"
    "Do NOT use markdown symbols like *, **, #, -, or backticks.\n"
    "Do NOT use bullet points.\n"
    "Keep output clean, readable, and professional.\n"
)

ROOT_DIR = Path(__file__).resolve().parent.parent
PROFILE_PATH = Path(os.getenv("USER_PROFILE_PATH", str(ROOT_DIR / "backend" / "data" / "user_profile.json")))
MAX_CONTEXT_CHARS = int(os.getenv("MAX_CONTEXT_CHARS", "7000"))
DEFAULT_GEMINI_MODEL = os.getenv("GEMINI_MODEL", "gemini-2.0-flash").strip() or "gemini-2.0-flash"
FALLBACK_GEMINI_MODEL = os.getenv("GEMINI_FALLBACK_MODEL", "gemini-2.5-flash").strip() or "gemini-2.5-flash"

_PROFILE_CACHE: Dict[str, Any] = {
    "mtime": None,
    "data": None,
}
_WEB_CACHE: Dict[str, Dict[str, Any]] = {}

logging.basicConfig(level=logging.INFO, format="%(asctime)s | %(levelname)s | %(message)s")
logger = logging.getLogger("langgraph-health-agent")


def _clean_response_text(text: str) -> str:
    if not text:
        return ""

    cleaned = text.replace("\r", "")
    cleaned = re.sub(r"^\s{0,3}#{1,6}\s?", "", cleaned, flags=re.MULTILINE)
    cleaned = re.sub(r"\*\*(.*?)\*\*", r"\1", cleaned)
    cleaned = re.sub(r"__(.*?)__", r"\1", cleaned)
    cleaned = re.sub(r"`([^`]*)`", r"\1", cleaned)
    cleaned = re.sub(r"^\s*[-*]\s+", "", cleaned, flags=re.MULTILINE)
    cleaned = re.sub(r"^\s*\d+[\.)]\s+", "", cleaned, flags=re.MULTILINE)
    cleaned = re.sub(r"\n{3,}", "\n\n", cleaned)
    return cleaned.strip()


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
        model=DEFAULT_GEMINI_MODEL,
        google_api_key=api_key,
        temperature=0.2,
    )


def _create_llm_for_model(model: str) -> ChatGoogleGenerativeAI:
    api_key = os.getenv("GEMINI_API_KEY", "").strip()
    if not api_key:
        raise ValueError("GEMINI_API_KEY is missing. Set it in project .env or shell env.")

    return ChatGoogleGenerativeAI(
        model=model,
        google_api_key=api_key,
        temperature=0.2,
    )


def _is_quota_error(exc: Exception) -> bool:
    text = str(exc).lower()
    return any(
        marker in text
        for marker in [
            "resourceexhausted",
            "quota exceeded",
            "rate limit",
            "too many requests",
            "429",
            "not found",
            "not supported",
            "404",
        ]
    )


def _invoke_llm_with_fallback(prompt: str) -> str:
    models = []
    for model in [DEFAULT_GEMINI_MODEL, FALLBACK_GEMINI_MODEL, "gemini-2.0-flash", "gemini-2.5-flash"]:
        if model and model not in models:
            models.append(model)

    last_error: Optional[Exception] = None
    for model in models:
        try:
            llm = _create_llm_for_model(model)
            msg = llm.invoke(prompt)
            raw = msg.content if hasattr(msg, "content") else str(msg)
            if raw:
                if model != DEFAULT_GEMINI_MODEL:
                    logger.warning("Gemini fallback succeeded using model=%s", model)
                return _clean_response_text(raw)
        except Exception as exc:
            last_error = exc
            if model != models[-1] and _is_quota_error(exc):
                logger.warning("Gemini model failed on %s; retrying with fallback model", model)
                continue
            raise

    if last_error is not None:
        raise last_error
    raise RuntimeError("LLM invocation failed")


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
    return dict(profile)


def _select_health_sections(profile: Dict[str, Any], query: str) -> Dict[str, Any]:
    q = query.lower()
    out: Dict[str, Any] = {}

    if any(t in q for t in ["heart", "hrv", "bp", "blood pressure"]):
        if "cardiovascular_matrix" in profile:
            out["cardiovascular_matrix"] = profile["cardiovascular_matrix"]

    if any(t in q for t in ["glucose", "metabolic", "dizzy", "weak", "fatigue", "hemoglobin", "haemoglobin", "hb", "biomarker", "vitamin"]):
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
        logger.info("Serper API working fine (cache hit) | query='%s'", query[:80])
        return _WEB_CACHE[cache_key]

    api_key = os.getenv("SERPER_API_KEY", "").strip()
    if not api_key:
        logger.warning("Serper API key missing (SERPER_API_KEY not set)")
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
                logger.info(
                    "Serper API working fine | query='%s' | results=%d",
                    query[:80],
                    len(top_results),
                )
                return result
        except (error.URLError, TimeoutError, json.JSONDecodeError, RuntimeError) as exc:
            last_error = exc
            time.sleep(1.5)

    result = {"error": f"web search failed: {last_error}"}
    _WEB_CACHE[cache_key] = result
    logger.error("Serper API failed after retries | query='%s' | error=%s", query[:80], last_error)
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
        wants_health = any(token in query for token in ["exercise", "workout", "fitness", "health", "hrv", "glucose", "bp", "heart", "diet", "hydration", "appointment", "hemoglobin", "haemoglobin", "hb", "biomarker", "vitamin", "dizzy", "weak", "fatigue", "tired"])
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
            state["response"] = _invoke_llm_with_fallback(prompt)
            state["status"] = "success"
        except Exception as exc:
            state["error"] = str(exc)
            state["status"] = "fail"
        return state

    def fail_node(state: GraphState) -> GraphState:
        detail = state.get("error", "unknown error")
        state["response"] = _clean_response_text(
            (
            "I'm having trouble right now. Please try again in a moment. "
            f"(LangGraph detail: {detail})"
            )
        )
        state["status"] = "fail"
        return state

    graph = StateGraph(GraphState)
    graph.add_node("process", process_node)
    graph.add_node("plan_tasks", tasks_node)
    graph.add_node("action", action_node)
    graph.add_node("success", success_node)
    graph.add_node("fail", fail_node)

    graph.set_entry_point("process")
    graph.add_edge("process", "plan_tasks")
    graph.add_edge("plan_tasks", "action")
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
