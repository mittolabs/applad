// Audited cryptographic primitives the chat protocol is built from. Nothing
// in this file implements protocol logic (X3DH/Double Ratchet live in their
// own modules) — it only wraps @noble/curves (X25519 key agreement, Ed25519
// signatures) and the platform's native WebCrypto (AES-GCM, HKDF, HMAC),
// never hand-rolling the underlying math. See the package README for why:
// no maintained, audited libsignal build exists for JS/browser targets, so
// the ratchet state machine is assembled here on top of primitives that
// already are audited, rather than trusting a from-scratch implementation of
// both the math and the protocol.
import { x25519, ed25519 } from '@noble/curves/ed25519';
import { concatBytes } from './bytes';

export interface KeyPair {
  publicKey: Uint8Array;
  secretKey: Uint8Array;
}

// ── X25519 (Diffie-Hellman key agreement) ───────────────────────────────────

export function generateX25519KeyPair(): KeyPair {
  const kp = x25519.keygen();
  return { publicKey: kp.publicKey, secretKey: kp.secretKey };
}

/** Computes the shared X25519 secret between our secret key and their public key. */
export function x25519SharedSecret(secretKey: Uint8Array, theirPublicKey: Uint8Array): Uint8Array {
  return x25519.getSharedSecret(secretKey, theirPublicKey);
}

// ── Ed25519 (signing — used only to sign a device's medium-term signed
// prekey, so a recipient can verify it actually came from the claimed
// identity before using it for X3DH) ────────────────────────────────────────

export function generateEd25519KeyPair(): KeyPair {
  const kp = ed25519.keygen();
  return { publicKey: kp.publicKey, secretKey: kp.secretKey };
}

export function ed25519Sign(message: Uint8Array, secretKey: Uint8Array): Uint8Array {
  return ed25519.sign(message, secretKey);
}

export function ed25519Verify(signature: Uint8Array, message: Uint8Array, publicKey: Uint8Array): boolean {
  return ed25519.verify(signature, message, publicKey);
}

// ── HKDF (RFC 5869) via WebCrypto ────────────────────────────────────────────

/** Derives lengthBytes of key material from ikm using HKDF-SHA256. */
export async function hkdf(
  ikm: Uint8Array,
  salt: Uint8Array,
  info: Uint8Array,
  lengthBytes: number
): Promise<Uint8Array> {
  const key = await globalThis.crypto.subtle.importKey('raw', toArrayBuffer(ikm), 'HKDF', false, ['deriveBits']);
  const bits = await globalThis.crypto.subtle.deriveBits(
    { name: 'HKDF', hash: 'SHA-256', salt: toArrayBuffer(salt), info: toArrayBuffer(info) },
    key,
    lengthBytes * 8
  );
  return new Uint8Array(bits);
}

// ── HMAC-SHA256 via WebCrypto ────────────────────────────────────────────────

export async function hmacSha256(key: Uint8Array, message: Uint8Array): Promise<Uint8Array> {
  const cryptoKey = await globalThis.crypto.subtle.importKey(
    'raw',
    toArrayBuffer(key),
    { name: 'HMAC', hash: 'SHA-256' },
    false,
    ['sign']
  );
  const sig = await globalThis.crypto.subtle.sign('HMAC', cryptoKey, toArrayBuffer(message));
  return new Uint8Array(sig);
}

// ── AES-256-GCM via WebCrypto ────────────────────────────────────────────────

const GCM_NONCE_LENGTH = 12; // bytes
const GCM_TAG_BITS = 128;

/** Seals plaintext with AES-256-GCM under a fresh random nonce, authenticating
 * (but not encrypting) aad. Returns nonce || ciphertext || tag. */
export async function aesGcmSeal(key: Uint8Array, plaintext: Uint8Array, aad: Uint8Array): Promise<Uint8Array> {
  if (key.length !== 32) throw new Error('chat-crypto: AES-256-GCM key must be 32 bytes');
  const nonce = new Uint8Array(GCM_NONCE_LENGTH);
  globalThis.crypto.getRandomValues(nonce);
  const cryptoKey = await globalThis.crypto.subtle.importKey('raw', toArrayBuffer(key), 'AES-GCM', false, ['encrypt']);
  const sealed = await globalThis.crypto.subtle.encrypt(
    { name: 'AES-GCM', iv: toArrayBuffer(nonce), additionalData: toArrayBuffer(aad), tagLength: GCM_TAG_BITS },
    cryptoKey,
    toArrayBuffer(plaintext)
  );
  return concatBytes(nonce, new Uint8Array(sealed));
}

/** Reverses aesGcmSeal. Throws if the tag doesn't verify (tampered ciphertext,
 * wrong key, or mismatched aad). */
export async function aesGcmOpen(key: Uint8Array, sealed: Uint8Array, aad: Uint8Array): Promise<Uint8Array> {
  if (key.length !== 32) throw new Error('chat-crypto: AES-256-GCM key must be 32 bytes');
  if (sealed.length < GCM_NONCE_LENGTH) throw new Error('chat-crypto: sealed payload too short');
  const nonce = sealed.slice(0, GCM_NONCE_LENGTH);
  const ciphertext = sealed.slice(GCM_NONCE_LENGTH);
  const cryptoKey = await globalThis.crypto.subtle.importKey('raw', toArrayBuffer(key), 'AES-GCM', false, ['decrypt']);
  const plaintext = await globalThis.crypto.subtle.decrypt(
    { name: 'AES-GCM', iv: toArrayBuffer(nonce), additionalData: toArrayBuffer(aad), tagLength: GCM_TAG_BITS },
    cryptoKey,
    toArrayBuffer(ciphertext)
  );
  return new Uint8Array(plaintext);
}

// WebCrypto wants an ArrayBuffer, not a Uint8Array view that might be backed
// by a larger buffer (e.g. a slice) — copy defensively.
function toArrayBuffer(bytes: Uint8Array): ArrayBuffer {
  return bytes.buffer.slice(bytes.byteOffset, bytes.byteOffset + bytes.byteLength) as ArrayBuffer;
}
