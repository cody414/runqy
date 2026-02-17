#!/bin/bash
sleep 0.3
echo "[INFO] Bootstrap attempt 1/3: contacting http://localhost:3000/worker/register"
sleep 0.3
echo "[INFO] Bootstrap successful: redis=localhost:6379, queue=image-resize"
sleep 0.2
echo "[INFO] Deploying code from https://github.com/acme/image-worker.git (branch: main)..."
sleep 0.3
echo "[INFO] Sparse checkout successful (only downloaded: src/)"
sleep 0.2
echo "[INFO] Installing dependencies with pip..."
sleep 0.5
echo "[INFO] Dependencies installed successfully"
sleep 0.2
echo "[INFO] [STDERR] Initializing processor..."
echo "[INFO] [STDERR] Processor ready!"
echo "[INFO] Process startup detected - service is ready"
echo "[INFO] Bootstrap complete with 1 sub-queue (priority=5)"
echo "[INFO] Worker ready — listening for tasks"
