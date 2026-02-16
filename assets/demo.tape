# Runqy Demo — VHS Script
# Generate with: vhs assets/demo.tape

Output assets/demo.gif
Set FontSize 14
Set Width 1200
Set Height 600
Set Theme "Catppuccin Mocha"
Set TypingSpeed 40ms
Set Padding 20

# Title
Type "# Runqy — Task Queue in 60 seconds"
Enter
Sleep 1s

# Step 1: Start the stack
Type "# 1. Start the stack"
Enter
Sleep 500ms
Type "docker-compose -f docker-compose.quickstart.yml up -d"
Enter
Sleep 5s

# Step 2: Check services are running
Type "# 2. Stack is ready ✓"
Enter
Sleep 500ms
Type "docker-compose -f docker-compose.quickstart.yml ps --format 'table {{.Service}}\t{{.Status}}'"
Enter
Sleep 3s

# Step 3: Show the queue config
Type "# 3. Queue defined in YAML — zero-touch deployment from Git"
Enter
Sleep 500ms
Type "cat examples/quickstart.yaml | head -20"
Enter
Sleep 3s

# Step 4: Enqueue a task via REST API
Type "# 4. Enqueue a task"
Enter
Sleep 500ms
Type@50ms 'curl -s -X POST http://localhost:3000/enqueue \' 
Enter
Type@50ms '  -H "Content-Type: application/json" \'
Enter
Type@50ms '  -H "Authorization: Bearer dev-api-key" \'
Enter
Type@50ms '  -d '"'"'{"queue":"quickstart-oneshot","data":{"operation":"uppercase","data":"hello runqy"}}'"'"' | jq .'
Enter
Sleep 3s

# Step 5: Enqueue with Python SDK
Type "# 5. Or use the Python SDK"
Enter
Sleep 500ms
Type "python3 -c \""
Enter
Type "from runqy_python import Client"
Enter
Type "client = Client('http://localhost:3000', api_key='dev-api-key')"
Enter
Type "task = client.enqueue('quickstart-longrunning', {'operation': 'double', 'data': 42})"
Enter
Type "print(f'Task {task.id} enqueued!')"
Enter
Type "\""
Enter
Sleep 3s

# Step 6: Check results
Type "# 6. Monitor at http://localhost:3000/monitoring/"
Enter
Sleep 500ms
Type "curl -s http://localhost:3000/workers -H 'Authorization: Bearer dev-api-key' | jq '.[] | {queue, status}'"
Enter
Sleep 3s

# Closing
Type ""
Enter
Type "# Your workers, your machines, your rules. 🚀"
Enter
Type "# → github.com/Publikey/runqy"
Enter
Sleep 3s
