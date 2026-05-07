// Package providerhttp contains small HTTP helpers shared by providers.
package providerhttp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// Fetch downloads rawURL with GET and requires HTTP 200 OK.
func Fetch(ctx context.Context, client *http.Client, rawURL string) ([]byte, error) {
	body, status, err := FetchStatus(ctx, client, rawURL)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("GET %s: %s", rawURL, http.StatusText(status))
	}
	return body, nil
}

// FetchStatus downloads rawURL with GET and returns response body plus status.
func FetchStatus(ctx context.Context, client *http.Client, rawURL string) ([]byte, int, error) {
	if isFileURL(rawURL) {
		return fetchFileStatus(rawURL)
	}
	if client == nil {
		client = http.DefaultClient
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("new request: %w", err)
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = response.Body.Close() }()
	data, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, 0, fmt.Errorf("read response: %w", err)
	}
	return data, response.StatusCode, nil
}

// Status requests rawURL with method and returns the HTTP response status.
func Status(ctx context.Context, client *http.Client, method string, rawURL string) (int, error) {
	if isFileURL(rawURL) {
		return fileStatus(rawURL)
	}
	if client == nil {
		client = http.DefaultClient
	}
	request, err := http.NewRequestWithContext(ctx, method, rawURL, nil)
	if err != nil {
		return 0, fmt.Errorf("new request: %w", err)
	}
	response, err := client.Do(request)
	if err != nil {
		return 0, err
	}
	defer func() { _ = response.Body.Close() }()
	return response.StatusCode, nil
}

// Probe returns whether rawURL exists. It treats 404 as absence and falls back
// from HEAD to GET when a server returns 405 Method Not Allowed.
func Probe(ctx context.Context, client *http.Client, rawURL string) (bool, error) {
	status, err := Status(ctx, client, http.MethodHead, rawURL)
	if err != nil {
		return false, err
	}
	if status == http.StatusMethodNotAllowed {
		status, err = Status(ctx, client, http.MethodGet, rawURL)
		if err != nil {
			return false, err
		}
	}
	if status == http.StatusNotFound {
		return false, nil
	}
	if status != http.StatusOK {
		return false, fmt.Errorf("probe %s: %s", rawURL, http.StatusText(status))
	}
	return true, nil
}

// EnsureTrailingSlash returns value with a trailing slash.
func EnsureTrailingSlash(value string) string {
	if strings.HasSuffix(value, "/") {
		return value
	}
	return value + "/"
}

// PathBase returns the final slash-separated component of value.
func PathBase(value string) string {
	value = strings.TrimRight(value, "/")
	if index := strings.LastIndex(value, "/"); index >= 0 {
		return value[index+1:]
	}
	return value
}

// LocalFileURL returns a file:// URL for local filesystem path.
func LocalFileURL(path string) string {
	return (&url.URL{Scheme: "file", Path: path}).String()
}

func isFileURL(rawURL string) bool {
	parsed, err := url.Parse(rawURL)
	return err == nil && parsed.Scheme == "file"
}

func fetchFileStatus(rawURL string) ([]byte, int, error) {
	path, status, err := filePath(rawURL)
	if err != nil || status != http.StatusOK {
		return nil, status, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, http.StatusNotFound, nil
		}
		return nil, 0, fmt.Errorf("read local metadata %s: %w", path, err)
	}
	return data, http.StatusOK, nil
}

func fileStatus(rawURL string) (int, error) {
	_, status, err := filePath(rawURL)
	return status, err
}

func filePath(rawURL string) (string, int, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", 0, fmt.Errorf("parse file URL: %w", err)
	}
	if parsed.Host != "" && parsed.Host != "localhost" {
		return "", 0, errors.New("file URL must reference a local path")
	}
	path, err := url.PathUnescape(parsed.Path)
	if err != nil {
		return "", 0, fmt.Errorf("unescape file path: %w", err)
	}
	if !filepath.IsAbs(path) {
		return "", 0, errors.New("file URL must reference an absolute path")
	}
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", http.StatusNotFound, nil
		}
		return "", 0, fmt.Errorf("stat local metadata %s: %w", path, err)
	}
	if info.IsDir() {
		path = filepath.Join(path, "index.html")
		info, err = os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				return "", http.StatusNotFound, nil
			}
			return "", 0, fmt.Errorf("stat local metadata %s: %w", path, err)
		}
	}
	if !info.Mode().IsRegular() {
		return "", 0, fmt.Errorf("local metadata %s is not a regular file", path)
	}
	return path, http.StatusOK, nil
}
