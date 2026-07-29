package worker

import (
	"encoding/json"
	"testing"
)

// TestEnvVarsFromPayload proves the worker recovers a function's configured
// env vars from a job payload after the JSON round-trip the queue performs, so
// they can be handed to the runtime and set on the container.
func TestEnvVarsFromPayload(t *testing.T) {
	// Mirror how functions.Service enqueues: a map[string]string under "envVars".
	original := map[string]interface{}{
		"envVars": map[string]string{"MY_VAR": "hello", "API_KEY": "secret"},
	}
	wire, _ := json.Marshal(original)
	var payload map[string]interface{}
	if err := json.Unmarshal(wire, &payload); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	env := envVarsFromPayload(payload["envVars"])
	if env["MY_VAR"] != "hello" || env["API_KEY"] != "secret" {
		t.Fatalf("expected env vars recovered, got %v", env)
	}
}

// TestEnvVarsFromPayloadEmpty confirms a missing or empty map yields nil rather
// than injecting anything.
func TestEnvVarsFromPayloadEmpty(t *testing.T) {
	if got := envVarsFromPayload(nil); got != nil {
		t.Errorf("nil payload should give nil, got %v", got)
	}
	if got := envVarsFromPayload(map[string]interface{}{}); got != nil {
		t.Errorf("empty map should give nil, got %v", got)
	}
	// Non-string values are skipped, never coerced into bogus env.
	got := envVarsFromPayload(map[string]interface{}{"N": 42, "OK": "yes"})
	if _, present := got["N"]; present {
		t.Errorf("non-string value should be skipped, got %v", got)
	}
	if got["OK"] != "yes" {
		t.Errorf("string value should survive, got %v", got)
	}
}
