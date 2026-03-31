import os
from pathlib import Path
from typing import Any, Dict, List, TypedDict

from fastapi import FastAPI
from langgraph.graph import END, StateGraph
from langchain_google_genai import ChatGoogleGenerativeAI


class GraphState(TypedDict):
    query: str
    history: List[Dict[str, Any]]
    context: Dict[str, Any]
    response: str


SYSTEM_PROMPT = (
    "You are a personal AI health assistant.\n"
    "You are given structured user data in 'Context'.\n"
    "You MUST treat this context as the user's real data.\n"
    "Always answer using this context when relevant.\n"
    "Never say you don't have access to user data.\n"
    "Do not hallucinate. If context is missing, say clearly.\n"
)


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


def build_graph() -> Any:
    def process_node(state: GraphState) -> GraphState:
        # Normalize incoming state to keep graph execution robust.
        state["query"] = str(state.get("query", "")).strip()

        history = state.get("history", [])
        state["history"] = history if isinstance(history, list) else []

        context = state.get("context", {})
        state["context"] = context if isinstance(context, dict) else {}

        return state

    def decide_node(state: GraphState) -> GraphState:
        query = state.get("query", "").lower()

        if any(token in query for token in ["age", "weight", "height"]):
            action = "fetch_user_profile"
        elif any(token in query for token in ["exercise", "workout", "fitness", "health"]):
            action = "fetch_health_profile"
        else:
            action = "general_response"

        state["context"] = {
            **state.get("context", {}),
            "_action": action,
        }
        return state

    def action_node(state: GraphState) -> GraphState:
        context = dict(state.get("context", {}))
        action = context.get("_action", "general_response")
        has_user_context = isinstance(context.get("user"), dict) and len(context.get("user", {})) > 0
        has_health_context = isinstance(context.get("health"), dict) and len(context.get("health", {})) > 0

        # Prefer trusted context provided by the backend. Only use fallback simulation
        # when required context is missing.
        if action == "fetch_user_profile" and not has_user_context:
            context["user"] = {
                "age": 22,
                "weight": 70,
                "height": 175,
            }
        elif action == "fetch_health_profile" and not has_health_context:
            context["health"] = {
                "goal": "fitness",
                "condition": "normal",
            }
        elif action == "general_response" and not context:
            context["general"] = "No specific data required"

        # Remove internal routing marker before sending context to the LLM.
        context.pop("_action", None)
        state["context"] = context
        return state

    def respond_node(state: GraphState) -> GraphState:
        prompt = (
            f"{SYSTEM_PROMPT}\n\n"
            f"Context: {state.get('context', {})}\n"
            f"History: {state.get('history', [])}\n"
            f"User Query: {state.get('query', '')}"
        )
        try:
            llm = _create_llm()
            msg = llm.invoke(prompt)
            state["response"] = msg.content if hasattr(msg, "content") else str(msg)
        except Exception as exc:
            state["response"] = (
                "I'm having trouble right now. Please try again in a moment. "
                f"(LangGraph detail: {exc})"
            )
        return state

    graph = StateGraph(GraphState)
    graph.add_node("process", process_node)
    graph.add_node("decide", decide_node)
    graph.add_node("action", action_node)
    graph.add_node("respond", respond_node)

    graph.set_entry_point("process")
    graph.add_edge("process", "decide")
    graph.add_edge("decide", "action")
    graph.add_edge("action", "respond")
    graph.add_edge("respond", END)

    return graph.compile()


app = FastAPI(title="LangGraph Health Agent")
chain = build_graph()


@app.get("/health")
def health() -> Dict[str, str]:
    return {"status": "ok"}


@app.post("/run")
def run(payload: Dict[str, Any]) -> Dict[str, str]:
    query = payload.get("query", "")
    history = payload.get("history", [])
    context = payload.get("context", {})

    out = chain.invoke(
        {
            "query": query,
            "history": history,
            "context": context,
            "response": "",
        }
    )
    return {"response": out.get("response", "I'm having trouble right now. Please try again in a moment.")}
