"""Multi-Queue Priority Router — Demonstrate queue routing with Runqy.

This example shows how a single worker codebase can handle
multiple queue types with different priorities and behaviors.

The YAML config (see examples.yaml) defines priority levels,
and Runqy routes tasks to the right workers automatically.
"""

import sys
import time
from runqy_python import task, load, run


@load
def setup():
    """Initialize shared resources."""
    print("Multi-queue worker starting...", file=sys.stderr)
    return {
        "start_time": time.time(),
        "processed": {"critical": 0, "standard": 0, "bulk": 0},
    }


@task
def process(payload: dict, ctx: dict) -> dict:
    """Process a task routed from any queue.

    Input:
        action (str): What to do ("notify", "process", "index")
        data (dict): Action-specific data
        priority_override (str, optional): For demo purposes

    Output:
        result (str): Processing result
        queue_stats (dict): Tasks processed per priority
    """
    action = payload.get("action", "process")
    data = payload.get("data", {})

    # Simulate different processing times by action type
    if action == "notify":
        # Critical: send notification immediately
        result = send_notification(data)
    elif action == "process":
        # Standard: normal processing
        result = process_data(data)
    elif action == "index":
        # Bulk: batch indexing, can be slow
        result = index_data(data)
    else:
        result = f"Unknown action: {action}"

    # Track stats
    priority = payload.get("_priority", "standard")
    ctx["processed"][priority] = ctx["processed"].get(priority, 0) + 1

    return {
        "result": result,
        "queue_stats": ctx["processed"],
    }


def send_notification(data: dict) -> str:
    """Simulate sending an urgent notification."""
    channel = data.get("channel", "email")
    recipient = data.get("to", "user@example.com")
    message = data.get("message", "Alert!")
    # In practice: call email/SMS/push API
    return f"Notification sent via {channel} to {recipient}: {message}"


def process_data(data: dict) -> str:
    """Simulate standard data processing."""
    items = data.get("items", [])
    operation = data.get("operation", "transform")
    # Simulate work
    time.sleep(0.1)
    return f"Processed {len(items)} items with {operation}"


def index_data(data: dict) -> str:
    """Simulate bulk indexing (slower, lower priority)."""
    documents = data.get("documents", [])
    index_name = data.get("index", "default")
    # Simulate slower bulk work
    time.sleep(0.5)
    return f"Indexed {len(documents)} documents into '{index_name}'"


if __name__ == "__main__":
    run()
