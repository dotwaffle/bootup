package providerhttp_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/dotwaffle/bootup/internal/providerhttp"
)

func TestFetchStatusReturnsBodyAndStatus(t *testing.T) {
	t.Parallel()

	client := &http.Client{Transport: responseMap{
		"https://example.test/index": response{statusCode: http.StatusAccepted, body: []byte("metadata")},
	}}

	body, status, err := providerhttp.FetchStatus(context.Background(), client, "https://example.test/index")
	if err != nil {
		t.Fatalf("fetch status: %v", err)
	}
	if status != http.StatusAccepted {
		t.Fatalf("status = %d, want 202", status)
	}
	if string(body) != "metadata" {
		t.Fatalf("body = %q, want metadata", body)
	}
}

func TestProbeTreats404AsAbsentAndFallsBackToGET(t *testing.T) {
	t.Parallel()

	client := &http.Client{Transport: responseMap{
		"HEAD https://example.test/missing":  response{statusCode: http.StatusNotFound},
		"HEAD https://example.test/get-only": response{statusCode: http.StatusMethodNotAllowed},
		"GET https://example.test/get-only":  response{statusCode: http.StatusOK},
	}}

	ok, err := providerhttp.Probe(context.Background(), client, "https://example.test/missing")
	if err != nil {
		t.Fatalf("probe missing: %v", err)
	}
	if ok {
		t.Fatal("probe missing returned true, want false")
	}

	ok, err = providerhttp.Probe(context.Background(), client, "https://example.test/get-only")
	if err != nil {
		t.Fatalf("probe get-only: %v", err)
	}
	if !ok {
		t.Fatal("probe get-only returned false, want true")
	}
}

func TestProbeRejectsUnexpectedStatus(t *testing.T) {
	t.Parallel()

	client := &http.Client{Transport: responseMap{
		"HEAD https://example.test/error": response{statusCode: http.StatusInternalServerError},
	}}

	_, err := providerhttp.Probe(context.Background(), client, "https://example.test/error")
	if err == nil {
		t.Fatal("probe succeeded, want unexpected status error")
	}
	if !strings.Contains(err.Error(), "Internal Server Error") {
		t.Fatalf("probe error = %v, want status text", err)
	}
}

func TestURLPathHelpers(t *testing.T) {
	t.Parallel()

	if got := providerhttp.EnsureTrailingSlash("https://example.test/releases"); got != "https://example.test/releases/" {
		t.Fatalf("ensure trailing slash = %q", got)
	}
	if got := providerhttp.PathBase("/pub/fedora/44/"); got != "44" {
		t.Fatalf("path base = %q, want 44", got)
	}
}

type response struct {
	statusCode int
	body       []byte
}

type responseMap map[string]response

func (m responseMap) RoundTrip(request *http.Request) (*http.Response, error) {
	key := request.Method + " " + request.URL.String()
	item, ok := m[key]
	if !ok {
		item, ok = m[request.URL.String()]
	}
	if !ok {
		item = response{statusCode: http.StatusNotFound, body: []byte("not found")}
	}
	return &http.Response{
		StatusCode: item.statusCode,
		Status:     http.StatusText(item.statusCode),
		Body:       io.NopCloser(bytes.NewReader(item.body)),
		Header:     make(http.Header),
		Request:    request,
	}, nil
}
