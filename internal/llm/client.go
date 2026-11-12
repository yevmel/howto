// Package llm talks to an OpenAI-compatible chat completions endpoint.
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// DefaultTimeout limits each request to the endpoint.
const DefaultTimeout = 60 * time.Second

const maxErrorBody = 500

// Client sends requests to one endpoint.
type Client struct {
	baseURL string
	token   string
	timeout time.Duration
	http    *http.Client
}

// New returns a client for url, which may be a base URL or a full .../chat/completions URL.
func New(url, token string) *Client {
	return &Client{baseURL: url, token: token, timeout: DefaultTimeout, http: http.DefaultClient}
}

// ChatCompletionsURL derives the chat completions route from the configured URL.
func ChatCompletionsURL(url string) string {
	url = strings.TrimRight(url, "/")
	if strings.HasSuffix(url, "/chat/completions") {
		return url
	}
	return url + "/chat/completions"
}

// ModelsURL derives the models route from the configured URL.
func ModelsURL(url string) string {
	url = strings.TrimSuffix(strings.TrimRight(url, "/"), "/chat/completions")
	return url + "/models"
}

// DetectModel returns the first model the endpoint reports.
func (c *Client) DetectModel(ctx context.Context) (string, error) {
	var resp struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	err := c.do(ctx, ModelsURL(c.baseURL), nil, &resp)
	if err == nil && (len(resp.Data) == 0 || resp.Data[0].ID == "") {
		err = errors.New("no models listed")
	}
	if err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return "", fmt.Errorf("no model given and none could be detected; use --model or set HOWTO_LLM_MODEL (%w)", err)
	}
	return resp.Data[0].ID, nil
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Complete sends a system and a user message and returns the model's answer.
func (c *Client) Complete(ctx context.Context, model, system, user string) (string, error) {
	req := struct {
		Model    string    `json:"model"`
		Messages []message `json:"messages"`
	}{
		Model:    model,
		Messages: []message{{Role: "system", Content: system}, {Role: "user", Content: user}},
	}
	var resp struct {
		Choices []struct {
			Message *struct {
				Content *string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := c.do(ctx, ChatCompletionsURL(c.baseURL), req, &resp); err != nil {
		return "", err
	}
	if len(resp.Choices) == 0 || resp.Choices[0].Message == nil {
		return "", errors.New("unexpected response format from the endpoint")
	}
	if content := resp.Choices[0].Message.Content; content != nil {
		return *content, nil
	}
	return "", nil
}

// do sends a GET (payload nil) or POST request and decodes the JSON response into out.
func (c *Client) do(ctx context.Context, url string, payload, out any) error {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	method, body := http.MethodGet, io.Reader(nil)
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		method, body = http.MethodPost, bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return fmt.Errorf("invalid endpoint URL %q: %w", url, err)
	}
	req.Header.Set("Accept", "application/json")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return c.requestError(ctx, url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBody))
		return fmt.Errorf("%s returned HTTP %d: %s", url, resp.StatusCode, strings.TrimSpace(strings.ToValidUTF8(string(snippet), "�")))
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		if ctx.Err() != nil {
			return c.requestError(ctx, url, err)
		}
		return fmt.Errorf("%s did not return valid JSON", url)
	}
	return nil
}

func (c *Client) requestError(ctx context.Context, url string, err error) error {
	switch {
	case errors.Is(context.Cause(ctx), context.Canceled):
		return context.Canceled
	case errors.Is(ctx.Err(), context.DeadlineExceeded):
		return fmt.Errorf("request to %s timed out after %s", url, c.timeout)
	default:
		return fmt.Errorf("cannot reach %s: %w", url, unwrapURLError(err))
	}
}

// unwrapURLError drops the "Get \"url\":" prefix that the message already contains.
func unwrapURLError(err error) error {
	if inner := errors.Unwrap(err); inner != nil {
		return inner
	}
	return err
}
