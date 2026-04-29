package trustmaterial_test

import (
	"testing"

	"github.com/dotwaffle/bootup/internal/trustmaterial"
)

func TestDebianArchiveKeyringReturnsCopy(t *testing.T) {
	first := trustmaterial.DebianArchiveKeyring()
	second := trustmaterial.DebianArchiveKeyring()
	if len(first) == 0 && len(second) == 0 {
		return
	}
	first[0] ^= 0xff
	if first[0] == second[0] {
		t.Fatal("DebianArchiveKeyring returned shared backing storage")
	}
}
