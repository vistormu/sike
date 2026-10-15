// Package ntfy publishes to an ntfy server (the homelab's): json to the base
// url, the topic in the body, so titles may hold emoji; a bearer token, as the
// server denies everything else.
package ntfy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"sike/domain/port"
)

type Client struct {
	url, token string
	http       *http.Client
}

// New is a client for the server at url (https://ntfy.example.com),
// publishing with token (tk_…; "" for an open server).
func New(url, token string) *Client {
	return &Client{url: strings.TrimRight(url, "/"), token: token, http: &http.Client{Timeout: 10 * time.Second}}
}

func (c *Client) Notify(ctx context.Context, topic string, m port.Message) error {
	body, err := json.Marshal(map[string]string{"topic": topic, "title": m.Title, "message": m.Body})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	res, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("ntfy: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		// ntfy says why as json ({"code":40301,"error":"forbidden"})
		var why struct{ Error string }
		msg, _ := io.ReadAll(io.LimitReader(res.Body, 512))
		if json.Unmarshal(msg, &why) != nil || why.Error == "" {
			why.Error = strings.TrimSpace(string(msg))
		}
		return fmt.Errorf("ntfy: %d %s", res.StatusCode, why.Error)
	}
	return nil
}
