// Double Ratchet: per-message forward secrecy and post-compromise security
// for an established session, seeded by X3DH's shared secret. Follows
// https://signal.org/docs/specifications/doubleratchet/ — symmetric-key
// ratchet (KDF_CK) advances every message, Diffie-Hellman ratchet (KDF_RK)
// advances whenever the peer starts using a new ratchet key, and out-of-order
// delivery is handled by caching skipped message keys rather than requiring
// in-order arrival.
import { x25519SharedSecret, hkdf, hmacSha256, aesGcmSeal, aesGcmOpen, generateX25519KeyPair, type KeyPair } from './primitives';
import { toBase64, fromBase64, concatBytes, u32be } from './bytes';

const RATCHET_INFO = new TextEncoder().encode('ApplaChatRatchet-v1');
// Bounds how many message keys one DH ratchet step will derive-and-cache
// before giving up, so a forged/corrupt "n" (message index) in a header
// cannot force unbounded work or memory growth.
const MAX_SKIP = 1000;

export interface RatchetState {
  // Fixed for the life of the session — used only to bind ciphertexts to
  // this specific pair of identities (see associatedData below).
  ourIdentityDHPublicKey: Uint8Array;
  theirIdentityDHPublicKey: Uint8Array;

  rootKey: Uint8Array;
  dhSelfPublicKey: Uint8Array;
  dhSelfSecretKey: Uint8Array;
  dhRemotePublicKey: Uint8Array | null;
  chainKeySend: Uint8Array | null;
  chainKeyRecv: Uint8Array | null;
  nSend: number;
  nRecv: number;
  /** Length of the previous sending chain — tells the peer how many
   * messages of the OLD chain to expect at most, so they know how far to
   * skip before ratcheting. */
  pn: number;
  /** Cached message keys for messages that arrived out of order, keyed by
   * "{base64 dhRemotePublicKey}:{n}". */
  skippedMessageKeys: Map<string, Uint8Array>;
}

export interface RatchetHeader {
  dh: string; // base64 sender's current ratchet public key
  pn: number;
  n: number;
}

export interface RatchetMessage {
  header: RatchetHeader;
  ciphertext: string; // base64 AES-256-GCM output (aesGcmSeal's nonce||ct||tag)
}

async function kdfRK(rootKey: Uint8Array, dhOut: Uint8Array): Promise<{ rootKey: Uint8Array; chainKey: Uint8Array }> {
  const out = await hkdf(dhOut, rootKey, RATCHET_INFO, 64);
  return { rootKey: out.slice(0, 32), chainKey: out.slice(32, 64) };
}

async function kdfCK(chainKey: Uint8Array): Promise<{ chainKey: Uint8Array; messageKey: Uint8Array }> {
  // Signal spec: message key from HMAC(chain_key, 0x01), next chain key from
  // HMAC(chain_key, 0x02) — two single-byte-constant HMACs over the same key.
  const [messageKey, nextChainKey] = await Promise.all([
    hmacSha256(chainKey, new Uint8Array([0x01])),
    hmacSha256(chainKey, new Uint8Array([0x02])),
  ]);
  return { chainKey: nextChainKey, messageKey };
}

/** Initializes ratchet state as the party who sends the first message (the
 * X3DH initiator). theirSignedPrekeyPublicKey is the recipient's signed
 * prekey — used as their initial ratchet public key, exactly as the Signal
 * spec's reference X3DH-to-Ratchet handoff does. */
export async function initRatchetAsSender(
  sharedSecret: Uint8Array,
  ourIdentityDHPublicKey: Uint8Array,
  theirIdentityDHPublicKey: Uint8Array,
  theirSignedPrekeyPublicKey: Uint8Array
): Promise<RatchetState> {
  const dh = generateX25519KeyPair();
  const dhOut = x25519SharedSecret(dh.secretKey, theirSignedPrekeyPublicKey);
  const { rootKey, chainKey } = await kdfRK(sharedSecret, dhOut);
  return {
    ourIdentityDHPublicKey,
    theirIdentityDHPublicKey,
    rootKey,
    dhSelfPublicKey: dh.publicKey,
    dhSelfSecretKey: dh.secretKey,
    dhRemotePublicKey: theirSignedPrekeyPublicKey,
    chainKeySend: chainKey,
    chainKeyRecv: null,
    nSend: 0,
    nRecv: 0,
    pn: 0,
    skippedMessageKeys: new Map(),
  };
}

/** Initializes ratchet state as the party who receives the first message.
 * ourSignedPrekey must be the same signed prekey (keyId) the sender's prekey
 * message header names — it doubles as our initial ratchet key pair. */
export function initRatchetAsReceiver(
  sharedSecret: Uint8Array,
  ourIdentityDHPublicKey: Uint8Array,
  theirIdentityDHPublicKey: Uint8Array,
  ourSignedPrekey: KeyPair
): RatchetState {
  return {
    ourIdentityDHPublicKey,
    theirIdentityDHPublicKey,
    rootKey: sharedSecret,
    dhSelfPublicKey: ourSignedPrekey.publicKey,
    dhSelfSecretKey: ourSignedPrekey.secretKey,
    dhRemotePublicKey: null,
    chainKeySend: null,
    chainKeyRecv: null,
    nSend: 0,
    nRecv: 0,
    pn: 0,
    skippedMessageKeys: new Map(),
  };
}

/** Encrypts plaintext, advancing the sending chain by one message. Mutates
 * state. */
export async function ratchetEncrypt(state: RatchetState, plaintext: Uint8Array): Promise<RatchetMessage> {
  if (!state.chainKeySend) {
    throw new Error('chat-crypto: no sending chain established yet (has a message been received to complete the handshake?)');
  }
  const { chainKey, messageKey } = await kdfCK(state.chainKeySend);
  const header: RatchetHeader = { dh: toBase64(state.dhSelfPublicKey), pn: state.pn, n: state.nSend };
  state.chainKeySend = chainKey;
  state.nSend += 1;

  const sealed = await aesGcmSeal(messageKey, plaintext, associatedData(state, header));
  return { header, ciphertext: toBase64(sealed) };
}

/** Decrypts a received message, performing a DH ratchet step first if the
 * message's header carries a new ratchet public key, and deriving (and
 * caching) any skipped message keys along the way so later out-of-order
 * messages still decrypt. Mutates state. Throws on a forged/corrupt header
 * or on AEAD authentication failure — never returns partial or garbage
 * plaintext. */
export async function ratchetDecrypt(state: RatchetState, message: RatchetMessage): Promise<Uint8Array> {
  const theirDh = fromBase64(message.header.dh);

  const cacheKey = skippedKeyId(theirDh, message.header.n);
  const cached = state.skippedMessageKeys.get(cacheKey);
  if (cached) {
    state.skippedMessageKeys.delete(cacheKey);
    return aesGcmOpen(cached, fromBase64(message.ciphertext), associatedData(state, message.header));
  }

  if (!state.dhRemotePublicKey || toBase64(state.dhRemotePublicKey) !== toBase64(theirDh)) {
    await skipMessageKeys(state, message.header.pn);
    await dhRatchetStep(state, theirDh);
  }
  await skipMessageKeys(state, message.header.n);

  if (!state.chainKeyRecv) {
    throw new Error('chat-crypto: no receiving chain established (unexpected message header)');
  }
  const { chainKey, messageKey } = await kdfCK(state.chainKeyRecv);
  state.chainKeyRecv = chainKey;
  state.nRecv += 1;

  return aesGcmOpen(messageKey, fromBase64(message.ciphertext), associatedData(state, message.header));
}

async function skipMessageKeys(state: RatchetState, until: number): Promise<void> {
  if (!state.chainKeyRecv) return; // no chain yet to skip within
  if (until - state.nRecv > MAX_SKIP) {
    throw new Error('chat-crypto: refusing to skip more than ' + MAX_SKIP + ' message keys');
  }
  while (state.nRecv < until) {
    const { chainKey, messageKey } = await kdfCK(state.chainKeyRecv);
    state.skippedMessageKeys.set(skippedKeyId(state.dhRemotePublicKey!, state.nRecv), messageKey);
    state.chainKeyRecv = chainKey;
    state.nRecv += 1;
  }
}

async function dhRatchetStep(state: RatchetState, theirNewDhPublicKey: Uint8Array): Promise<void> {
  state.pn = state.nSend;
  state.nSend = 0;
  state.nRecv = 0;
  state.dhRemotePublicKey = theirNewDhPublicKey;

  const dhOutRecv = x25519SharedSecret(state.dhSelfSecretKey, theirNewDhPublicKey);
  const recv = await kdfRK(state.rootKey, dhOutRecv);
  state.rootKey = recv.rootKey;
  state.chainKeyRecv = recv.chainKey;

  const newDh = generateX25519KeyPair();
  state.dhSelfPublicKey = newDh.publicKey;
  state.dhSelfSecretKey = newDh.secretKey;
  const dhOutSend = x25519SharedSecret(state.dhSelfSecretKey, theirNewDhPublicKey);
  const send = await kdfRK(state.rootKey, dhOutSend);
  state.rootKey = send.rootKey;
  state.chainKeySend = send.chainKey;
}

function skippedKeyId(dhPub: Uint8Array, n: number): string {
  return toBase64(dhPub) + ':' + n;
}

// Binds each ciphertext to this session's identity pair (in a fixed,
// side-independent byte order) plus the header, so a header can't be
// replayed against a different session and its integrity is authenticated
// alongside the ciphertext, not just carried alongside it.
function associatedData(state: RatchetState, header: RatchetHeader): Uint8Array {
  const a = state.ourIdentityDHPublicKey;
  const b = state.theirIdentityDHPublicKey;
  const [first, second] = compareBytes(a, b) <= 0 ? [a, b] : [b, a];
  return concatBytes(first, second, fromBase64(header.dh), u32be(header.pn), u32be(header.n));
}

function compareBytes(a: Uint8Array, b: Uint8Array): number {
  const len = Math.min(a.length, b.length);
  for (let i = 0; i < len; i++) {
    if (a[i] !== b[i]) return a[i] - b[i];
  }
  return a.length - b.length;
}
