package main

import (
	"bytes"
	"testing"
)

func TestFormatSourceWritesTrustMaterialInitializer(t *testing.T) {
	t.Parallel()

	source := formatSource([]byte{0x01, 0x02, 0xff})

	if !bytes.Contains(source, []byte("package trustmaterial")) {
		t.Fatalf("source = %q, want trustmaterial package", source)
	}
	if !bytes.Contains(source, []byte("0x01, 0x02, 0xff")) {
		t.Fatalf("source = %q, want byte literal", source)
	}
	if !bytes.Contains(source, []byte("debianArchiveKeyring")) {
		t.Fatalf("source = %q, want keyring assignment", source)
	}
}
