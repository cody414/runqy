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
// Request contains the model input parameters for image/video generation.
// Fields are generic: send any fields you have, unused ones are ignored.
type Request struct {
	// ID is the request identifier
	ID string `json:"id"`
	// URL is an alternative webhook URL field
	URL string `json:"webhook_url"`
	// Prompt is the text prompt for generation - REQUIRED
	Prompt string `json:"prompt"`
	// NegativePrompt specifies what to avoid in generation
	NegativePrompt string `json:"negative_prompt,omitempty"`
	// Width of the output image in pixels
	Width int `json:"width,omitempty"`
	// Height of the output image in pixels
	Height int `json:"height,omitempty"`
	// NumSteps is the number of diffusion steps
	NumSteps int `json:"num_steps,omitempty"`
	// ClipSkip is the number of CLIP layers to skip
	ClipSkip int `json:"clip_skip,omitempty"`
	// GuidanceScale controls how closely to follow the prompt
	GuidanceScale float64 `json:"guidance_scale,omitempty"`
	// Seed for reproducible generation
	Seed int `json:"seed,omitempty"`
	// Quality level: "hd", "standard"
	Quality string `json:"quality,omitempty"`
	// BatchNbr is the number of images to generate
	BatchNbr int `json:"batch_nbr,omitempty"`
	// NumFrames is the number of frames for video generation
	NumFrames int `json:"num_frames,omitempty"`
	// FPS is the frames per second for video output
	FPS int `json:"fps,omitempty"`
	// DecodeTimesteps for video decode process
	DecodeTimesteps float64 `json:"decode_timesteps,omitempty"`
	// DecodeNoiseScale for video decode process
	DecodeNoiseScale float64 `json:"decode_noise_scale,omitempty"`
	// ImgURL is the input image URL for img2img generation
	ImgURL string `json:"img_url,omitempty"`
	// Scheduler is the sampling scheduler to use
	Scheduler string `json:"scheduler,omitempty"`
	// ModelName is the model to use
	ModelName string `json:"model_name,omitempty"`
	// OutputCompression controls image compression 0-100
	OutputCompression int `json:"output_compression,omitempty"`
	// OutputFormat specifies output format: "png", "jpeg"
	OutputFormat string `json:"output_format,omitempty"`
	// AspectRatio for image generation
	AspectRatio string `json:"aspect_ratio,omitempty"`
	// ImageSize for output resolution
	ImageSize string `json:"image_size,omitempty"`
	// Resolution for video generation
	Resolution string `json:"resolution,omitempty"`
	// DurationSeconds for video length
	DurationSeconds int `json:"duration_seconds,omitempty"`
	// PersonGeneration controls people generation
	PersonGeneration string `json:"person_generation,omitempty"`
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
