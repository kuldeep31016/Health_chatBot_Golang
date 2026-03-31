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
    "You are a personal AI health assistant. Use ONLY the provided user data and memory "
    "context to answer. Do NOT hallucinate. If data is not available, say so clearly."
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
        return state

    def decide_node(state: GraphState) -> GraphState:
        return state

    def action_node(state: GraphState) -> GraphState:
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
