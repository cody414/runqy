"""GPU Inference Worker — Stable Diffusion image generation with Runqy.

This example demonstrates GPU-native task processing:
- @load initializes the model once (expensive GPU setup)
- @task processes inference requests without restarting
- Long-running mode keeps the model in VRAM between tasks

Requirements: NVIDIA GPU with 4GB+ VRAM
"""

import sys
import base64
import io
from runqy_python import task, load, run


@load
def setup():
    """Load Stable Diffusion pipeline into GPU memory (runs once)."""
    print("Loading Stable Diffusion model...", file=sys.stderr)

    from diffusers import StableDiffusionPipeline
    import torch

    pipe = StableDiffusionPipeline.from_pretrained(
        "stable-diffusion-v1-5/stable-diffusion-v1-5",
        torch_dtype=torch.float16,
    ).to("cuda")

    # Optional: enable memory-efficient attention
    pipe.enable_attention_slicing()

    print("Model loaded and ready!", file=sys.stderr)
    return {"pipe": pipe}


@task
def process(payload: dict, ctx: dict) -> dict:
    """Generate an image from a text prompt.

    Input:
        prompt (str): Text description of the image
        negative_prompt (str, optional): What to avoid
        steps (int, optional): Inference steps (default: 30)
        width (int, optional): Image width (default: 512)
        height (int, optional): Image height (default: 512)

    Output:
        image_base64 (str): Generated image as base64-encoded PNG
        seed (int): Random seed used
    """
    import torch

    pipe = ctx["pipe"]

    prompt = payload.get("prompt", "a photo of a cat")
    negative_prompt = payload.get("negative_prompt", "")
    steps = payload.get("steps", 30)
    width = payload.get("width", 512)
    height = payload.get("height", 512)

    # Generate with reproducible seed
    seed = payload.get("seed", torch.randint(0, 2**32, (1,)).item())
    generator = torch.Generator("cuda").manual_seed(seed)

    result = pipe(
        prompt=prompt,
        negative_prompt=negative_prompt,
        num_inference_steps=steps,
        width=width,
        height=height,
        generator=generator,
    )

    # Encode image to base64
    image = result.images[0]
    buffer = io.BytesIO()
    image.save(buffer, format="PNG")
    image_base64 = base64.b64encode(buffer.getvalue()).decode()

    return {
        "image_base64": image_base64,
        "seed": seed,
    }


if __name__ == "__main__":
    run()
