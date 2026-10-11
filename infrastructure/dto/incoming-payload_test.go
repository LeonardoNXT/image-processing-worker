package dto

import "testing"

func TestDecodeIncomingPayloadSchemaFailureKeepsRouting(t *testing.T) {
	payload := []byte(`{
        "eventId":"82c8e4ba-d96d-4076-9a73-109c908fd4bc",
        "imageId":"9a071332-a599-4559-b6b9-86d59e12d3e4",
        "ownerId":"owner",
        "service":"user",
        "objectKey":"user/owner/image",
        "outputTopic":"user.processedImage",
        "maxSizeBytes":"wrong-type"
    }`)
	request, failure, err := DecodeIncomingPayload(payload)
	if err != nil {
		t.Fatal(err)
	}
	if failure == nil || failure.Code != "INVALID_REQUEST" {
		t.Fatal("schema failure was not returned")
	}
	if request.EventID == "" || request.ImageID == "" || request.OutputTopic == "" {
		t.Fatal("routing envelope was lost")
	}
}

func TestDecodeIncomingPayloadInvalidSyntax(t *testing.T) {
	_, failure, err := DecodeIncomingPayload([]byte(`{"eventId":`))
	if err == nil || failure != nil {
		t.Fatal("invalid syntax must not produce a routable failure")
	}
}
