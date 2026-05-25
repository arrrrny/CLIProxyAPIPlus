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

	// Ensure all assistant messages carry reasoning_content. Providers that don't
	// use it simply ignore the field, while reasoning models (e.g. DeepSeek R1)
	// require it to be present in every assistant message for multi-turn
	// conversations — clients typically strip it when replaying history.
	updatedJSON = ensureReasoningContent(updatedJSON)

	return updatedJSON
}

// ensureReasoningContent adds reasoning_content: "" to any assistant message
// that is missing the field, so that reasoning-aware upstreams (e.g. DeepSeek)
// do not reject multi-turn requests where a prior turn produced reasoning output.
func ensureReasoningContent(body []byte) []byte {
	msgs := gjson.GetBytes(body, "messages")
	if !msgs.Exists() || !msgs.IsArray() {
		return body
	}

	result := body
	msgs.ForEach(func(key, value gjson.Result) bool {
		if value.Get("role").String() != "assistant" {
			return true
		}
		if !value.Get("reasoning_content").Exists() {
			path := fmt.Sprintf("messages.%d.reasoning_content", key.Int())
			if next, setErr := sjson.SetBytes(result, path, ""); setErr == nil {
				result = next
			}
		}
		return true
	})

	return result
}
