package models

import (
	"encoding/json"
	"time"
)

type APIErrorResponse struct {
	Errors []string `json:"errors,omitempty"`
}

// GenericTask is a generalized task structure with only queue and timeout as required fields.
// The payload is stored as raw JSON to allow any structure.
type GenericTask struct {
	Queue   string          `json:"queue"`
	Timeout int64           `json:"timeout"`
	Data    json.RawMessage `json:"data" swaggertype:"object"`
}
type Predict struct {
	UserId     int64   `json:"user_id" `
	PredictId  int64   `json:"predict_id"`
	Status     string  `json:"status"`
	Queue      string  `json:"queue"`
	Timeout    int64   `json:"timeout"`
	Provider   string  `json:"provider,omitempty"` // "azure", "openai", "google", "worker" or empty (default: worker)
	ModelInput Request `json:"model_input"`
	EnqueuedAt int64   `json:"enqueued_at"`
	WebhookUrl string  `json:"webhook_url"`
}

// Request contains the model input parameters for image/video generation.
// Fields are provider-agnostic: send any fields you have, unused ones are ignored.
// Providers: "worker" (SDXL/external workers), "azure" (Azure OpenAI DALL-E)
type Request struct {
	// ID is the request identifier (all providers)
	ID string `json:"id"`
	// URL is an alternative webhook URL field (all providers)
	URL string `json:"webhook_url"`
	// Prompt is the text prompt for generation - REQUIRED (all providers)
	Prompt string `json:"prompt"`
	// NegativePrompt specifies what to avoid in generation (worker/SDXL only)
	NegativePrompt string `json:"negative_prompt,omitempty"`
	// Width of the output image in pixels (all providers)
	Width int `json:"width,omitempty"`
	// Height of the output image in pixels (all providers)
	Height int `json:"height,omitempty"`
	// NumSteps is the number of diffusion steps (worker/SDXL only)
	NumSteps int `json:"num_steps,omitempty"`
	// ClipSkip is the number of CLIP layers to skip (worker/SDXL only)
	ClipSkip int `json:"clip_skip,omitempty"`
	// GuidanceScale controls how closely to follow the prompt (worker/SDXL only)
	GuidanceScale float64 `json:"guidance_scale,omitempty"`
	// Seed for reproducible generation (worker/SDXL only)
	Seed int `json:"seed,omitempty"`
	// Quality level: "hd", "standard" (Azure: dall-e-3, gpt-image-1 only - not flux)
	Quality string `json:"quality,omitempty"`
	// BatchNbr is the number of images to generate (all providers)
	BatchNbr int `json:"batch_nbr,omitempty"`
	// NumFrames is the number of frames for video generation (worker/SDXL only)
	NumFrames int `json:"num_frames,omitempty"`
	// FPS is the frames per second for video output (worker/SDXL only)
	FPS int `json:"fps,omitempty"`
	// DecodeTimesteps for video decode process (worker/SDXL only)
	DecodeTimesteps float64 `json:"decode_timesteps,omitempty"`
	// DecodeNoiseScale for video decode process (worker/SDXL only)
	DecodeNoiseScale float64 `json:"decode_noise_scale,omitempty"`
	// ImgURL is the input image URL for img2img generation (worker/SDXL only)
	ImgURL string `json:"img_url,omitempty"`
	// Scheduler is the sampling scheduler to use (worker/SDXL only)
	Scheduler string `json:"scheduler,omitempty"`
	// ModelName is the model to use. Worker: SDXL model name. Azure: "gpt-image-1", "gpt-image-1.5", "flux.2-pro", "flux-1.1-pro" (required)
	ModelName string `json:"model_name,omitempty"`
	// OutputCompression controls image compression 0-100 (Azure: gpt-image-1, gpt-image-1.5 only)
	OutputCompression int `json:"output_compression,omitempty"`
	// OutputFormat specifies output format: "png", "jpeg" (Azure: gpt-image-1, gpt-image-1.5, flux-1.1-pro)
	OutputFormat string `json:"output_format,omitempty"`
	// AspectRatio for image generation (Google: "1:1", "2:3", "3:2", "3:4", "4:3", "4:5", "5:4", "9:16", "16:9", "21:9")
	AspectRatio string `json:"aspect_ratio,omitempty"`
	// ImageSize for output resolution (Google Pro model only: "1K", "2K", "4K")
	ImageSize string `json:"image_size,omitempty"`
	// Resolution for video generation (Google Veo: "720p", "1080p")
	Resolution string `json:"resolution,omitempty"`
	// DurationSeconds for video length (Google Veo: 4, 6, 8)
	DurationSeconds int `json:"duration_seconds,omitempty"`
	// PersonGeneration controls people generation (Google Veo: "allow_all", "allow_adult")
	PersonGeneration string `json:"person_generation,omitempty"`
}

type ResponsePost struct {
	Info        AddTaskInfoDoc
	UserPredict Predict
}

type GenericResponsePost struct {
	Info AddTaskInfoDoc  `json:"info"`
	Data json.RawMessage `json:"data" swaggertype:"object"`
}
type ResponseGet struct {
	Info GetTaskInfoDoc
}

type AddTaskInfoDoc struct {
	ID            string `json:"id"`
	Type          string `json:"type"`
	Payload       []byte `json:"payload"`
	State         string `json:"state"`
	Queue         string `json:"queue"`
	MaxRetry      int    `json:"max_retry"`
	Retried       int    `json:"retried"`
	LastErr       string
	LastFailedAt  time.Time
	Deadline      time.Time
	Group         string
	NextProcessAt time.Time
	IsOrphaned    bool
	CompletedAt   time.Time
	Result        []byte
}
type GetTaskInfoDoc struct {
	ID            string `json:"id"`
	Type          string `json:"type"`
	Payload       string `json:"payload"`
	State         string `json:"state"`
	Queue         string `json:"queue"`
	MaxRetry      int    `json:"max_retry"`
	Retried       int    `json:"retried"`
	LastErr       string
	LastFailedAt  time.Time
	Deadline      time.Time
	Group         string
	NextProcessAt time.Time
	IsOrphaned    bool
	CompletedAt   time.Time
	Result        string
}

// TypedPayload represents the validated input fields sent to workers.
// It is simply a map of field name → value (no wrapper field).
type TypedPayload map[string]interface{}
