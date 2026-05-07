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

func TestFetchAndStageArtifactsDefaultsToFileISOExtractor(t *testing.T) {
	t.Parallel()

	iso := tinyMFSBSDISO(t)
	loaderArchive := loaderTXZ(t)
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
	})
	if err != nil {
		t.Fatalf("fetch and stage: %v", err)
	}
	assertFile(t, filepath.Join(staged.FreeBSDKboot.PayloadRoot, "boot/kernel/kernel"), "kernel")
	assertFile(t, filepath.Join(staged.FreeBSDKboot.PayloadRoot, "mfsroot"), "mfsroot")
}

func TestFileISOExtractorExtractsRockRidgeNames(t *testing.T) {
	t.Parallel()

	isoPath := filepath.Join(t.TempDir(), "mfsbsd.iso")
	if err := os.WriteFile(isoPath, tinyMFSBSDISO(t), 0o644); err != nil {
		t.Fatalf("write test ISO: %v", err)
	}
	dest := t.TempDir()

	if err := (mfsbsd.FileISOExtractor{}).Extract(context.Background(), isoPath, dest); err != nil {
		t.Fatalf("extract ISO: %v", err)
	}
	assertGzipFile(t, filepath.Join(dest, "boot/kernel/kernel.gz"), "kernel")
	assertGzipFile(t, filepath.Join(dest, "mfsroot.gz"), "mfsroot")
	if _, err := os.Stat(filepath.Join(dest, "BOOT")); !os.IsNotExist(err) {
		t.Fatalf("uppercase ISO9660 path exists, want Rock Ridge lowercase names: %v", err)
	}
}

func TestFileISOExtractorExtractsConfiguredISO(t *testing.T) {
	t.Parallel()

	isoPath := os.Getenv("BOOTUP_MFSBSD_ISO_TESTDATA")
	if isoPath == "" {
		t.Skip("BOOTUP_MFSBSD_ISO_TESTDATA is required")
	}
	dest := t.TempDir()

	if err := (mfsbsd.FileISOExtractor{}).Extract(context.Background(), isoPath, dest); err != nil {
		t.Fatalf("extract ISO: %v", err)
	}
	assertReadableGzip(t, filepath.Join(dest, "boot/kernel/kernel.gz"))
	assertReadableGzip(t, filepath.Join(dest, "mfsroot.gz"))
}

func TestFileISOExtractorRejectsExtentPastEOF(t *testing.T) {
	t.Parallel()

	iso := tinyMFSBSDISO(t)
	patchISORecordSize(t, iso, "MFSROOT.GZ;1", uint32(len(iso)+1))
	isoPath := filepath.Join(t.TempDir(), "mfsbsd.iso")
	if err := os.WriteFile(isoPath, iso, 0o644); err != nil {
		t.Fatalf("write test ISO: %v", err)
	}

	err := (mfsbsd.FileISOExtractor{}).Extract(context.Background(), isoPath, t.TempDir())
	if err == nil {
		t.Fatal("extract ISO succeeded, want extent bounds error")
	}
	if !strings.Contains(err.Error(), "exceeds ISO size") {
		t.Fatalf("extract error = %v, want ISO size error", err)
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

func gzipData(t *testing.T, body string) []byte {
	t.Helper()

	var out bytes.Buffer
	writer := gzip.NewWriter(&out)
	if _, err := writer.Write([]byte(body)); err != nil {
		t.Fatalf("write gzip body: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close gzip writer: %v", err)
	}
	return out.Bytes()
}

func assertGzipFile(t *testing.T, path string, want string) {
	t.Helper()

	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer func() { _ = file.Close() }()
	reader, err := gzip.NewReader(file)
	if err != nil {
		t.Fatalf("open gzip %s: %v", path, err)
	}
	defer func() { _ = reader.Close() }()
	got, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read gzip %s: %v", path, err)
	}
	if string(got) != want {
		t.Fatalf("%s = %q, want %q", path, got, want)
	}
}

func assertReadableGzip(t *testing.T, path string) {
	t.Helper()

	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer func() { _ = file.Close() }()
	reader, err := gzip.NewReader(file)
	if err != nil {
		t.Fatalf("open gzip %s: %v", path, err)
	}
	defer func() { _ = reader.Close() }()
	buf := make([]byte, 1)
	if _, err := io.ReadFull(reader, buf); err != nil {
		t.Fatalf("read gzip %s: %v", path, err)
	}
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

func tinyMFSBSDISO(t *testing.T) []byte {
	t.Helper()

	const sectorSize = 2048
	const (
		pvdSector     = 16
		rootSector    = 20
		bootSector    = 21
		kernelSector  = 22
		mfsrootSector = 23
		kernelFile    = 24
		sectorCount   = 25
	)

	mfsroot := gzipData(t, "mfsroot")
	kernel := gzipData(t, "kernel")
	image := make([]byte, sectorCount*sectorSize)

	rootDir := isoDirSector(
		isoRecord("\x00", "", rootSector, sectorSize, true),
		isoRecord("\x01", "", rootSector, sectorSize, true),
		isoRecord("BOOT", "boot", bootSector, sectorSize, true),
		isoRecord("MFSROOT.GZ;1", "mfsroot.gz", mfsrootSector, len(mfsroot), false),
	)
	bootDir := isoDirSector(
		isoRecord("\x00", "", bootSector, sectorSize, true),
		isoRecord("\x01", "", rootSector, sectorSize, true),
		isoRecord("KERNEL", "kernel", kernelSector, sectorSize, true),
	)
	kernelDir := isoDirSector(
		isoRecord("\x00", "", kernelSector, sectorSize, true),
		isoRecord("\x01", "", bootSector, sectorSize, true),
		isoRecord("KERNEL.GZ;1", "kernel.gz", kernelFile, len(kernel), false),
	)

	copy(image[rootSector*sectorSize:], rootDir)
	copy(image[bootSector*sectorSize:], bootDir)
	copy(image[kernelSector*sectorSize:], kernelDir)
	copy(image[mfsrootSector*sectorSize:], mfsroot)
	copy(image[kernelFile*sectorSize:], kernel)

	pvd := image[pvdSector*sectorSize : (pvdSector+1)*sectorSize]
	pvd[0] = 1
	copy(pvd[1:6], "CD001")
	pvd[6] = 1
	putBothEndian16(pvd[128:132], sectorSize)
	copy(pvd[156:], isoRecord("\x00", "", rootSector, sectorSize, true))

	terminator := image[(pvdSector+1)*sectorSize : (pvdSector+2)*sectorSize]
	terminator[0] = 255
	copy(terminator[1:6], "CD001")
	terminator[6] = 1
	return image
}

func isoDirSector(records ...[]byte) []byte {
	const sectorSize = 2048

	sector := make([]byte, sectorSize)
	offset := 0
	for _, record := range records {
		copy(sector[offset:], record)
		offset += len(record)
	}
	return sector
}

func isoRecord(isoName string, rockRidgeName string, extent int, size int, dir bool) []byte {
	fileID := []byte(isoName)
	systemUse := rockRidgeNM(rockRidgeName)
	systemUseStart := 33 + len(fileID)
	if len(fileID)%2 == 0 {
		systemUseStart++
	}
	recordLen := systemUseStart + len(systemUse)
	if recordLen%2 != 0 {
		recordLen++
	}
	record := make([]byte, recordLen)
	record[0] = byte(recordLen)
	record[1] = 0
	putBothEndian32(record[2:10], uint32(extent))
	putBothEndian32(record[10:18], uint32(size))
	copy(record[18:25], []byte{126, 1, 1, 0, 0, 0, 0})
	if dir {
		record[25] = 0x02
	}
	putBothEndian16(record[28:32], 1)
	record[32] = byte(len(fileID))
	copy(record[33:], fileID)
	copy(record[systemUseStart:], systemUse)
	return record
}

func rockRidgeNM(name string) []byte {
	if name == "" {
		return nil
	}
	entryLen := 5 + len(name)
	entry := make([]byte, entryLen)
	copy(entry[0:2], "NM")
	entry[2] = byte(entryLen)
	entry[3] = 1
	entry[4] = 0
	copy(entry[5:], name)
	return entry
}

func putBothEndian16(target []byte, value int) {
	target[0] = byte(value)
	target[1] = byte(value >> 8)
	target[2] = byte(value >> 8)
	target[3] = byte(value)
}

func putBothEndian32(target []byte, value uint32) {
	target[0] = byte(value)
	target[1] = byte(value >> 8)
	target[2] = byte(value >> 16)
	target[3] = byte(value >> 24)
	target[4] = byte(value >> 24)
	target[5] = byte(value >> 16)
	target[6] = byte(value >> 8)
	target[7] = byte(value)
}

func patchISORecordSize(t *testing.T, image []byte, isoName string, size uint32) {
	t.Helper()

	const sectorSize = 2048
	const rootSector = 20

	root := image[rootSector*sectorSize : (rootSector+1)*sectorSize]
	for offset := 0; offset < len(root); {
		recordLen := int(root[offset])
		if recordLen == 0 {
			offset = ((offset / sectorSize) + 1) * sectorSize
			continue
		}
		if offset+recordLen > len(root) {
			t.Fatalf("record at offset %d exceeds image length", offset)
		}
		record := root[offset : offset+recordLen]
		fileIDLen := int(record[32])
		fileIDEnd := 33 + fileIDLen
		if fileIDEnd <= len(record) && string(record[33:fileIDEnd]) == isoName {
			putBothEndian32(record[10:18], size)
			return
		}
		offset += recordLen
	}
	t.Fatalf("ISO record %q not found", isoName)
}
