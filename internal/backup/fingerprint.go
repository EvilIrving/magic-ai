package backup

import (
	"crypto/sha256"
	"encoding/hex"
)

// Fingerprint is a content digest. It never carries plaintext, so a plan and a
// manifest can be logged, diffed or committed without leaking a key.
type Fingerprint string

// FingerprintOf returns a short digest of b: "sha256:" plus the first 12 hex
// digits. plan.json and manifest.json record the same form, and rollback
// compares them, so there is one implementation here and core calls it.
func FingerprintOf(b []byte) Fingerprint {
	sum := sha256.Sum256(b)
	return Fingerprint("sha256:" + hex.EncodeToString(sum[:])[:12])
}
