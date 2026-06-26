package serve

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"filippo.io/age"
)

// newTestSession builds a TransferSession that writes into a fresh temp dir,
// using a low scrypt work factor to keep encryption fast in tests.
func newTestSession(t *testing.T, password string) *TransferSession {
	t.Helper()
	recipient, err := age.NewScryptRecipient(password)
	if err != nil {
		t.Fatalf("creating recipient: %v", err)
	}
	recipient.SetWorkFactor(10)
	dir := t.TempDir()
	return &TransferSession{
		Identifier: "synco-test",
		WorkDir:    &dir,
		Password:   password,
		recipient:  recipient,
	}
}

// decryptFile reads and decrypts a file written by the TransferSession, failing
// the test if either the header or the payload cannot be decrypted.
func decryptFile(t *testing.T, ts *TransferSession, fileName string) []byte {
	t.Helper()
	encrypted, err := os.ReadFile(filepath.Join(*ts.WorkDir, fileName))
	if err != nil {
		t.Fatalf("reading encrypted file: %v", err)
	}
	identity, err := age.NewScryptIdentity(ts.Password)
	if err != nil {
		t.Fatalf("creating identity: %v", err)
	}
	r, err := age.Decrypt(bytes.NewReader(encrypted), identity)
	if err != nil {
		t.Fatalf("age.Decrypt (header): %v", err)
	}
	decrypted, err := io.ReadAll(r)
	if err != nil {
		// This is the symptom of the O_TRUNC bug: a stale tail left behind by a
		// previous, longer write corrupts the age stream's final chunk.
		t.Fatalf("decrypting payload: %v", err)
	}
	return decrypted
}

// Regression test for the EncryptToFile O_TRUNC bug: writing a shorter payload
// onto a pre-existing, longer encrypted file must not leave a stale tail behind.
// This mirrors UpdateMetadata rewriting meta.json.enc, which shrinks on the
// final Initializing -> Ready transition. Without O_TRUNC the leftover bytes
// get appended to the age stream and `receive` fails with
// "failed to decrypt and authenticate payload chunk" even though the password
// is valid.
func TestEncryptToFileTruncatesExistingFile(t *testing.T) {
	const password = "super-secret-pass"
	ts := newTestSession(t, password)

	const fileName = "meta.json.enc"
	long := bytes.Repeat([]byte("LONG-INITIALIZING-CONTENT\n"), 100)
	short := []byte("short ready content")

	if err := ts.EncryptBytesToFile(fileName, long); err != nil {
		t.Fatalf("writing long payload: %v", err)
	}
	if err := ts.EncryptBytesToFile(fileName, short); err != nil {
		t.Fatalf("writing short payload: %v", err)
	}

	got := decryptFile(t, ts, fileName)
	if !bytes.Equal(got, short) {
		t.Fatalf("decrypted content mismatch after overwrite:\n got: %q\nwant: %q", got, short)
	}
}
