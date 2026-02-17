# Runqy Demo v8 — Full storytelling flow
Output "/tmp/demo-v8/demo.webm"
Set FontSize 14
Set Width 1200
Set Height 700
Set Theme "Catppuccin Mocha"
Set TypingSpeed 30ms
Set Padding 15
Set Framerate 24

# Load mock commands
Hide
Type "source /tmp/demo-v8/mock-bashrc.sh"
Enter
Sleep 500ms
Type "clear"
Enter
Show
Sleep 500ms

# ── Scene 1: Start the server ──
Type "# Start the Runqy server"
Enter
Sleep 1.5s
Type "runqy serve"
Enter
Sleep 4s

# Transition (no extra wait for first one)
Sleep 2s
Hide
Type "clear"
Enter
Show
Sleep 500ms

# ── Scene 2: Create a queue config ──
Type "# Define a queue — what work looks like"
Enter
Sleep 1.5s
Hide
Type "cp /tmp/demo-v8/queue.yaml ./queue.yaml"
Enter
Sleep 300ms
Type "clear"
Enter
Show
Sleep 500ms
Type "cat queue.yaml"
Enter
Sleep 4s

# Transition (+2s reading time)
Sleep 2s
Hide
Type "clear"
Enter
Show
Sleep 500ms

# ── Scene 3: Deploy the queue ──
Type "# Deploy the queue to the server"
Enter
Sleep 1.5s
Type "runqy config create -f queue.yaml"
Enter
Sleep 2s

# Transition (+2s reading time)
Sleep 2s
Hide
Type "clear"
Enter
Show
Sleep 500ms

# ── Scene 4: Start a worker ──
Type "# Start a worker — can run anywhere (local, VM, Docker)"
Enter
Sleep 1s
Type "# Python code is deployed and executed automatically on the worker"
Enter
Sleep 1s
Type "# Pre-built Docker images available: ghcr.io/publikey/runqy-worker"
Enter
Sleep 1.5s
Type "runqy-worker --server http://localhost:3000 --queue image-resize --api-key dev-api-key"
Enter
Sleep 5s

# Transition (+2s reading time)
Sleep 2s
Hide
Type "clear"
Enter
Show
Sleep 500ms

# ── Scene 5: Check everything is ready ──
Type "# Check: queue deployed, worker connected"
Enter
Sleep 1.5s
Type "runqy queue list"
Enter
Sleep 2s
Type "runqy worker list"
Enter
Sleep 2.5s

# Transition (+2s reading time)
Sleep 2s
Hide
Type "clear"
Enter
Show
Sleep 500ms

# ── Scene 6: Enqueue tasks ──
Type "# Enqueue tasks — CLI, REST API, or Python SDK"
Enter
Sleep 1.5s

# CLI
Type `runqy task enqueue -q image-resize -p '{"image":"img001.jpg","size":256}'`
Enter
Sleep 2s

# REST API
Type `curl -s POST localhost:3000/queue/add -H "X-API-Key: dev-api-key" -d '{"queue":"image-resize","data":{"image":"img002.jpg"}}' | jq .`
Enter
Sleep 2s

# Python SDK
Type `python3 -c "from runqy_python import RunqyClient; c = RunqyClient('http://localhost:3000', api_key='dev-api-key'); t = c.enqueue('image-resize', {'image':'img003.jpg'}); print(f'Enqueued: {t.task_id}')"`
Enter
Sleep 2.5s

# Transition (+2s reading time)
Sleep 2s
Hide
Type "clear"
Enter
Show
Sleep 500ms

# ── Scene 7: Check the result ──
Type "# Get the result"
Enter
Sleep 1.5s
Type "runqy task get image-resize a2df37ba-77a2-409c-8664-799c3e22cf07"
Enter
Sleep 3s

# Closing — fully hidden, only output visible
Sleep 2s
Hide
Type "clear && closing"
Enter
Sleep 1s
Show
Sleep 3s
