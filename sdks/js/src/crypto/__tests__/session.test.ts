// End-to-end session tests: two independent "devices" (Alice, Bob) each with
// their own identity, running the real X3DH handshake and Double Ratchet
// exchange against each other exactly as two real clients would. This is the
// most important test file in the whole chat-crypto layer — it's what
// stands in for cross-implementation test vectors, verifying the two ends
// of the algorithm actually agree, not just that each function looks right
// in isolation.
import {
  generateIdentityKeyPair,
  generateSignedPrekey,
  generateOneTimePrekeys,
} from '../identity';
import { initiateX3DH, receiveX3DH, buildPrekeyMessageHeader, type PrekeyBundle } from '../x3dh';
import {
  initRatchetAsSender,
  initRatchetAsReceiver,
  ratchetEncrypt,
  ratchetDecrypt,
  type RatchetState,
  type RatchetMessage,
} from '../doubleRatchet';
import { utf8ToBytes, bytesToUtf8, toBase64, fromBase64 } from '../bytes';

/** Simulates a device: its identity, one signed prekey, and a small
 * one-time-prekey pool, plus the bundle it would publish to the server. */
function makeDevice() {
  const identity = generateIdentityKeyPair();
  const signedPrekey = generateSignedPrekey(identity, 1);
  const oneTimePrekeys = generateOneTimePrekeys(1, 5);
  return { identity, signedPrekey, oneTimePrekeys };
}

function publishBundle(device: ReturnType<typeof makeDevice>, consumeOneTime: boolean): PrekeyBundle {
  // Peek, don't remove: the test later looks up this exact keyId on the
  // device's own pool to get its private half (as the device itself would
  // still hold it locally) — the server-side "this key is now consumed"
  // bookkeeping isn't what these tests exercise.
  const otp = consumeOneTime ? device.oneTimePrekeys[0] : undefined;
  return {
    deviceId: 'device',
    identityKey: toBase64(concatIdentity(device)),
    signedPrekeyId: device.signedPrekey.keyId,
    signedPrekey: toBase64(device.signedPrekey.keyPair.publicKey),
    signedPrekeySig: toBase64(device.signedPrekey.signature),
    oneTimePrekeyId: otp?.keyId,
    oneTimePrekey: otp ? toBase64(otp.keyPair.publicKey) : undefined,
  };
}

function concatIdentity(device: ReturnType<typeof makeDevice>): Uint8Array {
  const dh = device.identity.dh.publicKey;
  const sign = device.identity.sign.publicKey;
  const out = new Uint8Array(dh.length + sign.length);
  out.set(dh, 0);
  out.set(sign, dh.length);
  return out;
}

/** Runs the full handshake: Alice initiates against Bob's bundle, Bob
 * completes it from the resulting prekey message header. Returns both
 * sides' initialized ratchet state. */
async function establishSession(alice: ReturnType<typeof makeDevice>, bob: ReturnType<typeof makeDevice>, withOneTimePrekey: boolean) {
  const bundle = publishBundle(bob, withOneTimePrekey);
  const x3dh = await initiateX3DH(alice.identity, bundle);
  const header = buildPrekeyMessageHeader(alice.identity, x3dh);

  const aliceState = await initRatchetAsSender(
    x3dh.sharedSecret,
    alice.identity.dh.publicKey,
    x3dh.theirIdentityDHPublicKey,
    bob.signedPrekey.keyPair.publicKey
  );

  // Bob looks up the one-time prekey by id (if the header names one) before
  // it's consumed — this is what the real device keeps locally by keyId.
  const usedOtp = header.oneTimePrekeyId != null
    ? bob.oneTimePrekeys.find((k) => k.keyId === header.oneTimePrekeyId) ?? null
    : null;

  const bobSharedSecret = await receiveX3DH(
    bob.identity,
    bob.signedPrekey.keyPair,
    usedOtp ? usedOtp.keyPair : null,
    alice.identity.dh.publicKey,
    fromBase64(header.ephemeralKey)
  );

  const bobState = initRatchetAsReceiver(
    bobSharedSecret,
    bob.identity.dh.publicKey,
    alice.identity.dh.publicKey,
    bob.signedPrekey.keyPair
  );

  return { aliceState, bobState, header };
}

describe('X3DH handshake', () => {
  it('both sides derive the same shared secret (with a one-time prekey)', async () => {
    const alice = makeDevice();
    const bob = makeDevice();
    const bundle = publishBundle(bob, true);
    const x3dh = await initiateX3DH(alice.identity, bundle);
    const header = buildPrekeyMessageHeader(alice.identity, x3dh);
    const usedOtp = bob.oneTimePrekeys.find((k) => k.keyId === header.oneTimePrekeyId)!;

    const bobSecret = await receiveX3DH(
      bob.identity,
      bob.signedPrekey.keyPair,
      usedOtp.keyPair,
      alice.identity.dh.publicKey,
      x3dh.ephemeralPublicKey
    );

    expect(toBase64(bobSecret)).toBe(toBase64(x3dh.sharedSecret));
  });

  it('both sides derive the same shared secret without a one-time prekey', async () => {
    const alice = makeDevice();
    const bob = makeDevice();
    const bundle = publishBundle(bob, false); // pool not consumed -> bundle has none
    const x3dh = await initiateX3DH(alice.identity, bundle);

    const bobSecret = await receiveX3DH(
      bob.identity,
      bob.signedPrekey.keyPair,
      null,
      alice.identity.dh.publicKey,
      x3dh.ephemeralPublicKey
    );

    expect(toBase64(bobSecret)).toBe(toBase64(x3dh.sharedSecret));
  });

  it('rejects a bundle with an invalid signed-prekey signature', async () => {
    const alice = makeDevice();
    const bob = makeDevice();
    const bundle = publishBundle(bob, false);
    bundle.signedPrekeySig = toBase64(new Uint8Array(64)); // garbage signature
    await expect(initiateX3DH(alice.identity, bundle)).rejects.toThrow(/signature/);
  });
});

describe('Double Ratchet session', () => {
  it('Alice can send and Bob can decrypt the first message', async () => {
    const alice = makeDevice();
    const bob = makeDevice();
    const { aliceState, bobState } = await establishSession(alice, bob, true);

    const msg = await ratchetEncrypt(aliceState, utf8ToBytes('hello bob'));
    const plaintext = await ratchetDecrypt(bobState, msg);
    expect(bytesToUtf8(plaintext)).toBe('hello bob');
  });

  it('supports a full back-and-forth conversation', async () => {
    const alice = makeDevice();
    const bob = makeDevice();
    const { aliceState, bobState } = await establishSession(alice, bob, true);

    const m1 = await ratchetEncrypt(aliceState, utf8ToBytes('hi bob'));
    expect(bytesToUtf8(await ratchetDecrypt(bobState, m1))).toBe('hi bob');

    const m2 = await ratchetEncrypt(bobState, utf8ToBytes('hi alice'));
    expect(bytesToUtf8(await ratchetDecrypt(aliceState, m2))).toBe('hi alice');

    const m3 = await ratchetEncrypt(aliceState, utf8ToBytes('how are you'));
    expect(bytesToUtf8(await ratchetDecrypt(bobState, m3))).toBe('how are you');

    const m4 = await ratchetEncrypt(bobState, utf8ToBytes('good, you?'));
    expect(bytesToUtf8(await ratchetDecrypt(aliceState, m4))).toBe('good, you?');
  });

  it('handles several messages within the same sending chain', async () => {
    const alice = makeDevice();
    const bob = makeDevice();
    const { aliceState, bobState } = await establishSession(alice, bob, true);

    for (let i = 0; i < 5; i++) {
      const msg = await ratchetEncrypt(aliceState, utf8ToBytes(`message ${i}`));
      expect(bytesToUtf8(await ratchetDecrypt(bobState, msg))).toBe(`message ${i}`);
    }
  });

  it('decrypts out-of-order messages within a chain via the skipped-key cache', async () => {
    const alice = makeDevice();
    const bob = makeDevice();
    const { aliceState, bobState } = await establishSession(alice, bob, true);

    const m0 = await ratchetEncrypt(aliceState, utf8ToBytes('zero'));
    const m1 = await ratchetEncrypt(aliceState, utf8ToBytes('one'));
    const m2 = await ratchetEncrypt(aliceState, utf8ToBytes('two'));

    // Bob receives them out of order: 2, 0, 1.
    expect(bytesToUtf8(await ratchetDecrypt(bobState, m2))).toBe('two');
    expect(bytesToUtf8(await ratchetDecrypt(bobState, m0))).toBe('zero');
    expect(bytesToUtf8(await ratchetDecrypt(bobState, m1))).toBe('one');
  });

  it('decrypts messages out of order across a ratchet direction change', async () => {
    const alice = makeDevice();
    const bob = makeDevice();
    const { aliceState, bobState } = await establishSession(alice, bob, true);

    // Alice sends two messages in her first chain.
    const a1 = await ratchetEncrypt(aliceState, utf8ToBytes('a1'));
    const a2 = await ratchetEncrypt(aliceState, utf8ToBytes('a2'));
    // Bob only receives a1 for now (a2 arrives late, tested below).
    expect(bytesToUtf8(await ratchetDecrypt(bobState, a1))).toBe('a1');

    // Bob replies — this is a NEW ratchet key for Bob, so when Alice
    // receives it she'll need to DH-ratchet.
    const b1 = await ratchetEncrypt(bobState, utf8ToBytes('b1'));
    expect(bytesToUtf8(await ratchetDecrypt(aliceState, b1))).toBe('b1');

    // Alice replies again (new chain, since she just ratcheted on receipt of b1).
    const a3 = await ratchetEncrypt(aliceState, utf8ToBytes('a3'));
    expect(bytesToUtf8(await ratchetDecrypt(bobState, a3))).toBe('a3');

    // The late a2 (from Alice's OLD chain, before the ratchet) must still
    // decrypt via Bob's skipped-key cache, recorded when he ratcheted on a3.
    expect(bytesToUtf8(await ratchetDecrypt(bobState, a2))).toBe('a2');
  });

  it('fails closed on a tampered ciphertext', async () => {
    const alice = makeDevice();
    const bob = makeDevice();
    const { aliceState, bobState } = await establishSession(alice, bob, true);

    const msg = await ratchetEncrypt(aliceState, utf8ToBytes('secret'));
    const tampered: RatchetMessage = { header: msg.header, ciphertext: msg.ciphertext.slice(0, -4) + 'AAAA' };
    await expect(ratchetDecrypt(bobState, tampered)).rejects.toThrow();
  });

  it('a third party without the session keys cannot decrypt', async () => {
    const alice = makeDevice();
    const bob = makeDevice();
    const eve = makeDevice();
    const { aliceState } = await establishSession(alice, bob, true);
    // Eve establishes her own (unrelated) session with Bob and tries to
    // decrypt Alice's message using her own state — simulates an outsider
    // who intercepted the ciphertext but has no valid session for it.
    const { bobState: eveViewOfBob } = await establishSession(eve, bob, true);

    const msg = await ratchetEncrypt(aliceState, utf8ToBytes('for bob only'));
    await expect(ratchetDecrypt(eveViewOfBob, msg)).rejects.toThrow();
  });
});
