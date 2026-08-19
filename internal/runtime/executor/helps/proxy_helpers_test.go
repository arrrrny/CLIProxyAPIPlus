package helps

import (
	"context"
	"net/http"
	"testing"

	"github.com/router-for-me/CLIProxyAPI/v7/internal/config"
	cliproxyauth "github.com/router-for-me/CLIProxyAPI/v7/sdk/cliproxy/auth"
	sdkconfig "github.com/router-for-me/CLIProxyAPI/v7/sdk/config"
)

func TestNewProxyAwareHTTPClientDirectBypassesGlobalProxy(t *testing.T) {
	t.Parallel()

	client := NewProxyAwareHTTPClient(
		context.Background(),
		&config.Config{SDKConfig: sdkconfig.SDKConfig{ProxyURL: "http://global-proxy.example.com:8080"}},
		&cliproxyauth.Auth{ProxyURL: "direct"},
		0,
	)

	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("transport type = %T, want *http.Transport", client.Transport)
	}
	if transport.Proxy != nil {
		t.Fatal("expected direct transport to disable proxy function")
	}
}

func TestNewProxyAwareHTTPClientProxyDisabledByDefault(t *testing.T) {
	t.Parallel()

	// With ProxyEnabledByDefault=false (default), providers without explicit proxy URL
	// should not use the global proxy, even if one is configured
	client := NewProxyAwareHTTPClient(
		context.Background(),
		&config.Config{SDKConfig: sdkconfig.SDKConfig{ProxyURL: "http://global-proxy.example.com:8080", ProxyEnabledByDefault: false}},
		&cliproxyauth.Auth{ProxyURL: ""}, // No explicit proxy
		0,
	)

	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("transport type = %T, want *http.Transport", client.Transport)
	}
	// Should use direct transport (no proxy function) when proxy is disabled by default
	if transport.Proxy != nil {
		t.Fatal("expected direct transport when proxy is disabled by default and auth has no proxy")
	}
}

func TestNewProxyAwareHTTPClientProxyEnabledByDefault(t *testing.T) {
	t.Parallel()

	// With ProxyEnabledByDefault=true, providers without explicit proxy URL
	// should use the global proxy
	client := NewProxyAwareHTTPClient(
		context.Background(),
		&config.Config{SDKConfig: sdkconfig.SDKConfig{ProxyURL: "http://global-proxy.example.com:8080", ProxyEnabledByDefault: true}},
		&cliproxyauth.Auth{ProxyURL: ""}, // No explicit proxy
		0,
	)

	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("transport type = %T, want *http.Transport", client.Transport)
	}
	// Should use the global proxy when proxy is enabled by default
	if transport.Proxy == nil {
		t.Fatal("expected proxy transport when proxy is enabled by default")
	}

	req, _ := http.NewRequest(http.MethodGet, "https://example.com", nil)
	proxyURL, err := transport.Proxy(req)
	if err != nil {
		t.Fatalf("transport.Proxy() error = %v", err)
	}
	if proxyURL == nil || proxyURL.String() != "http://global-proxy.example.com:8080" {
		t.Fatalf("proxy URL = %v, want http://global-proxy.example.com:8080", proxyURL)
	}
}

func TestNewProxyAwareHTTPClientExplicitAuthProxyTakesPrecedence(t *testing.T) {
	t.Parallel()

	// Even when ProxyEnabledByDefault=false, explicit auth proxy should be used
	client := NewProxyAwareHTTPClient(
		context.Background(),
		&config.Config{SDKConfig: sdkconfig.SDKConfig{ProxyURL: "http://global-proxy.example.com:8080", ProxyEnabledByDefault: false}},
		&cliproxyauth.Auth{ProxyURL: "http://auth-proxy.example.com:8080"},
		0,
	)

	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("transport type = %T, want *http.Transport", client.Transport)
	}
	if transport.Proxy == nil {
		t.Fatal("expected proxy transport when auth has explicit proxy URL")
	}

	req, _ := http.NewRequest(http.MethodGet, "https://example.com", nil)
	proxyURL, err := transport.Proxy(req)
	if err != nil {
		t.Fatalf("transport.Proxy() error = %v", err)
	}
	if proxyURL == nil || proxyURL.String() != "http://auth-proxy.example.com:8080" {
		t.Fatalf("proxy URL = %v, want http://auth-proxy.example.com:8080", proxyURL)
	}
}
