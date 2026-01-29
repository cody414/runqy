"""Long-running task for runqy Quick Start.

This task demonstrates the long-running mode where:
- @load runs once at startup (e.g., load ML models)
- @task processes multiple tasks without restarting
- run() keeps the process alive between tasks
"""

import sys
from runqy_python import task, load, run


class SimpleProcessor:
    """Simulates a resource that's expensive to initialize."""

    def __init__(self):
        self.call_count = 0

    def process(self, operation: str, data):
        self.call_count += 1
        if operation == "echo":
            return data
        elif operation == "uppercase":
            return data.upper() if isinstance(data, str) else data
        elif operation == "double":
            return data * 2 if isinstance(data, (int, float)) else data
        elif operation == "count":
            return self.call_count
        else:
            raise ValueError(f"Unknown operation: {operation}")


@load
def setup():
    """Initialize resources (runs once before ready signal)."""
    print("Initializing processor...", file=sys.stderr)
    processor = SimpleProcessor()
    print("Processor ready!", file=sys.stderr)
    return {"processor": processor}


@task
def process(payload: dict, ctx: dict) -> dict:
    """Process a task using the initialized processor."""
    processor = ctx["processor"]
    operation = payload.get("operation", "echo")
    data = payload.get("data")

    try:
        result = processor.process(operation, data)
        return {"result": result, "calls": processor.call_count}
    except ValueError as e:
        return {"error": str(e)}


if __name__ == "__main__":
    run()
