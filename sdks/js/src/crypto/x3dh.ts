// X3DH (Extended Triple Diffie-Hellman) session establishment: the handshake
// that lets one device start an encrypted session with another using only
// that device's published prekey bundle — no round trip required before the
// first message. Output is a 32-byte shared secret that seeds a Double
// Ratchet session (see doubleRatchet.ts). Follows the algorithm from
// https://signal.org/docs/specifications/x3dh/.
import { x25519SharedSecret, hkdf, generateX25519KeyPair, type KeyPair } from './primitives';
import { decodeIdentityPublicKey, verifySignedPrekey, type IdentityKeyPair } from './identity';
import { toBase64, fromBase64, concatBytes, utf8ToBytes } from './bytes';

const X3DH_SALT = new Uint8Array(32); // all-zero; the info string below provides domain separation
const X3DH_INFO = utf8ToBytes('ApplaChatX3DH-v1');

/** A recipient device's public prekey bundle, as fetched from
 * GET /chat/devices/{deviceId}/prekey-bundle. */
export interface PrekeyBundle {
  deviceId: string;
  identityKey: string;
  signedPrekeyId: number;
  signedPrekey: string;
  signedPrekeySig: string;
  oneTimePrekeyId?: number;
  oneTimePrekey?: string;
}

export interface X3DHInitiatorResult {
  sharedSecret: Uint8Array;
  ephemeralPublicKey: Uint8Array;
  usedSignedPrekeyId: number;
  usedOneTimePrekeyId?: number;
  theirIdentityDHPublicKey: Uint8Array;
}

/** Initiates a session with another device from its published bundle (the
 * "Alice" role in X3DH — the side sending the first message). Throws if the
 * bundle's signed prekey signature doesn't verify against its claimed
 * identity, since trusting an unverified prekey would let a
 * man-in-the-middle substitute their own key. */
export async function initiateX3DH(ourIdentity: IdentityKeyPair, bundle: PrekeyBundle): Promise<X3DHInitiatorResult> {
  const theirIdentity = decodeIdentityPublicKey(bundle.identityKey);
  const theirSignedPrekey = fromBase64(bundle.signedPrekey);
  const signature = fromBase64(bundle.signedPrekeySig);
  if (!verifySignedPrekey(theirIdentity.sign, theirSignedPrekey, signature)) {
    throw new Error('chat-crypto: signed prekey signature verification failed — refusing to establish a session');
  }

  const ephemeral: KeyPair = generateX25519KeyPair();

  const dh1 = x25519SharedSecret(ourIdentity.dh.secretKey, theirSignedPrekey);
  const dh2 = x25519SharedSecret(ephemeral.secretKey, theirIdentity.dh);
  const dh3 = x25519SharedSecret(ephemeral.secretKey, theirSignedPrekey);
  const parts = [dh1, dh2, dh3];
  if (bundle.oneTimePrekey) {
    parts.push(x25519SharedSecret(ephemeral.secretKey, fromBase64(bundle.oneTimePrekey)));
  }

  const sharedSecret = await hkdf(concatBytes(...parts), X3DH_SALT, X3DH_INFO, 32);

  return {
    sharedSecret,
    ephemeralPublicKey: ephemeral.publicKey,
    usedSignedPrekeyId: bundle.signedPrekeyId,
    usedOneTimePrekeyId: bundle.oneTimePrekeyId,
    theirIdentityDHPublicKey: theirIdentity.dh,
  };
}

/** The "prekey message" header an initiator sends alongside their first
 * Double Ratchet message, carrying everything the recipient needs to derive
 * the same shared secret. Server-facing envelope_type: "prekey". */
export interface PrekeyMessageHeader {
  identityKey: string; // sender's encoded identity (dh + sign public halves)
  ephemeralKey: string; // base64 X25519 public key
  signedPrekeyId: number;
  oneTimePrekeyId?: number;
}

export function buildPrekeyMessageHeader(ourIdentity: IdentityKeyPair, result: X3DHInitiatorResult): PrekeyMessageHeader {
  return {
    identityKey: toBase64(concatBytes(ourIdentity.dh.publicKey, ourIdentity.sign.publicKey)),
    ephemeralKey: toBase64(result.ephemeralPublicKey),
    signedPrekeyId: result.usedSignedPrekeyId,
    oneTimePrekeyId: result.usedOneTimePrekeyId,
  };
}

/** Completes the handshake on the receiving ("Bob") side: given the sender's
 * identity and ephemeral public keys from a prekey message header, and our
 * own matching private key material, derives the same shared secret the
 * initiator did. ourOneTimePrekey must be the specific key the header names
 * (null if the header carries none — X3DH degrades gracefully without one,
 * per spec, at the cost of one fewer DH term). */
export async function receiveX3DH(
  ourIdentity: IdentityKeyPair,
  ourSignedPrekey: KeyPair,
  ourOneTimePrekey: KeyPair | null,
  theirIdentityDHPublicKey: Uint8Array,
  theirEphemeralPublicKey: Uint8Array
): Promise<Uint8Array> {
  const dh1 = x25519SharedSecret(ourSignedPrekey.secretKey, theirIdentityDHPublicKey);
  const dh2 = x25519SharedSecret(ourIdentity.dh.secretKey, theirEphemeralPublicKey);
  const dh3 = x25519SharedSecret(ourSignedPrekey.secretKey, theirEphemeralPublicKey);
  const parts = [dh1, dh2, dh3];
  if (ourOneTimePrekey) {
    parts.push(x25519SharedSecret(ourOneTimePrekey.secretKey, theirEphemeralPublicKey));
  }
  return hkdf(concatBytes(...parts), X3DH_SALT, X3DH_INFO, 32);
}
