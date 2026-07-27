// Package openai provides request translation functionality for OpenAI-compatible providers.
package chat_completions

import (
	"fmt"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// ConvertOpenAIRequestToOpenAI converts an OpenAI Chat Completions request (raw JSON)
// into an OpenAI-compatible request JSON. All JSON construction uses sjson and lookups use gjson.
//
// Parameters:
//   - modelName: The name of the model to use for the request
//   - rawJSON: The raw JSON request data from the OpenAI API
//   - stream: A boolean indicating if the request is for a streaming response (unused in current implementation)
//
// Returns:
//   - []byte: The transformed request data in OpenAI-compatible format
func ConvertOpenAIRequestToOpenAI(modelName string, inputRawJSON []byte, _ bool) []byte {
	updatedJSON, err := sjson.SetBytes(inputRawJSON, "model", modelName)
	if err != nil {
		return inputRawJSON
	}

	updatedJSON = normalizeMessages(updatedJSON)

	return updatedJSON
}

// normalizeMessages fixes common problems that strict upstreams (e.g. DeepSeek,
// Crucible via OpenRouter) reject in multi-turn conversation history:
//
//  1. content is null or missing on ANY message role — happens when a streaming
//     tool call is cancelled mid-flight; the client stores the partial assistant
//     message with content: null. Providers require content to be a string or array.
//
//  2. reasoning_content is absent on assistant messages — reasoning-aware models
//     (e.g. DeepSeek R1) require the field to be present in every assistant message
//     for multi-turn conversations; clients typically drop it when replaying history.
func normalizeMessages(body []byte) []byte {
	msgs := gjson.GetBytes(body, "messages")
	if !msgs.Exists() || !msgs.IsArray() {
		return body
	}

	result := body
	msgs.ForEach(func(key, value gjson.Result) bool {
		idx := key.Int()
		role := value.Get("role").String()

		// Fix 1: null or missing content → "" for ALL message roles.
		// Any role can end up with null content after a cancelled stream.
		content := value.Get("content")
		if !content.Exists() || content.Type == gjson.Null {
			path := fmt.Sprintf("messages.%d.content", idx)
			if next, setErr := sjson.SetBytes(result, path, ""); setErr == nil {
				result = next
			}
		}

		// Fix 2: missing reasoning_content → "" for assistant messages only.
		if role == "assistant" && !value.Get("reasoning_content").Exists() {
			path := fmt.Sprintf("messages.%d.reasoning_content", idx)
			if next, setErr := sjson.SetBytes(result, path, ""); setErr == nil {
				result = next
			}
		}

		return true
	})

	return result
}
