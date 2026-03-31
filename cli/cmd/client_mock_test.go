package cmd

import (
	"context"
	"io"
	"net/http"
	"sync"
)

// MockClient implements pkg.APIClient for testing
type MockClient struct {
	mu sync.Mutex

	GetFunc      func(path string, result interface{}) error
	PostFunc     func(path string, body interface{}, result interface{}) error
	PutFunc      func(path string, body interface{}, result interface{}) error
	DeleteFunc   func(path string) error
	DownloadFunc func(path string, out io.Writer) error
	StreamFunc   func(ctx context.Context, coreName string, tail int, follow bool, out io.Writer) error

	Calls []struct {
		Method string
		Path   string
		Body   interface{}
	}
}

func (m *MockClient) recordCall(method, path string, body interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls = append(m.Calls, struct {
		Method string
		Path   string
		Body   interface{}
	}{method, path, body})
}

func (m *MockClient) Get(path string, result interface{}) error {
	m.recordCall(http.MethodGet, path, nil)
	if m.GetFunc != nil {
		return m.GetFunc(path, result)
	}
	return nil
}

func (m *MockClient) Post(path string, body interface{}, result interface{}) error {
	m.recordCall(http.MethodPost, path, body)
	if m.PostFunc != nil {
		return m.PostFunc(path, body, result)
	}
	return nil
}

func (m *MockClient) Put(path string, body interface{}, result interface{}) error {
	m.recordCall(http.MethodPut, path, body)
	if m.PutFunc != nil {
		return m.PutFunc(path, body, result)
	}
	return nil
}

func (m *MockClient) Delete(path string) error {
	m.recordCall(http.MethodDelete, path, nil)
	if m.DeleteFunc != nil {
		return m.DeleteFunc(path)
	}
	return nil
}

func (m *MockClient) Download(path string, out io.Writer) error {
	m.recordCall("DOWNLOAD", path, nil)
	if m.DownloadFunc != nil {
		return m.DownloadFunc(path, out)
	}
	return nil
}

func (m *MockClient) StreamCoreLogs(ctx context.Context, coreName string, tail int, follow bool, out io.Writer) error {
	m.recordCall("STREAM", "/logs/"+coreName, nil)
	if m.StreamFunc != nil {
		return m.StreamFunc(ctx, coreName, tail, follow, out)
	}
	return nil
}
