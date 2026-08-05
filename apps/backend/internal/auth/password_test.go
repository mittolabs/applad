package auth

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"testing"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/bcrypt"
)

// TestFirebaseScryptVector verifies the modified-scrypt implementation against
// Firebase's own published test vector. If this passes, the algorithm (scrypt
// params, key length, AES-256-CTR with a zero IV over the signer key, base64
// comparison) matches Firebase exactly and imported Firebase passwords will
// verify.
func TestFirebaseScryptVector(t *testing.T) {
	params := map[string]any{
		"salt":          "42xEC+ixf3L2lw==",
		"saltSeparator": "Bw==",
		"signerKey":     "jxspr8Ki0RYycVU8zykbdLGjFQ3McFUH0uiiTvC8pVMXAn210wjLNmdZJzxUECKbm0QsEmYUSDzZvpjeJ9WmXA==",
		"rounds":        8,
		"memCost":       14,
	}
	const hash = "lSrfV15cpx95/sZS2W9c9Kp6i/LVgQNDNC/qzrCnh1SAyZvqmZqAjTdn3aoItz+VHjoZilo78198JAdRuid5lQ=="

	ok, err := verifyForeignPassword(AlgoScryptFirebase, hash, params, "user1password")
	if err != nil {
		t.Fatalf("firebase scrypt: unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("firebase scrypt: correct password rejected (algorithm mismatch)")
	}

	bad, err := verifyForeignPassword(AlgoScryptFirebase, hash, params, "wrongpassword")
	if err != nil {
		t.Fatalf("firebase scrypt: unexpected error on wrong password: %v", err)
	}
	if bad {
		t.Fatal("firebase scrypt: wrong password accepted")
	}
}

func TestBcryptVerify(t *testing.T) {
	h, _ := bcrypt.GenerateFromPassword([]byte("hunter2"), 10)
	ok, err := verifyForeignPassword(AlgoBcrypt, string(h), nil, "hunter2")
	if err != nil || !ok {
		t.Fatalf("bcrypt: correct password rejected (ok=%v err=%v)", ok, err)
	}
	ok, _ = verifyForeignPassword(AlgoBcrypt, string(h), nil, "nope")
	if ok {
		t.Fatal("bcrypt: wrong password accepted")
	}
	// Empty algo must behave as bcrypt (backward compatibility).
	ok, err = verifyForeignPassword("", string(h), nil, "hunter2")
	if err != nil || !ok {
		t.Fatalf("empty algo: expected bcrypt behavior (ok=%v err=%v)", ok, err)
	}
}

func TestArgon2idVerify(t *testing.T) {
	// Build a genuine PHC string the same way Appwrite/passlib would, then verify.
	salt := []byte("0123456789abcdef")
	var m, tCost uint32 = 65536, 3
	var p uint8 = 2
	key := argon2.IDKey([]byte("s3cret"), salt, tCost, m, p, 32)
	phc := "$argon2id$v=19$m=65536,t=3,p=2$" +
		base64.RawStdEncoding.EncodeToString(salt) + "$" +
		base64.RawStdEncoding.EncodeToString(key)

	ok, err := verifyForeignPassword(AlgoArgon2id, phc, nil, "s3cret")
	if err != nil || !ok {
		t.Fatalf("argon2id: correct password rejected (ok=%v err=%v)", ok, err)
	}
	ok, _ = verifyForeignPassword(AlgoArgon2id, phc, nil, "wrong")
	if ok {
		t.Fatal("argon2id: wrong password accepted")
	}
}

func TestSHA256Verify(t *testing.T) {
	params := map[string]any{"salt": "saltval", "order": "salt+password", "encoding": "hex"}
	sum := sha256.Sum256([]byte("saltval" + "passw0rd"))
	digest := hex.EncodeToString(sum[:])

	ok, err := verifyForeignPassword(AlgoSHA256, digest, params, "passw0rd")
	if err != nil || !ok {
		t.Fatalf("sha256: correct password rejected (ok=%v err=%v)", ok, err)
	}
	ok, _ = verifyForeignPassword(AlgoSHA256, digest, params, "nope")
	if ok {
		t.Fatal("sha256: wrong password accepted")
	}
}

// Regression tests for the adversarial-review findings.

func TestEmptySignerKeyNeverMatches(t *testing.T) {
	params := map[string]any{"signerKey": "", "salt": "42xEC+ixf3L2lw==", "saltSeparator": "Bw==", "rounds": 8, "memCost": 14}
	// Empty stored hash + empty signer key must not authenticate any password.
	if ok, _ := verifyForeignPassword(AlgoScryptFirebase, "", params, "anything"); ok {
		t.Fatal("empty signerKey/hash matched a password (auth bypass)")
	}
	// Non-empty hash with empty signer key is malformed, never a match.
	if ok, _ := verifyForeignPassword(AlgoScryptFirebase, "AAAA", params, ""); ok {
		t.Fatal("empty signerKey matched (auth bypass)")
	}
}

func TestEmptyHashNeverMatches(t *testing.T) {
	for _, algo := range []string{AlgoPlaintext, AlgoBcrypt, AlgoArgon2id, AlgoSHA256, AlgoScryptFirebase} {
		if ok, _ := verifyForeignPassword(algo, "", nil, ""); ok {
			t.Fatalf("%s: empty stored hash matched empty password (auth bypass)", algo)
		}
	}
}

func TestArgon2EmptyHashSegmentNoPanic(t *testing.T) {
	// Empty final segment previously caused keyLen=0 → panic in the KDF.
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("argon2 verify panicked on malformed hash: %v", r)
		}
	}()
	ok, err := verifyForeignPassword(AlgoArgon2id, "$argon2id$v=19$m=8,t=1,p=1$c29tZXNhbHQ$", nil, "x")
	if ok || err == nil {
		t.Fatalf("malformed argon2 should be (false, error), got ok=%v err=%v", ok, err)
	}
}

func TestArgon2RejectsExcessiveCost(t *testing.T) {
	// m far above the bound must be rejected before the KDF allocates.
	ok, err := verifyForeignPassword(AlgoArgon2id, "$argon2id$v=19$m=999999999,t=100,p=1$c29tZXNhbHQ$aGFzaA", nil, "x")
	if ok || err == nil {
		t.Fatalf("excessive argon2 cost should be rejected, got ok=%v err=%v", ok, err)
	}
}

func TestFirebaseRejectsExcessiveCost(t *testing.T) {
	params := map[string]any{"signerKey": "QUJD", "salt": "42xEC+ixf3L2lw==", "saltSeparator": "Bw==", "rounds": 9999, "memCost": 30}
	if ok, err := verifyForeignPassword(AlgoScryptFirebase, "AAAA", params, "x"); ok || err == nil {
		t.Fatalf("excessive firebase cost should be rejected, got ok=%v err=%v", ok, err)
	}
}

func TestUnsupportedAlgo(t *testing.T) {
	if _, err := verifyForeignPassword("phpass", "x", nil, "y"); err == nil {
		t.Fatal("expected error for unsupported algorithm")
	}
}

func TestPasswordNeedsRehash(t *testing.T) {
	if passwordNeedsRehash("bcrypt") {
		t.Fatal("bcrypt should not need rehash")
	}
	for _, a := range []string{"argon2", "scrypt-firebase", "sha256", "plaintext"} {
		if !passwordNeedsRehash(a) {
			t.Fatalf("%s should need rehash", a)
		}
	}
}
