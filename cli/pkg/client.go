package pkg

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

// APIClient defines the interface for backend interactions
type APIClient interface {
	// Base HTTP Methods
	Get(path string, result interface{}) error
	Post(path string, body interface{}, result interface{}) error
	Put(path string, body interface{}, result interface{}) error
	Delete(path string) error
	Download(path string, out io.Writer) error

	// Core Logging
	StreamCoreLogs(ctx context.Context, coreName string, tail int, follow bool, out io.Writer) error
}

// Client implements APIClient interacting with the backend API
type Client struct {
	httpClient *http.Client
	baseURL    string
	token      string
}

// DefaultClient is the package-level client instance, which can be overridden in tests
var DefaultClient APIClient = nil

// GetClient returns the default initialized client based on current profile
func GetClient() (APIClient, error) {
	if DefaultClient != nil {
		return DefaultClient, nil
	}

	config, err := LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	profile, err := config.GetCurrentProfile()
	if err != nil {
		return nil, fmt.Errorf("no profile selected. Use 'isolate-panel login' first")
	}

	client := &Client{
		httpClient: &http.Client{Timeout: 300 * time.Second}, // Increased for backups
		baseURL:    strings.TrimSuffix(profile.PanelURL, "/"),
		token:      profile.AccessToken,
	}
	
	DefaultClient = client
	return DefaultClient, nil
}

func (c *Client) doReqReader(method, path string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, c.baseURL+path, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer resp.Body.Close()
		var errorResp struct {
			Error string `json:"error"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&errorResp); err == nil && errorResp.Error != "" {
			return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, errorResp.Error)
		}
		return nil, fmt.Errorf("API error: HTTP %d", resp.StatusCode)
	}

	return resp, nil
}

func (c *Client) doReq(method, path string, body interface{}, result interface{}) error {
	var bodyReader io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonData)
	}

	resp, err := c.doReqReader(method, path, bodyReader)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

func (c *Client) Download(path string, out io.Writer) error {
	resp, err := c.doReqReader(http.MethodGet, path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func (c *Client) Get(path string, result interface{}) error {
	return c.doReq(http.MethodGet, path, nil, result)
}

func (c *Client) Post(path string, body interface{}, result interface{}) error {
	return c.doReq(http.MethodPost, path, body, result)
}

func (c *Client) Put(path string, body interface{}, result interface{}) error {
	return c.doReq(http.MethodPut, path, body, result)
}

func (c *Client) Delete(path string) error {
	return c.doReq(http.MethodDelete, path, nil, nil)
}

func (c *Client) StreamCoreLogs(ctx context.Context, coreName string, tail int, follow bool, out io.Writer) error {
	// Parse URL
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return fmt.Errorf("invalid base url: %w", err)
	}

	// Determine WebSocket Scheme
	scheme := "ws"
	if u.Scheme == "https" {
		scheme = "wss"
	}

	wsPath := fmt.Sprintf("/api/cores/%s/logs", coreName)
	wsURL := fmt.Sprintf("%s://%s%s?lines=%d", scheme, u.Host, wsPath, tail)
	if follow {
		wsURL += "&follow=true"
	}

	headers := http.Header{}
	headers.Add("Authorization", "Bearer "+c.token) // Some websocket setups handle Authorization Header or expect token via query param?
	// Note: backend's CoreLogsHandler expects `token` query param or Authorization header or cookie. We will use query param as it's safe for JS websockets too.
	wsURL += fmt.Sprintf("&token=%s", url.QueryEscape(c.token))

	conn, resp, err := websocket.DefaultDialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		if resp != nil {
             return fmt.Errorf("websocket dial failed (status %d): %w", resp.StatusCode, err)
		}
		return fmt.Errorf("websocket dial failed: %w", err)
	}
	defer conn.Close()

	for {
		select {
		case <-ctx.Done():
			// send close frame gracefully
			conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			return nil
		default:
			_, message, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					return fmt.Errorf("websocket error: %w", err)
				}
				// Normal close
				return nil
			}
			fmt.Fprintln(out, string(message))
			if !follow {
				// We expect the server to close connection when follow=false after sending tail lines, 
				// but just in case, break loop on first batch read if the message is parsed somehow.
				// However backend actually streams logs chunk by chunk. If follow is false, the backend itself shouldn't keep the socket open.
				// For now we rely on backend closing.
			}
		}
	}
}
