# GPU Inference Example — Stable Diffusion

Generate images from text prompts using Stable Diffusion on GPU workers.

This demonstrates Runqy's **GPU-native task processing** — the model loads once into VRAM and stays resident between tasks, eliminating cold-start latency.

## Requirements

- NVIDIA GPU with 8GB+ VRAM
- CUDA drivers installed

## Enqueue a task

```bash
runqy task enqueue -q gpu-inference -p '{
  "prompt": "a cyberpunk fox in neon city, digital art",
  "steps": 30,
  "width": 512,
  "height": 512
}'
```

## Why this matters

Traditional setup: spin up GPU instance → load model (30s+) → run inference → tear down.

With Runqy: model stays loaded in VRAM. Cold start once, then sub-second task pickup. **10x cheaper** for bursty workloads.
