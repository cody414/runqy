# Runqy Demo v7 — Clean binary startup
Output "/tmp/runqy-review/assets/demo.webm"
Set FontSize 16
Set Width 1400
Set Height 800
Set Theme "Catppuccin Mocha"
Set TypingSpeed 25ms
Set Padding 15
Set Framerate 24

# Load mock commands
Hide
Type "source /tmp/demo-bashrc.sh"
Enter
Sleep 500ms
Type "clear"
Enter
Show
Sleep 500ms

# Scene 1: Start server
Type "runqy serve"
Enter
Sleep 4s

# Clear for worker
Hide
Type "clear"
Enter
Show
Sleep 500ms

# Scene 2: Start worker
Type "runqy-worker --server http://localhost:3000"
Enter
Sleep 4s

# Clear for CLI
Hide
Type "clear"
Enter
Show
Sleep 500ms

# Scene 3: Enqueue via CLI
Type `runqy task enqueue -q quickstart-oneshot -p '{"operation":"uppercase","data":"hello runqy"}'`
Enter
Sleep 2.5s

# Scene 4: REST API
Type `curl -s POST localhost:3000/queue/add -d '{"queue":"quickstart-oneshot","data":{"msg":"via REST"}}' | jq .`
Enter
Sleep 2.5s

# Scene 5: Python SDK
Type `python3 -c "from runqy_python import RunqyClient; c = RunqyClient('http://localhost:3000'); t = c.enqueue('quickstart-longrunning', {'data': 42}); print(f'Enqueued: {t.task_id}')"`
Enter
Sleep 2.5s

# Scene 6: Monitor
Type "runqy queue list"
Enter
Sleep 3s

# Closing
Type ""
Enter
Type `echo "🚀 Your workers, your machines, your rules. — github.com/Publikey/runqy"`
Enter
Sleep 3s
