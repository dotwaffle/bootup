package policy_test

import (
	"crypto/ed25519"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/dotwaffle/bootup/internal/policy"
	"github.com/dotwaffle/bootup/internal/provider"
)

func TestLoadFileAuthenticatesAndParsesFreshDecision(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 5, 7, 10, 0, 0, 0, time.UTC)
	policyPath, signaturePath, publicKey := writeSignedPolicy(t, `{
		"schema_version": 1,
		"decision_id": "site-a-node-03",
		"target_id": "ubuntu-2604-amd64-netboot",
		"options": {"console": "serial"},
		"published_at": "2026-05-07T09:59:00Z",
		"expires_at": "2026-05-07T10:10:00Z"
	}`)
	signature, err := os.ReadFile(signaturePath)
	if err != nil {
		t.Fatalf("read signature: %v", err)
	}

	decision, err := policy.LoadFile(policy.LoadOptions{
		Path: policyPath,
		Trust: policy.Trust{
			Ed25519Signature: signature,
			Ed25519PublicKey: publicKey,
		},
		Now: func() time.Time { return now },
	})
	if err != nil {
		t.Fatalf("load policy: %v", err)
	}
	if decision.DecisionID != "site-a-node-03" || decision.TargetID != "ubuntu-2604-amd64-netboot" {
		t.Fatalf("decision = %#v, want parsed target decision", decision)
	}
	if decision.Options["console"] != "serial" {
		t.Fatalf("decision options = %#v, want console=serial", decision.Options)
	}
}

func TestLoadFileRejectsUnsafePolicyDecisions(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 5, 7, 10, 0, 0, 0, time.UTC)
	tests := []struct {
		name      string
		body      string
		mutateSig bool
		noTrust   bool
		maxAge    time.Duration
		want      string
	}{
		{
			name: "missing trust",
			body: `{
				"schema_version": 1,
				"decision_id": "site-a-node-03",
				"target_id": "ubuntu-2604-amd64-netboot",
				"expires_at": "2026-05-07T10:10:00Z"
			}`,
			noTrust: true,
			want:    "trust",
		},
		{
			name: "signature mismatch",
			body: `{
				"schema_version": 1,
				"decision_id": "site-a-node-03",
				"target_id": "ubuntu-2604-amd64-netboot",
				"expires_at": "2026-05-07T10:10:00Z"
			}`,
			mutateSig: true,
			want:      "signature verification failed",
		},
		{
			name: "expired",
			body: `{
				"schema_version": 1,
				"decision_id": "site-a-node-03",
				"target_id": "ubuntu-2604-amd64-netboot",
				"expires_at": "2026-05-07T09:59:00Z"
			}`,
			want: "expired",
		},
		{
			name: "past maximum age",
			body: `{
				"schema_version": 1,
				"decision_id": "site-a-node-03",
				"target_id": "ubuntu-2604-amd64-netboot",
				"published_at": "2026-05-07T09:00:00Z",
				"expires_at": "2026-05-07T11:00:00Z"
			}`,
			maxAge: 10 * time.Minute,
			want:   "maximum age",
		},
		{
			name: "missing freshness",
			body: `{
				"schema_version": 1,
				"decision_id": "site-a-node-03",
				"target_id": "ubuntu-2604-amd64-netboot"
			}`,
			want: "freshness",
		},
		{
			name: "executable field",
			body: `{
				"schema_version": 1,
				"decision_id": "site-a-node-03",
				"target_id": "ubuntu-2604-amd64-netboot",
				"script": "reboot",
				"expires_at": "2026-05-07T10:10:00Z"
			}`,
			want: "unknown field",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			policyPath, signaturePath, publicKey := writeSignedPolicy(t, tt.body)
			signature, err := os.ReadFile(signaturePath)
			if err != nil {
				t.Fatalf("read signature: %v", err)
			}
			if tt.mutateSig {
				signature[0] ^= 0xff
			}
			trust := policy.Trust{
				Ed25519Signature: signature,
				Ed25519PublicKey: publicKey,
			}
			if tt.noTrust {
				trust = policy.Trust{}
			}

			_, err = policy.LoadFile(policy.LoadOptions{
				Path:   policyPath,
				Trust:  trust,
				Now:    func() time.Time { return now },
				MaxAge: tt.maxAge,
			})
			if err == nil {
				t.Fatal("load policy succeeded, want error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("load policy error = %q, want %q", err, tt.want)
			}
		})
	}
}

func TestLoadFileFallsBackToAuthenticatedCache(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 5, 7, 10, 0, 0, 0, time.UTC)
	policyPath, signaturePath, publicKey := writeSignedPolicy(t, `{
		"schema_version": 1,
		"decision_id": "cached-site-a-node-03",
		"target_id": "ubuntu-2604-amd64-netboot",
		"expires_at": "2026-05-07T10:10:00Z"
	}`)
	signature, err := os.ReadFile(signaturePath)
	if err != nil {
		t.Fatalf("read signature: %v", err)
	}
	cachePath := filepath.Join(t.TempDir(), "policy-cache.json")
	data, err := os.ReadFile(policyPath)
	if err != nil {
		t.Fatalf("read policy: %v", err)
	}
	if err := os.WriteFile(cachePath, data, 0o644); err != nil {
		t.Fatalf("write cache: %v", err)
	}

	decision, err := policy.LoadFile(policy.LoadOptions{
		Path:          filepath.Join(t.TempDir(), "missing-policy.json"),
		CachePath:     cachePath,
		CacheFallback: true,
		Trust: policy.Trust{
			Ed25519Signature: signature,
			Ed25519PublicKey: publicKey,
		},
		Now: func() time.Time { return now },
	})
	if err != nil {
		t.Fatalf("load policy from cache: %v", err)
	}
	if decision.DecisionID != "cached-site-a-node-03" {
		t.Fatalf("decision ID = %q, want cached decision", decision.DecisionID)
	}
}

func TestValidateDecisionAgainstInventory(t *testing.T) {
	t.Parallel()

	target := policyTarget()
	selection, err := policy.Validate(policy.ValidateInput{
		Decision: policy.Decision{
			TargetID: "ubuntu-2604-amd64-netboot",
			Options:  map[string]string{"console": "serial"},
			SecretRefs: map[string]string{
				"installer-password": "site-installer-password",
			},
		},
		Targets: []provider.Target{target},
		Secrets: staticSecretStore{id: "site-installer-password"},
	})
	if err != nil {
		t.Fatalf("validate decision: %v", err)
	}
	if selection.Target.ID != target.ID {
		t.Fatalf("target ID = %q, want %q", selection.Target.ID, target.ID)
	}
	if !slices.Equal(selection.Options, []provider.SelectedOption{{ID: "console", Value: "serial"}}) {
		t.Fatalf("options = %#v, want console option", selection.Options)
	}
	if !slices.Equal(selection.SecretRefs, []provider.SecretRef{{ID: "installer-password", InputID: "site-installer-password"}}) {
		t.Fatalf("secret refs = %#v, want installer-password mapping", selection.SecretRefs)
	}
}

func TestValidateDecisionRejectsUnsupportedOutput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		decision policy.Decision
		secrets  provider.SecretStore
		want     string
	}{
		{
			name:     "unknown target",
			decision: policy.Decision{TargetID: "missing"},
			want:     "target",
		},
		{
			name: "unsupported option",
			decision: policy.Decision{
				TargetID: "ubuntu-2604-amd64-netboot",
				Options:  map[string]string{"bad-option": "true"},
			},
			want: "option",
		},
		{
			name: "undeclared secret",
			decision: policy.Decision{
				TargetID:   "ubuntu-2604-amd64-netboot",
				SecretRefs: map[string]string{"bad-secret": "site-installer-password"},
			},
			secrets: staticSecretStore{id: "site-installer-password"},
			want:    "secret",
		},
		{
			name: "missing required secret",
			decision: policy.Decision{
				TargetID: "ubuntu-2604-amd64-netboot",
			},
			want: "required secret",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := policy.Validate(policy.ValidateInput{
				Decision: tt.decision,
				Targets:  []provider.Target{policyTarget()},
				Secrets:  tt.secrets,
			})
			if err == nil {
				t.Fatal("validate decision succeeded, want error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("validate error = %q, want %q", err, tt.want)
			}
		})
	}
}

func writeSignedPolicy(t *testing.T, body string) (string, string, ed25519.PublicKey) {
	t.Helper()

	publicKey, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	dir := t.TempDir()
	policyPath := filepath.Join(dir, "policy.json")
	data := []byte(body)
	if err := os.WriteFile(policyPath, data, 0o644); err != nil {
		t.Fatalf("write policy: %v", err)
	}
	signature := ed25519.Sign(privateKey, data)
	signaturePath := filepath.Join(dir, "policy.json.sig")
	if err := os.WriteFile(signaturePath, signature, 0o644); err != nil {
		t.Fatalf("write signature: %v", err)
	}
	return policyPath, signaturePath, publicKey
}

func policyTarget() provider.Target {
	return provider.Target{
		ID:         "ubuntu-2604-amd64-netboot",
		ProviderID: "ubuntu",
		Name:       "Ubuntu 26.04 amd64 netboot",
		Catalog: provider.CatalogEntry{
			Distribution: "ubuntu",
			Release:      "26.04",
			Architecture: "amd64",
			Kind:         "installer",
		},
		Options: []provider.TargetOption{{
			ID:    "console",
			Label: "Console",
			Type:  provider.TargetOptionEnum,
			Values: []provider.TargetOptionValue{{
				Value:    "serial",
				Label:    "Serial",
				Fragment: "console=ttyS0",
			}},
		}},
		Secrets: []provider.SecretInput{{
			ID:       "installer-password",
			Label:    "Installer password",
			Purpose:  "Used by installer automation.",
			Required: true,
			Delivery: provider.SecretDeliveryStagedFile,
		}},
	}
}

type staticSecretStore struct {
	id string
}

func (s staticSecretStore) IDs() []string {
	return []string{s.id}
}

func (s staticSecretStore) Has(id string) bool {
	return id == s.id
}

func (s staticSecretStore) StageFile(string, string, string) (string, error) {
	return "", nil
}
