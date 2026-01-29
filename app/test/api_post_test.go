package test

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
)

func TestPost(t *testing.T) {
	// Skip if server is not running (this is an integration test)
	url := "http://localhost:3000/queue/add"

	// Quick connectivity check
	_, err := http.Get("http://localhost:3000/health")
	if err != nil {
		t.Skip("Skipping integration test: server not running at localhost:3000")
	}

	apiKey := os.Getenv("RUNQY_API_KEY")

	payload := strings.NewReader(`{"uuid": 1234,"predict_id":1,"priority": 0,"model_input":{"prompt":"Majestic blue whale breaching ocean waves under a vibrant sunset sky","negative_prompt": "score_6, score_5, score_4, simplified, abstract, unrealistic, impressionistic, low resolution, lowres, bad anatomy, bad hands, missing fingers, worst quality, low quality, normal quality, cartoon, anime, drawing, sketch, illustration, artificial, poor quality","width": 1024,"height": 1024,"steps": 30,"clip_skip": 2,"guidance_scale": 6,"batch_nbr": 1,"model_name": "cyberrealisticpony"}, "webhook_url": "http://test.com/webhook/runpod"}`)

	req, _ := http.NewRequest("POST", url, payload)

	req.Header.Add("Accept", "*/*")
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+apiKey)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("error reading body: %v", err)
	}

	fmt.Printf("Status: %d\n", res.StatusCode)
	fmt.Printf("Body: %s\n", string(body))
}
