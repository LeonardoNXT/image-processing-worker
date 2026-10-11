package dto

import (
	"encoding/json"
	"fmt"
	"image-processing-worker/domain"
)

type IncomingPayload = domain.Request
type OutputShape = domain.OutputShape

func DecodeIncomingPayload(value []byte) (domain.Request, *domain.ProcessingError, error) {
	var request domain.Request
	if err := json.Unmarshal(value, &request); err == nil {
		return request, nil, nil
	}
	if !json.Valid(value) {
		return request, nil, fmt.Errorf("invalid JSON syntax")
	}
	var routing struct {
		EventID     string `json:"eventId"`
		ImageID     string `json:"imageId"`
		OwnerID     string `json:"ownerId"`
		ObjectKey   string `json:"objectKey"`
		Service     string `json:"service"`
		OutputTopic string `json:"outputTopic"`
	}
	if err := json.Unmarshal(value, &routing); err != nil {
		return request, nil, fmt.Errorf("invalid routing envelope: %w", err)
	}
	request.EventID = routing.EventID
	request.ImageID = routing.ImageID
	request.OwnerID = routing.OwnerID
	request.ObjectKey = routing.ObjectKey
	request.Service = routing.Service
	request.OutputTopic = routing.OutputTopic
	return request, domain.Reject("INVALID_REQUEST", "VALIDATION", "request JSON does not match schema version 1"), nil
}
