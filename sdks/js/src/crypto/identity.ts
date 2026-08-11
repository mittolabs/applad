// Device identity: the long-term key material a device generates once and
// publishes (public halves only) so other devices can start an encrypted
// session with it. Two separate keypairs stand in for Signal's single
// XEdDSA identity key (which reuses one Curve25519 key for both signing and
// DH via a birational map): an X25519 pair for Diffie-Hellman agreement and
// an Ed25519 pair for signing the signed prekey. This is a deliberate
// simplification — one extra public key in the bundle, in exchange for using
// each curve the way @noble/curves exposes it directly, with no custom
// key-format conversion math to get right.
import {
  generateX25519KeyPair,
  generateEd25519KeyPair,
  ed25519Sign,
  ed25519Verify,
  type KeyPair,
} from './primitives';
import { toBase64, fromBase64, concatBytes } from './bytes';

export interface IdentityKeyPair {
  dh: KeyPair;
  sign: KeyPair;
}

export interface SignedPrekey {
  keyId: number;
  keyPair: KeyPair;
  signature: Uint8Array;
}

export interface OneTimePrekey {
  keyId: number;
  keyPair: KeyPair;
}

/** Generates a new device identity. Call this once per device and persist
 * the private halves in platform secure storage — they must never leave
 * the device or be sent to the server. */
export function generateIdentityKeyPair(): IdentityKeyPair {
  return { dh: generateX25519KeyPair(), sign: generateEd25519KeyPair() };
}

/** Generates a new signed prekey, rotated periodically (recommend every 7-30
 * days) and re-published via the device's registration endpoint. */
export function generateSignedPrekey(identity: IdentityKeyPair, keyId: number): SignedPrekey {
  const keyPair = generateX25519KeyPair();
  const signature = ed25519Sign(keyPair.publicKey, identity.sign.secretKey);
  return { keyId, keyPair, signature };
}

/** Generates a batch of one-time prekeys with sequential ids starting at
 * startId, to upload alongside registration or a top-up. */
export function generateOneTimePrekeys(startId: number, count: number): OneTimePrekey[] {
  const out: OneTimePrekey[] = [];
  for (let i = 0; i < count; i++) {
    out.push({ keyId: startId + i, keyPair: generateX25519KeyPair() });
  }
  return out;
}

/** Verifies a signed prekey's signature against the claimed identity's
 * signing key, before it's trusted for X3DH. */
export function verifySignedPrekey(identityPublicSign: Uint8Array, signedPrekeyPublic: Uint8Array, signature: Uint8Array): boolean {
  return ed25519Verify(signature, signedPrekeyPublic, identityPublicSign);
}

// ── Wire encoding ────────────────────────────────────────────────────────────
//
// The backend's `identityKey` field is an opaque string as far as the server
// is concerned — it never decodes it. The client packs both public halves
// (DH + sign, 32 bytes each) into one base64 blob so no schema/migration
// change was needed to add the signing key.

export function encodeIdentityPublicKey(identity: IdentityKeyPair): string {
  return toBase64(concatBytes(identity.dh.publicKey, identity.sign.publicKey));
}

export function decodeIdentityPublicKey(encoded: string): { dh: Uint8Array; sign: Uint8Array } {
  const bytes = fromBase64(encoded);
  if (bytes.length !== 64) {
    throw new Error('chat-crypto: malformed identity key (expected 64 bytes: dh(32) + sign(32))');
  }
  return { dh: bytes.slice(0, 32), sign: bytes.slice(32, 64) };
}
