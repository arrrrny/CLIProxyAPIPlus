package claude

import (
	"bytes"
	"context"
	"testing"
)

type sseEvent struct {
	Type    string
	Payload string
}

func runStream(t *testing.T, originalReq string, chunks ...string) []sseEvent {
	t.Helper()

	var paramAny any
	var emitted [][]byte
	for _, chunk := range chunks {
		emitted = append(emitted, ConvertOpenAIResponseToClaude(
			context.Background(),
			"",
			[]byte(originalReq),
			nil,
			[]byte("data: "+chunk),
			&paramAny,
		)...)
	}
	emitted = append(emitted, ConvertOpenAIResponseToClaude(
		context.Background(),
		"",
		[]byte(originalReq),
		nil,
		[]byte("data: [DONE]"),
		&paramAny,
	)...)

	var events []sseEvent
	for _, raw := range emitted {
		s := string(raw)
		if !strings.HasPrefix(s, "event: ") {
			continue
		}
		nl := strings.Index(s, "\n")
		if nl < 0 {
			continue
		}
		typ := strings.TrimPrefix(s[:nl], "event: ")
		rest := s[nl+1:]
		if !strings.HasPrefix(rest, "data: ") {
			continue
		}
		payload := strings.TrimRight(strings.TrimPrefix(rest, "data: "), "\n")
		events = append(events, sseEvent{Type: typ, Payload: payload})
	}
	return events
}

func countByType(events []sseEvent, typ string) int {
	n := 0
	for _, e := range events {
		if e.Type == typ {
			n++
		}
	}
	return n
}

func toolUseStarts(events []sseEvent) []sseEvent {
	var out []sseEvent
	for _, e := range events {
		if e.Type != "content_block_start" {
			continue
		}
		if gjson.Get(e.Payload, "content_block.type").String() == "tool_use" {
			out = append(out, e)
		}
	}
	return out
}

func blockIndices(events []sseEvent) []int64 {
	var idx []int64
	for _, e := range events {
		if e.Type == "content_block_start" {
			idx = append(idx, gjson.Get(e.Payload, "index").Int())
		}
	}
	return idx
}

func lastStopReason(events []sseEvent) string {
	for i := len(events) - 1; i >= 0; i-- {
		if events[i].Type == "message_delta" {
			return gjson.Get(events[i].Payload, "delta.stop_reason").String()
		}
	}
	return ""
}

const streamReq = `{"stream":true}`

func TestStreaming_LateUsageOnlyDoesNotEmitAfterMessageStop(t *testing.T) {
	events := runStream(t, streamReq,
		`{"id":"c1","model":"m","choices":[{"index":0,"delta":{"role":"assistant"},"finish_reason":null}]}`,
		`{"id":"c1","model":"m","choices":[{"index":0,"delta":{"content":"hello"},"finish_reason":null}]}`,
		`{"id":"c1","model":"m","choices":[{"index":0,"delta":{},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":1}}`,
		`{"id":"c1","model":"m","choices":[],"usage":{"prompt_tokens":1,"completion_tokens":1}}`,
	)

	if got := countByType(events, "message_delta"); got != 1 {
		t.Fatalf("expected exactly one message_delta, got %d (events=%+v)", got, events)
	}
	if got := countByType(events, "message_stop"); got != 1 {
		t.Fatalf("expected exactly one message_stop, got %d (events=%+v)", got, events)
	}
	if len(events) == 0 || events[len(events)-1].Type != "message_stop" {
		t.Fatalf("message_stop must be the last semantic event (events=%+v)", events)
	}
}

func TestConvertOpenAIResponseToClaude_StreamIgnoresNullToolNameDelta(t *testing.T) {
	originalRequest := []byte(`{"stream":true}`)
	var param any

	firstChunks := ConvertOpenAIResponseToClaude(
		context.Background(),
		"test-model",
		originalRequest,
		nil,
		[]byte(`data: {"id":"chatcmpl_1","model":"test-model","created":1,"choices":[{"index":0,"delta":{"role":"assistant","tool_calls":[{"index":0,"id":"call_1","type":"function","function":{"name":"read_file","arguments":""}}]},"finish_reason":null}]}`),
		&param,
	)
	firstOutput := bytes.Join(firstChunks, nil)
	if !bytes.Contains(firstOutput, []byte(`"name":"read_file"`)) {
		t.Fatalf("expected first chunk to start read_file tool block, got %s", string(firstOutput))
	}

	secondChunks := ConvertOpenAIResponseToClaude(
		context.Background(),
		"test-model",
		originalRequest,
		nil,
		[]byte(`data: {"id":"chatcmpl_1","model":"test-model","created":1,"choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"name":null,"arguments":"{\"path\":\"/tmp/a\"}"}}]},"finish_reason":null}]}`),
		&param,
	)
	secondOutput := bytes.Join(secondChunks, nil)
	if bytes.Contains(secondOutput, []byte(`content_block_start`)) {
		t.Fatalf("did not expect null tool name delta to start a new content block, got %s", string(secondOutput))
	}
	if bytes.Contains(secondOutput, []byte(`"name":""`)) {
		t.Fatalf("did not expect null tool name delta to emit an empty tool name, got %s", string(secondOutput))
	}
}
