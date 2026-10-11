package domain

import (
	"encoding/json"
	"image-processing-worker/domain/patterns"
	"testing"
)

func TestNullableDefaultsAndAliases(t *testing.T) {
	var shape OutputShape
	if err := json.Unmarshal([]byte(`{"name":"avatar","maxWidth":500,"maxHeight":500,"format":null,"quality":null}`), &shape); err != nil {
		t.Fatal(err)
	}
	if shape.Encoding() != "webp" || shape.EncodingQuality() != 80 {
		t.Fatal("invalid defaults")
	}
	f := "JPG"
	shape.Format = &f
	if shape.Encoding() != "jpeg" {
		t.Fatal("JPG alias not normalized")
	}
	r := Request{PermittedFormats: []string{"jpg"}}
	if !r.Allows("jpeg") || r.Allows("png") || r.Allows("pdf") {
		t.Fatal("incorrect format intersection")
	}
}
func TestRejectInvalidOutputs(t *testing.T) {
	for _, name := range []string{"duplicate", "path", "gif", "oversize", "quality"} {
		t.Run(name, func(t *testing.T) {
			r := Request{SchemaVersion: 1, MaxSizeBytes: 1, PermittedFormats: []string{"png"}, OutputShape: []OutputShape{{Name: "avatar", MaxWidth: 500, MaxHeight: 500}}}
			switch name {
			case "duplicate":
				r.OutputShape = append(r.OutputShape, r.OutputShape[0])
			case "path":
				r.OutputShape[0].Name = "../avatar"
			case "gif":
				f := "gif"
				r.OutputShape[0].Format = &f
			case "oversize":
				r.OutputShape[0].MaxWidth = 1000000
			case "quality":
				q := 0
				r.OutputShape[0].Quality = &q
			}
			if r.Validate(patterns.Config()) == nil {
				t.Fatal("expected rejection")
			}
		})
	}
}

func TestRoutingAllowsBusinessValidationFailure(t *testing.T) {
	r := Request{EventID: "82c8e4ba-d96d-4076-9a73-109c908fd4bc", ImageID: "9a071332-a599-4559-b6b9-86d59e12d3e4", OutputTopic: "user.processedImage"}
	if err := r.ValidateRouting(); err != nil {
		t.Fatal(err)
	}
	if r.Validate(patterns.Config()) == nil {
		t.Fatal("missing business fields must fail validation")
	}
}
