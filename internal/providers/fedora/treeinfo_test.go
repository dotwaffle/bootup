package fedora

import (
	"strings"
	"testing"
)

func TestParseTreeinfoChecksumsExtractsPXEBootSHA256(t *testing.T) {
	t.Parallel()

	got, err := parseTreeinfoChecksums([]byte(`
[checksums]
images/boot.iso = sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc
images/pxeboot/initrd.img = sha256:BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB
images/pxeboot/vmlinuz = sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa

[images-x86_64]
initrd = images/pxeboot/initrd.img
kernel = images/pxeboot/vmlinuz
`))
	if err != nil {
		t.Fatalf("parse checksums: %v", err)
	}
	if got.kernelSHA256 != strings.Repeat("a", 64) {
		t.Fatalf("kernel hash = %q, want lowercase SHA-256", got.kernelSHA256)
	}
	if got.initrdSHA256 != strings.Repeat("b", 64) {
		t.Fatalf("initrd hash = %q, want lowercase SHA-256", got.initrdSHA256)
	}
}

func TestParseTreeinfoChecksumsRejectsIncompleteMetadata(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		data string
		want string
	}{
		{
			name: "missing checksums",
			data: `[images-x86_64]
kernel = images/pxeboot/vmlinuz
`,
			want: "images/pxeboot/vmlinuz",
		},
		{
			name: "missing initrd",
			data: `[checksums]
images/pxeboot/vmlinuz = sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
`,
			want: "images/pxeboot/initrd.img",
		},
		{
			name: "unsupported algorithm",
			data: `[checksums]
images/pxeboot/vmlinuz = sha512:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
images/pxeboot/initrd.img = sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb
`,
			want: "sha256",
		},
		{
			name: "invalid digest",
			data: `[checksums]
images/pxeboot/vmlinuz = sha256:not-a-digest
images/pxeboot/initrd.img = sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb
`,
			want: "64-character SHA-256",
		},
		{
			name: "malformed line",
			data: `[checksums]
images/pxeboot/vmlinuz sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
images/pxeboot/initrd.img = sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb
`,
			want: "parse .treeinfo checksum",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := parseTreeinfoChecksums([]byte(tt.data))
			if err == nil {
				t.Fatal("parse checksums succeeded, want error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("parse error = %q, want %q", err, tt.want)
			}
		})
	}
}
