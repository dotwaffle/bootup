package mfsbsd_test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dotwaffle/bootup/internal/provider"
	"github.com/dotwaffle/bootup/internal/providers/mfsbsd"
	"github.com/ulikunitz/xz"
)

func TestProviderPlanBuildsFreeBSDKbootPlan(t *testing.T) {
	t.Parallel()

	target := mfsbsdTarget()
	p := mfsbsd.NewProvider(mfsbsd.Config{
		Targets:             []provider.Target{target},
		LoaderArchiveURL:    "https://download.freebsd.org/releases/amd64/amd64/15.0-RELEASE/base.txz",
		LoaderArchiveSHA256: strings.Repeat("b", 64),
	})

	plan, err := p.Plan(context.Background(), provider.PlanInput{Target: target})
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if plan.Action != provider.BootActionFreeBSDKboot {
		t.Fatalf("plan action = %q, want freebsd-kboot", plan.Action)
	}
	if plan.FreeBSDKboot.Payload.URL != "https://mfsbsd.example/files/iso/14/amd64/mfsbsd-14.2-RELEASE-amd64.iso" {
		t.Fatalf("payload URL = %q", plan.FreeBSDKboot.Payload.URL)
	}
	if plan.FreeBSDKboot.Payload.SHA256 != strings.Repeat("a", 64) {
		t.Fatalf("payload SHA256 = %q", plan.FreeBSDKboot.Payload.SHA256)
	}
	if plan.FreeBSDKboot.LoaderArchive.URL == "" || plan.FreeBSDKboot.LoaderArchive.SHA256 != strings.Repeat("b", 64) {
		t.Fatalf("loader archive = %#v, want configured archive", plan.FreeBSDKboot.LoaderArchive)
	}
}

func TestFetchAndStageArtifactsExtractsMemoryRootPayload(t *testing.T) {
	t.Parallel()

	iso := []byte("iso")
	loaderArchive := loaderTXZ(t)
	extractor := &fakeExtractor{}
	plan := provider.BootPlan{
		Action: provider.BootActionFreeBSDKboot,
		Target: provider.Target{ID: "mfsbsd-142-amd64", ProviderID: "mfsbsd"},
		FreeBSDKboot: provider.FreeBSDKbootPlan{
			Payload: provider.Artifact{
				Name:   "mfsbsd-14.2-RELEASE-amd64.iso",
				URL:    "https://mfsbsd.example/mfsbsd.iso",
				SHA256: sha256Hex(iso),
			},
			LoaderArchive: provider.Artifact{
				Name:   "base.txz",
				URL:    "https://download.example/base.txz",
				SHA256: sha256Hex(loaderArchive),
			},
		},
	}
	client := &http.Client{Transport: responseMap{
		"https://mfsbsd.example/mfsbsd.iso": body(iso),
		"https://download.example/base.txz": body(loaderArchive),
	}}

	staged, err := mfsbsd.FetchAndStageArtifacts(context.Background(), mfsbsd.FetchConfig{
		Plan:       plan,
		Client:     client,
		StagingDir: t.TempDir(),
		Extractor:  extractor,
	})
	if err != nil {
		t.Fatalf("fetch and stage: %v", err)
	}
	if len(extractor.calls) != 1 {
		t.Fatalf("extract calls = %#v, want one call", extractor.calls)
	}
	if staged.FreeBSDKboot.Loader.Path == "" || staged.FreeBSDKboot.LoaderHelp.Path == "" {
		t.Fatalf("staged loader paths = %#v", staged.FreeBSDKboot)
	}
	assertFile(t, staged.FreeBSDKboot.Loader.Path, "loader")
	assertFile(t, staged.FreeBSDKboot.LoaderHelp.Path, "help")
	assertFile(t, filepath.Join(staged.FreeBSDKboot.PayloadRoot, "boot/kernel/kernel"), "kernel")
	assertFile(t, filepath.Join(staged.FreeBSDKboot.PayloadRoot, "mfsroot"), "mfsroot")
	if len(staged.FreeBSDKboot.Args) == 0 || staged.FreeBSDKboot.Args[0] != "hostfs_root="+staged.FreeBSDKboot.PayloadRoot {
		t.Fatalf("loader args = %#v, want hostfs root first", staged.FreeBSDKboot.Args)
	}
}

type extractCall struct {
	isoPath string
	dest    string
}

type fakeExtractor struct {
	calls []extractCall
}

func (e *fakeExtractor) Extract(_ context.Context, isoPath string, dest string) error {
	e.calls = append(e.calls, extractCall{isoPath: isoPath, dest: dest})
	if err := os.MkdirAll(filepath.Join(dest, "boot/kernel"), 0o755); err != nil {
		return err
	}
	if err := writeGzip(filepath.Join(dest, "boot/kernel/kernel.gz"), "kernel"); err != nil {
		return err
	}
	return writeGzip(filepath.Join(dest, "mfsroot.gz"), "mfsroot")
}

type responseMap map[string]httpBody

type httpBody struct {
	data []byte
}

func (m responseMap) RoundTrip(request *http.Request) (*http.Response, error) {
	response, ok := m[request.URL.String()]
	if !ok {
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Status:     "404 Not Found",
			Body:       io.NopCloser(strings.NewReader("not found")),
			Header:     make(http.Header),
			Request:    request,
		}, nil
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Status:     "200 OK",
		Body:       io.NopCloser(bytes.NewReader(response.data)),
		Header:     make(http.Header),
		Request:    request,
	}, nil
}

func body(data []byte) httpBody {
	return httpBody{data: data}
}

func mfsbsdTarget() provider.Target {
	return provider.Target{
		ID:         "mfsbsd-142-amd64",
		ProviderID: "mfsbsd",
		Name:       "mfsBSD 14.2 amd64",
		Action:     provider.BootActionFreeBSDKboot,
		Catalog: provider.CatalogEntry{
			Distribution: "mfsbsd",
			Release:      "14.2",
			Architecture: "amd64",
			Kind:         "rescue",
		},
		Source: provider.SourceEntry{
			BaseURL:   "https://mfsbsd.example/files/iso/14/amd64",
			ISOName:   "mfsbsd-14.2-RELEASE-amd64.iso",
			ISOSHA256: strings.Repeat("a", 64),
		},
	}
}

func loaderTXZ(t *testing.T) []byte {
	t.Helper()

	var out bytes.Buffer
	xzWriter, err := xz.NewWriter(&out)
	if err != nil {
		t.Fatalf("create xz writer: %v", err)
	}
	tarWriter := tar.NewWriter(xzWriter)
	for name, body := range map[string]string{
		"./boot/loader.kboot":      "loader",
		"./boot/loader.help.kboot": "help",
	} {
		if err := tarWriter.WriteHeader(&tar.Header{Name: name, Mode: 0o555, Size: int64(len(body))}); err != nil {
			t.Fatalf("write tar header: %v", err)
		}
		if _, err := tarWriter.Write([]byte(body)); err != nil {
			t.Fatalf("write tar body: %v", err)
		}
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatalf("close tar writer: %v", err)
	}
	if err := xzWriter.Close(); err != nil {
		t.Fatalf("close xz writer: %v", err)
	}
	return out.Bytes()
}

func writeGzip(path string, body string) (err error) {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := file.Close(); err == nil && closeErr != nil {
			err = closeErr
		}
	}()
	writer := gzip.NewWriter(file)
	if _, err := writer.Write([]byte(body)); err != nil {
		return err
	}
	return writer.Close()
}

func assertFile(t *testing.T, path string, want string) {
	t.Helper()

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if string(got) != want {
		t.Fatalf("%s = %q, want %q", path, got, want)
	}
}

func sha256Hex(data []byte) string {
	return fmt.Sprintf("%x", sha256.Sum256(data))
}
