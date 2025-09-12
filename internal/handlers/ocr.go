package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	vision "cloud.google.com/go/vision/apiv1"
	"google.golang.org/api/option"
)

type OCRRequest struct {
	ImageURL string `json:"imageUrl"`
}

type OCRResponse struct {
	Text string `json:"text"`
}

func OCRHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	var req OCRRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	client, err := vision.NewImageAnnotatorClient(ctx, option.WithCredentialsFile("internal/config/service-account.json"))
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create Vision client: %v", err), http.StatusInternalServerError)
		return
	}
	defer client.Close()

	image := vision.NewImageFromURI(req.ImageURL)
	annotations, err := client.DetectTexts(ctx, image, nil, 1)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to detect text: %v", err), http.StatusInternalServerError)
		return
	}

	var text string
	if len(annotations) > 0 {
		text = annotations[0].Description
	}

	resp := OCRResponse{Text: text}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
