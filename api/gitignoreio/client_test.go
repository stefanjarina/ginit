package gitignoreio

import (
	"bufio"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestFetchConfigUsesAPISlash(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/windows,linux" {
			t.Fatalf("path = %q, want /api/windows,linux", r.URL.Path)
		}
		_, _ = w.Write([]byte("ignore-content"))
	})

	client := NewClientWithBaseURL("http://example.test/api", &http.Client{Transport: handlerTransport(handler)})
	got, err := client.FetchConfig([]string{"windows", "linux"})
	if err != nil {
		t.Fatalf("FetchConfig() error = %v", err)
	}
	if got != "ignore-content" {
		t.Fatalf("FetchConfig() = %q, want ignore-content", got)
	}
}

func TestFetchConfigFailsOnNon2xx(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "missing", http.StatusNotFound)
	})

	client := NewClientWithBaseURL("http://example.test/api", &http.Client{Transport: handlerTransport(handler)})
	if _, err := client.FetchConfig([]string{"unknown"}); err == nil {
		t.Fatal("FetchConfig() error = nil, want non-2xx error")
	}
}

var errBodyRead = errors.New("connection reset")

// failingBodyTransport returns a 200 response whose body yields prefix and
// then fails, simulating a connection dropped mid-download.
func failingBodyTransport(prefix string) http.RoundTripper {
	return roundTripFunc(func(r *http.Request) (*http.Response, error) {
		body := io.MultiReader(strings.NewReader(prefix), errReader{})
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(body),
			Header:     make(http.Header),
			Request:    r,
		}, nil
	})
}

type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, errBodyRead }

func TestListFailsOnBodyReadError(t *testing.T) {
	client := NewClientWithBaseURL("http://example.test/api", &http.Client{Transport: failingBodyTransport("go,rust\nlinux,windows\n")})
	got, err := client.List()
	if !errors.Is(err, errBodyRead) {
		t.Fatalf("List() error = %v, want %v", err, errBodyRead)
	}
	if got != nil {
		t.Fatalf("List() = %v, want nil on read error", got)
	}
}

func TestListFailsOnOverlongLine(t *testing.T) {
	line := strings.Repeat("a", bufio.MaxScanTokenSize+1)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("go,rust\n" + line + "\n"))
	})

	client := NewClientWithBaseURL("http://example.test/api", &http.Client{Transport: handlerTransport(handler)})
	if _, err := client.List(); !errors.Is(err, bufio.ErrTooLong) {
		t.Fatalf("List() error = %v, want %v", err, bufio.ErrTooLong)
	}
}

func TestListParsesNames(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("go,rust\nlinux\n"))
	})

	client := NewClientWithBaseURL("http://example.test/api", &http.Client{Transport: handlerTransport(handler)})
	got, err := client.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if strings.Join(got, ",") != "go,rust,linux" {
		t.Fatalf("List() = %v, want [go rust linux]", got)
	}
}

func TestFetchConfigFailsOnBodyReadError(t *testing.T) {
	client := NewClientWithBaseURL("http://example.test/api", &http.Client{Transport: failingBodyTransport("partial")})
	if _, err := client.FetchConfig([]string{"go"}); !errors.Is(err, errBodyRead) {
		t.Fatalf("FetchConfig() error = %v, want %v", err, errBodyRead)
	}
}

func TestFetchAllFailsOnBodyReadError(t *testing.T) {
	client := NewClientWithBaseURL("http://example.test/api", &http.Client{Transport: failingBodyTransport(`{"go":`)})
	if _, err := client.FetchAll(); !errors.Is(err, errBodyRead) {
		t.Fatalf("FetchAll() error = %v, want %v", err, errBodyRead)
	}
}

func TestDefaultClientTimeout(t *testing.T) {
	if got := NewClient().httpClient.Timeout; got != defaultTimeout {
		t.Fatalf("NewClient() timeout = %v, want %v", got, defaultTimeout)
	}
	if got := NewClientWithBaseURL("http://example.test", nil).httpClient.Timeout; got != 30*time.Second {
		t.Fatalf("NewClientWithBaseURL(nil) timeout = %v, want 30s", got)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func handlerTransport(handler http.Handler) http.RoundTripper {
	return roundTripFunc(func(r *http.Request) (*http.Response, error) {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, r)
		return rec.Result(), nil
	})
}
