package runtime

import "testing"

// TestFuncEnvIncludesConfiguredVars proves a function's configured envVars are
// placed on the container environment as KEY=VALUE (Docker Config.Env), which is
// what makes them readable inside the runtime via process.env / os.environ.
func TestFuncEnvIncludesConfiguredVars(t *testing.T) {
	req := ExecRequest{
		FunctionID: "fn123",
		ProjectID:  "proj456",
		EnvVars: map[string]string{
			"MY_VAR":  "hello",
			"API_KEY": "secret",
			"EMPTY":   "",
			"WITH_EQ": "a=b",
		},
	}

	env := funcEnv(req)

	got := map[string]bool{}
	for _, e := range env {
		got[e] = true
	}

	want := []string{
		"MY_VAR=hello",
		"API_KEY=secret",
		"EMPTY=",
		"WITH_EQ=a=b",
		"APPLAD_FUNCTION_ID=fn123",
		"APPLAD_PROJECT_ID=proj456",
	}
	for _, w := range want {
		if !got[w] {
			t.Errorf("expected container env to carry %q, got %v", w, env)
		}
	}
}

// TestFuncEnvOnlyFunctionVars proves nothing beyond the function's own vars and
// the two Applad identifiers is injected, so unrelated config cannot leak in.
func TestFuncEnvOnlyFunctionVars(t *testing.T) {
	req := ExecRequest{
		FunctionID: "fn1",
		ProjectID:  "p1",
		EnvVars:    map[string]string{"ONLY": "one"},
	}
	env := funcEnv(req)
	if len(env) != 3 {
		t.Fatalf("expected exactly 3 env entries (1 configured + 2 applad), got %d: %v", len(env), env)
	}
}
