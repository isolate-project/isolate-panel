package cores

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"time"
)

// SupervisorClient handles communication with supervisord via XML-RPC
type SupervisorClient struct {
	url    string
	client *http.Client
}

// ProcessInfo represents supervisord process information
type ProcessInfo struct {
	Name           string
	Group          string
	Description    string
	Start          int64
	Stop           int64
	Now            int64
	State          int
	StateName      string
	SpawnErr       string
	ExitStatus     int
	Stdout_logfile string
	Stderr_logfile string
	PID            int
}

// NewSupervisorClient creates a new supervisord client
func NewSupervisorClient(url string) *SupervisorClient {
	return &SupervisorClient{
		url: url,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// StartProcess starts a supervisord process
func (sc *SupervisorClient) StartProcess(name string) error {
	response, err := sc.call("supervisor.startProcess", name, true)
	if err != nil {
		return fmt.Errorf("failed to start process %s: %w", name, err)
	}

	var result bool
	if err := xml.Unmarshal(response, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if !result {
		return fmt.Errorf("failed to start process %s", name)
	}

	return nil
}

// StopProcess stops a supervisord process
func (sc *SupervisorClient) StopProcess(name string) error {
	response, err := sc.call("supervisor.stopProcess", name, true)
	if err != nil {
		return fmt.Errorf("failed to stop process %s: %w", name, err)
	}

	var result bool
	if err := xml.Unmarshal(response, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if !result {
		return fmt.Errorf("failed to stop process %s", name)
	}

	return nil
}

// RestartProcess restarts a supervisord process
func (sc *SupervisorClient) RestartProcess(name string) error {
	// Stop first
	if err := sc.StopProcess(name); err != nil {
		// Ignore error if process is not running
	}

	// Wait a bit
	time.Sleep(500 * time.Millisecond)

	// Start
	return sc.StartProcess(name)
}

// GetProcessInfo gets information about a process
func (sc *SupervisorClient) GetProcessInfo(name string) (*ProcessInfo, error) {
	response, err := sc.call("supervisor.getProcessInfo", name)
	if err != nil {
		return nil, fmt.Errorf("failed to get process info for %s: %w", name, err)
	}

	var info ProcessInfo
	if err := xml.Unmarshal(response, &info); err != nil {
		return nil, fmt.Errorf("failed to parse process info: %w", err)
	}

	return &info, nil
}

// IsProcessRunning checks if a process is running
func (sc *SupervisorClient) IsProcessRunning(name string) (bool, error) {
	info, err := sc.GetProcessInfo(name)
	if err != nil {
		return false, err
	}

	// State 20 = RUNNING
	return info.State == 20, nil
}

// call makes an XML-RPC call to supervisord
func (sc *SupervisorClient) call(method string, params ...interface{}) ([]byte, error) {
	// Build XML-RPC request
	var buf bytes.Buffer
	buf.WriteString(`<?xml version="1.0"?>`)
	buf.WriteString(`<methodCall>`)
	buf.WriteString(`<methodName>` + method + `</methodName>`)
	buf.WriteString(`<params>`)

	for _, param := range params {
		buf.WriteString(`<param>`)
		switch v := param.(type) {
		case string:
			var escaped bytes.Buffer
			_ = xml.EscapeText(&escaped, []byte(v))
			buf.WriteString(`<value><string>` + escaped.String() + `</string></value>`)
		case int:
			buf.WriteString(fmt.Sprintf(`<value><int>%d</int></value>`, v))
		case bool:
			boolStr := "0"
			if v {
				boolStr = "1"
			}
			buf.WriteString(`<value><boolean>` + boolStr + `</boolean></value>`)
		}
		buf.WriteString(`</param>`)
	}

	buf.WriteString(`</params>`)
	buf.WriteString(`</methodCall>`)

	// Make HTTP request
	req, err := http.NewRequest("POST", sc.url, &buf)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "text/xml")

	resp, err := sc.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return body, nil
}
