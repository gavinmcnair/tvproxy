package mtls

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveCertsAndHasCerts(t *testing.T) {
	dir := t.TempDir()
	accountID := "test-account-123"

	if HasCerts(dir, accountID) {
		t.Fatal("should not have certs yet")
	}

	result := &EnrollResult{
		Cert: "-----BEGIN CERTIFICATE-----\ntest\n-----END CERTIFICATE-----\n",
		Key:  "-----BEGIN EC PRIVATE KEY-----\ntest\n-----END EC PRIVATE KEY-----\n",
		CA:   "-----BEGIN CERTIFICATE-----\nca\n-----END CERTIFICATE-----\n",
	}

	if err := SaveCerts(dir, accountID, result); err != nil {
		t.Fatal(err)
	}

	if !HasCerts(dir, accountID) {
		t.Fatal("should have certs after save")
	}

	certPath := filepath.Join(dir, "certs", accountID, "client.crt")
	data, err := os.ReadFile(certPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != result.Cert {
		t.Errorf("cert content mismatch")
	}
}

func TestDeleteCerts(t *testing.T) {
	dir := t.TempDir()
	accountID := "delete-test"

	result := &EnrollResult{Cert: "cert", Key: "key", CA: "ca"}
	SaveCerts(dir, accountID, result)

	if !HasCerts(dir, accountID) {
		t.Fatal("should have certs")
	}

	DeleteCerts(dir, accountID)

	if HasCerts(dir, accountID) {
		t.Fatal("should not have certs after delete")
	}
}

func TestHasCertsNonexistent(t *testing.T) {
	dir := t.TempDir()
	if HasCerts(dir, "nonexistent") {
		t.Fatal("should not have certs for nonexistent account")
	}
}
