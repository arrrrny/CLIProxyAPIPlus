package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/router-for-me/CLIProxyAPI/v7/internal/config"
	cliproxyauth "github.com/router-for-me/CLIProxyAPI/v7/sdk/cliproxy/auth"
	cliproxyexecutor "github.com/router-for-me/CLIProxyAPI/v7/sdk/cliproxy/executor"
	sdktranslator "github.com/router-for-me/CLIProxyAPI/v7/sdk/translator"
)

func TestDirectAntigravityModelsMultiAccount(t *testing.T) {
	accounts := []string{
		"../../../antigravity-support@zuzu.dev.json",
		"../../../antigravity-support@amzsync.com.json",
		"../../../antigravity-zikzakzikzakwtf@gmail.com.json",
	}

	models := []string{
		"gemini-2.5-flash-lite",
		"gemini-2.5-pro",
		"gemini-2.5-flash",
		"gemini-3-flash-agent",
	}

	exec := NewAntigravityExecutor(&config.Config{})

	for _, accPath := range accounts {
		authData, err := os.ReadFile(accPath)
		if err != nil {
			fmt.Printf("Skipping %s: %v\n", accPath, err)
			continue
		}

		var auth cliproxyauth.Auth
		json.Unmarshal(authData, &auth)

		// Map metadata for executor compatibility
		var raw map[string]any
		json.Unmarshal(authData, &raw)
		if auth.Metadata == nil {
			auth.Metadata = make(map[string]any)
		}
		for k, v := range raw {
			auth.Metadata[k] = v
		}

		email, _ := auth.Metadata["email"].(string)
		fmt.Printf("\n--- Testing Account: %s ---\n", email)

		for _, model := range models {
			req := cliproxyexecutor.Request{
				Model:   model,
				Payload: []byte(`{"contents": [{"role": "user", "parts": [{"text": "hi"}]}]}`),
			}
			opts := cliproxyexecutor.Options{
				SourceFormat: sdktranslator.FromString("gemini"),
			}

			resp, err := exec.Execute(context.Background(), &auth, req, opts)
			if err != nil {
				fmt.Printf("Model %s: FAIL - %v\n", model, err)
			} else {
				fmt.Printf("Model %s: SUCCESS! Response length: %d\n", model, len(resp.Payload))
			}
		}
	}
}
