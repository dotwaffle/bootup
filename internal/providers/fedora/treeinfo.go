package fedora

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

const (
	kernelTreeinfoPath = "images/pxeboot/vmlinuz"
	initrdTreeinfoPath = "images/pxeboot/initrd.img"
)

type treeinfoChecksums struct {
	kernelSHA256 string
	initrdSHA256 string
}

func parseTreeinfoChecksums(data []byte) (treeinfoChecksums, error) {
	checksums := make(map[string]string)
	section := ""
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.ToLower(strings.TrimSpace(line[1 : len(line)-1]))
			continue
		}
		if section != "checksums" {
			continue
		}
		path, value, ok := strings.Cut(line, "=")
		if !ok {
			return treeinfoChecksums{}, fmt.Errorf("parse .treeinfo checksum line %q", line)
		}
		path = strings.TrimSpace(path)
		if path != kernelTreeinfoPath && path != initrdTreeinfoPath {
			continue
		}
		digest, err := parseTreeinfoSHA256(path, value)
		if err != nil {
			return treeinfoChecksums{}, err
		}
		checksums[path] = digest
	}
	if err := scanner.Err(); err != nil {
		return treeinfoChecksums{}, fmt.Errorf("read .treeinfo: %w", err)
	}

	kernelSHA256, ok := checksums[kernelTreeinfoPath]
	if !ok {
		return treeinfoChecksums{}, fmt.Errorf("fedora .treeinfo missing checksum for %s", kernelTreeinfoPath)
	}
	initrdSHA256, ok := checksums[initrdTreeinfoPath]
	if !ok {
		return treeinfoChecksums{}, fmt.Errorf("fedora .treeinfo missing checksum for %s", initrdTreeinfoPath)
	}
	return treeinfoChecksums{
		kernelSHA256: kernelSHA256,
		initrdSHA256: initrdSHA256,
	}, nil
}

func parseTreeinfoSHA256(path string, value string) (string, error) {
	algorithm, digest, ok := strings.Cut(strings.TrimSpace(value), ":")
	if !ok || !strings.EqualFold(strings.TrimSpace(algorithm), "sha256") {
		return "", fmt.Errorf("fedora .treeinfo checksum for %s must use sha256", path)
	}
	digest = strings.ToLower(strings.TrimSpace(digest))
	decoded, err := hex.DecodeString(digest)
	if err != nil || len(decoded) != sha256.Size {
		return "", fmt.Errorf("fedora .treeinfo checksum for %s must be a 64-character SHA-256 hex digest", path)
	}
	return digest, nil
}
