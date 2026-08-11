import {
  generateX25519KeyPair,
  x25519SharedSecret,
  generateEd25519KeyPair,
  ed25519Sign,
  ed25519Verify,
  hkdf,
  hmacSha256,
  aesGcmSeal,
  aesGcmOpen,
} from '../primitives';
import { utf8ToBytes, bytesToUtf8, toBase64, fromBase64, randomBytes } from '../bytes';

describe('X25519', () => {
  it('two parties derive the same shared secret', () => {
    const alice = generateX25519KeyPair();
    const bob = generateX25519KeyPair();
    const aliceShared = x25519SharedSecret(alice.secretKey, bob.publicKey);
    const bobShared = x25519SharedSecret(bob.secretKey, alice.publicKey);
    expect(toBase64(aliceShared)).toBe(toBase64(bobShared));
  });

  it('produces 32-byte keys', () => {
    const kp = generateX25519KeyPair();
    expect(kp.publicKey.length).toBe(32);
    expect(kp.secretKey.length).toBe(32);
  });
});

describe('Ed25519', () => {
  it('verifies a signature made with the matching secret key', () => {
    const kp = generateEd25519KeyPair();
    const msg = utf8ToBytes('signed prekey bytes');
    const sig = ed25519Sign(msg, kp.secretKey);
    expect(ed25519Verify(sig, msg, kp.publicKey)).toBe(true);
  });

  it('rejects a signature from the wrong key', () => {
    const kp = generateEd25519KeyPair();
    const other = generateEd25519KeyPair();
    const msg = utf8ToBytes('signed prekey bytes');
    const sig = ed25519Sign(msg, kp.secretKey);
    expect(ed25519Verify(sig, msg, other.publicKey)).toBe(false);
  });

  it('rejects a tampered message', () => {
    const kp = generateEd25519KeyPair();
    const sig = ed25519Sign(utf8ToBytes('original'), kp.secretKey);
    expect(ed25519Verify(sig, utf8ToBytes('tampered'), kp.publicKey)).toBe(false);
  });
});

describe('HKDF', () => {
  it('is deterministic for the same inputs', async () => {
    const ikm = randomBytes(32);
    const salt = randomBytes(32);
    const info = utf8ToBytes('test-info');
    const a = await hkdf(ikm, salt, info, 64);
    const b = await hkdf(ikm, salt, info, 64);
    expect(toBase64(a)).toBe(toBase64(b));
    expect(a.length).toBe(64);
  });

  it('produces different output for different info (domain separation)', async () => {
    const ikm = randomBytes(32);
    const salt = randomBytes(32);
    const a = await hkdf(ikm, salt, utf8ToBytes('info-a'), 32);
    const b = await hkdf(ikm, salt, utf8ToBytes('info-b'), 32);
    expect(toBase64(a)).not.toBe(toBase64(b));
  });
});

describe('HMAC-SHA256', () => {
  it('is deterministic and 32 bytes', async () => {
    const key = randomBytes(32);
    const msg = utf8ToBytes('chain key advance');
    const a = await hmacSha256(key, msg);
    const b = await hmacSha256(key, msg);
    expect(a.length).toBe(32);
    expect(toBase64(a)).toBe(toBase64(b));
  });
});

describe('AES-256-GCM', () => {
  it('round-trips plaintext', async () => {
    const key = randomBytes(32);
    const aad = utf8ToBytes('header-bytes');
    const plaintext = utf8ToBytes('hello, this is a secret message');
    const sealed = await aesGcmSeal(key, plaintext, aad);
    const opened = await aesGcmOpen(key, sealed, aad);
    expect(bytesToUtf8(opened)).toBe('hello, this is a secret message');
  });

  it('produces ciphertext that does not contain the plaintext', async () => {
    const key = randomBytes(32);
    const plaintext = utf8ToBytes('do not leak this');
    const sealed = await aesGcmSeal(key, plaintext, new Uint8Array(0));
    expect(toBase64(sealed)).not.toContain('do not leak this');
  });

  it('produces different ciphertext each time (random nonce)', async () => {
    const key = randomBytes(32);
    const plaintext = utf8ToBytes('same plaintext');
    const a = await aesGcmSeal(key, plaintext, new Uint8Array(0));
    const b = await aesGcmSeal(key, plaintext, new Uint8Array(0));
    expect(toBase64(a)).not.toBe(toBase64(b));
  });

  it('fails to open with the wrong key', async () => {
    const key = randomBytes(32);
    const wrongKey = randomBytes(32);
    const sealed = await aesGcmSeal(key, utf8ToBytes('secret'), new Uint8Array(0));
    await expect(aesGcmOpen(wrongKey, sealed, new Uint8Array(0))).rejects.toThrow();
  });

  it('fails to open with mismatched aad (header tampering detected)', async () => {
    const key = randomBytes(32);
    const sealed = await aesGcmSeal(key, utf8ToBytes('secret'), utf8ToBytes('header-a'));
    await expect(aesGcmOpen(key, sealed, utf8ToBytes('header-b'))).rejects.toThrow();
  });

  it('fails to open tampered ciphertext', async () => {
    const key = randomBytes(32);
    const sealed = await aesGcmSeal(key, utf8ToBytes('secret'), new Uint8Array(0));
    const tampered = new Uint8Array(sealed);
    tampered[tampered.length - 1] ^= 0xff;
    await expect(aesGcmOpen(key, tampered, new Uint8Array(0))).rejects.toThrow();
  });
});

describe('base64 round trip', () => {
  it('round-trips arbitrary bytes', () => {
    const bytes = randomBytes(64);
    expect(toBase64(fromBase64(toBase64(bytes)))).toBe(toBase64(bytes));
  });
});
