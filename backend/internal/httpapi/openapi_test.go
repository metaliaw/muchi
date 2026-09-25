package httpapi

import (
	"gopkg.in/yaml.v3"
	"os"
	"testing"
)

func TestValidateOpenAPI(t *testing.T) {
	data, err := os.ReadFile("../../openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := yaml.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	if document["openapi"] != "3.1.0" {
		t.Fatalf("OpenAPI version = %v", document["openapi"])
	}
}
