"""One-shot task for runqy Quick Start.

This task demonstrates the one-shot mode where:
- @task processes a single task
- run_once() exits after processing
- A new process is spawned for each task
"""

from runqy_python import task, run_once


@task
def process(payload: dict) -> dict:
    """Process a single task and exit."""
    operation = payload.get("operation", "echo")
    data = payload.get("data")

    if operation == "echo":
        return {"result": data}
    elif operation == "uppercase":
        return {"result": data.upper() if isinstance(data, str) else data}
    elif operation == "double":
        return {"result": data * 2 if isinstance(data, (int, float)) else data}
    else:
        return {"error": f"Unknown operation: {operation}"}


if __name__ == "__main__":
    run_once()
