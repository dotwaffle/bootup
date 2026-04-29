//go:build bootup_debian_fixture

// Package debianfixture provides hermetic Debian provider data for VM tests.
package debianfixture

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/armor"
	"github.com/ProtonMail/go-crypto/openpgp/clearsign"
	"github.com/dotwaffle/bootup/internal/providers/debian"
)

const mirrorURL = "https://fixture.invalid/debian"

// NewProvider returns a Debian provider backed by signed in-memory fixture
// metadata and artifacts.
func NewProvider() (*debian.Provider, error) {
	keyring, signed, shaSums, kernel, initrd, err := fixtureData()
	if err != nil {
		return nil, err
	}
	client := &http.Client{Transport: responseMap{
		mirrorURL + "/dists/trixie/InRelease":                                                                    signed,
		mirrorURL + "/dists/trixie/main/installer-amd64/current/images/SHA256SUMS":                               shaSums,
		mirrorURL + "/dists/trixie/main/installer-amd64/current/images/netboot/debian-installer/amd64/linux":     kernel,
		mirrorURL + "/dists/trixie/main/installer-amd64/current/images/netboot/debian-installer/amd64/initrd.gz": initrd,
	}}
	return debian.NewProvider(debian.Config{
		MirrorURL: mirrorURL,
		Client:    client,
		Keyring:   keyring,
	}), nil
}

type responseMap map[string][]byte

func (m responseMap) RoundTrip(request *http.Request) (*http.Response, error) {
	data, ok := m[request.URL.String()]
	if !ok {
		return nil, fmt.Errorf("fixture response not found for %s", request.URL)
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Status:     "200 OK",
		Header:     make(http.Header),
		Body:       io.NopCloser(bytes.NewReader(data)),
		Request:    request,
	}, nil
}

func fixtureData() (keyring []byte, signedRelease []byte, shaSums []byte, kernel []byte, initrd []byte, err error) {
	kernel = []byte("fixture kernel\n")
	initrd = []byte("fixture initrd\n")
	kernelSum := sha256.Sum256(kernel)
	initrdSum := sha256.Sum256(initrd)
	shaSums = fmt.Appendf(nil,
		"%x  ./netboot/debian-installer/amd64/linux\n%x  ./netboot/debian-installer/amd64/initrd.gz\n",
		kernelSum,
		initrdSum,
	)
	shaSumsSum := sha256.Sum256(shaSums)
	release := fmt.Appendf(nil, "SHA256:\n %x %d main/installer-amd64/current/images/SHA256SUMS\n", shaSumsSum, len(shaSums))

	entity, err := openpgp.NewEntity("Bootup Debian Fixture", "", "fixture@example.invalid", nil)
	if err != nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("create fixture signer: %w", err)
	}

	var keyringBuffer bytes.Buffer
	armorWriter, err := armor.Encode(&keyringBuffer, openpgp.PublicKeyType, nil)
	if err != nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("armor fixture keyring: %w", err)
	}
	if err := entity.Serialize(armorWriter); err != nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("serialize fixture keyring: %w", err)
	}
	if err := armorWriter.Close(); err != nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("close fixture keyring: %w", err)
	}

	var signed bytes.Buffer
	writer, err := clearsign.Encode(&signed, entity.PrivateKey, nil)
	if err != nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("clearsign fixture release: %w", err)
	}
	if _, err := writer.Write(release); err != nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("write fixture release: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("close fixture release: %w", err)
	}
	return keyringBuffer.Bytes(), signed.Bytes(), shaSums, kernel, initrd, nil
}
