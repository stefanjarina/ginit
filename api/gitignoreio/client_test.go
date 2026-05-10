package gitignoreio

import (
	"net/http"
	"net/http/httptest"
	"testing"
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
