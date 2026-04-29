// Package trustmaterial provides optional compile-time trust roots.
package trustmaterial

import "bytes"

var debianArchiveKeyring []byte

// DebianArchiveKeyring returns configured Debian archive OpenPGP public key
// material.
//
// The default repository build returns nil. Local application builds can add an
// ignored generated Go source file in this package that sets
// debianArchiveKeyring during init. The keyring must be OpenPGP public key
// material accepted by verify.ReadKeyring, such as an exported ASCII-armored
// public key or a binary Debian archive keyring file.
func DebianArchiveKeyring() []byte {
	return bytes.Clone(debianArchiveKeyring)
}
