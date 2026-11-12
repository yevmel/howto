package llm

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestURLs(t *testing.T) {
	tests := []struct{ in, chat, models string }{
		{"http://h/v1", "http://h/v1/chat/completions", "http://h/v1/models"},
		{"http://h/v1/", "http://h/v1/chat/completions", "http://h/v1/models"},
		{"http://h/v1/chat/completions", "http://h/v1/chat/completions", "http://h/v1/models"},
		{"http://h/v1/chat/completions/", "http://h/v1/chat/completions", "http://h/v1/models"},
	}
	for _, tt := range tests {
		if got := ChatCompletionsURL(tt.in); got != tt.chat {
			t.Errorf("ChatCompletionsURL(%q) = %q, want %q", tt.in, got, tt.chat)
		}
		if got := ModelsURL(tt.in); got != tt.models {
			t.Errorf("ModelsURL(%q) = %q, want %q", tt.in, got, tt.models)
		}
	}
}

func server(t *testing.T, h http.HandlerFunc) *httptest.Server {
	t.Helper()
	s := httptest.NewServer(h)
	t.Cleanup(s.Close)
	return s
}

func TestComplete(t *testing.T) {
	s := server(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/chat/completions" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer secret" {
			t.Errorf("Authorization = %q", got)
		}
		var req struct {
			Model    string    `json:"model"`
			Messages []message `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatal(err)
		}
		if req.Model != "m" || len(req.Messages) != 2 || req.Messages[0].Role != "system" || req.Messages[1].Content != "list files" {
			t.Errorf("unexpected payload %+v", req)
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"ls -la"}}]}`))
	})

	got, err := New(s.URL+"/v1", "secret").Complete(context.Background(), "m", "sys", "list files")
	if err != nil || got != "ls -la" {
		t.Fatalf("got %q, %v", got, err)
	}
}

func TestNoTokenNoHeader(t *testing.T) {
	s := server(t, func(w http.ResponseWriter, r *http.Request) {
		if _, ok := r.Header["Authorization"]; ok {
			t.Error("Authorization header sent without a token")
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":null}}]}`))
	})
	got, err := New(s.URL, "").Complete(context.Background(), "m", "s", "u")
	if err != nil || got != "" {
		t.Fatalf("got %q, %v", got, err)
	}
}

func TestDetectModel(t *testing.T) {
	s := server(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1/models" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"data":[{"id":"first"},{"id":"second"}]}`))
	})
	got, err := New(s.URL+"/v1/chat/completions", "").DetectModel(context.Background())
	if err != nil || got != "first" {
		t.Fatalf("got %q, %v", got, err)
	}
}

func TestErrors(t *testing.T) {
	tests := []struct {
		name    string
		handler http.HandlerFunc
		detect  bool
		want    string
	}{
		{"http error", func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, strings.Repeat("x", 1000), http.StatusUnauthorized)
		}, false, "returned HTTP 401: " + strings.Repeat("x", maxErrorBody)},
		{"invalid json", func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("<html>"))
		}, false, "did not return valid JSON"},
		{"unexpected format", func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"choices":[]}`))
		}, false, "unexpected response format"},
		{"no models", func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"data":[]}`))
		}, true, "no model given and none could be detected"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New(server(t, tt.handler).URL, "")
			var err error
			if tt.detect {
				_, err = c.DetectModel(context.Background())
			} else {
				_, err = c.Complete(context.Background(), "m", "s", "u")
			}
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("got %v, want it to contain %q", err, tt.want)
			}
			if strings.Contains(err.Error(), strings.Repeat("x", maxErrorBody+1)) {
				t.Error("error body not truncated")
			}
		})
	}
}

func TestUnreachable(t *testing.T) {
	s := httptest.NewServer(http.NotFoundHandler())
	url := s.URL
	s.Close()
	_, err := New(url, "").Complete(context.Background(), "m", "s", "u")
	if err == nil || !strings.HasPrefix(err.Error(), "cannot reach "+url) {
		t.Fatalf("got %v", err)
	}
}

func TestTimeoutAndCancel(t *testing.T) {
	release := make(chan struct{})
	s := server(t, func(_ http.ResponseWriter, r *http.Request) {
		select {
		case <-release:
		case <-r.Context().Done():
		}
	})
	defer close(release)

	c := New(s.URL, "")
	c.timeout = 50 * time.Millisecond
	_, err := c.Complete(context.Background(), "m", "s", "u")
	if err == nil || !strings.Contains(err.Error(), "timed out after 50ms") {
		t.Errorf("timeout: got %v", err)
	}

	c.timeout = time.Minute
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(50*time.Millisecond, cancel)
	_, err = c.Complete(ctx, "m", "s", "u")
	if !errors.Is(err, context.Canceled) {
		t.Errorf("cancel: got %v, want context.Canceled", err)
	}
	_, err = c.DetectModel(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("cancel during detection: got %v, want context.Canceled", err)
	}
}
