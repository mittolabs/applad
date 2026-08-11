// NOTE: written without a Dart toolchain available to run it — mirrors
// sdks/js/src/crypto/__tests__/session.test.ts, which IS verified (25
// passing tests covering X3DH agreement, bidirectional messaging, in-order
// and out-of-order delivery across ratchet direction changes, tamper
// detection, and third-party isolation). Run `melos test` (or `dart test`
// from sdks/dart/) before trusting this port.
//
// Two independent "devices" (Alice, Bob) each with their own identity,
// running the real X3DH handshake and Double Ratchet exchange against each
// other exactly as two real clients would — this is what stands in for
// cross-implementation test vectors here, verifying the two ends of the
// algorithm actually agree, not just that each function looks right alone.
import 'dart:typed_data';

import 'package:test/test.dart';
import 'package:applad/src/crypto/crypto.dart';

class _Device {
  final IdentityKeyPair identity;
  final SignedPrekey signedPrekey;
  final List<OneTimePrekey> oneTimePrekeys;
  _Device({required this.identity, required this.signedPrekey, required this.oneTimePrekeys});
}

Future<_Device> _makeDevice() async {
  final identity = await generateIdentityKeyPair();
  final signedPrekey = await generateSignedPrekey(identity, 1);
  final oneTimePrekeys = await generateOneTimePrekeys(1, 5);
  return _Device(identity: identity, signedPrekey: signedPrekey, oneTimePrekeys: oneTimePrekeys);
}

/// Peek, don't remove: the test later looks up this exact keyId on the
/// device's own pool to get its private half (as the device itself would
/// still hold it locally) — the server-side "this key is now consumed"
/// bookkeeping isn't what these tests exercise.
PrekeyBundle _publishBundle(_Device device, bool consumeOneTime) {
  final otp = consumeOneTime && device.oneTimePrekeys.isNotEmpty ? device.oneTimePrekeys.first : null;
  return PrekeyBundle(
    deviceId: 'device',
    identityKey: encodeIdentityPublicKey(device.identity),
    signedPrekeyId: device.signedPrekey.keyId,
    signedPrekey: toBase64(device.signedPrekey.keyPair.publicKey),
    signedPrekeySig: toBase64(device.signedPrekey.signature),
    oneTimePrekeyId: otp?.keyId,
    oneTimePrekey: otp != null ? toBase64(otp.keyPair.publicKey) : null,
  );
}

class _Session {
  final RatchetState aliceState;
  final RatchetState bobState;
  final PrekeyMessageHeader header;
  _Session({required this.aliceState, required this.bobState, required this.header});
}

Future<_Session> _establishSession(_Device alice, _Device bob, bool withOneTimePrekey) async {
  final bundle = _publishBundle(bob, withOneTimePrekey);
  final x3dh = await initiateX3DH(alice.identity, bundle);
  final header = buildPrekeyMessageHeader(alice.identity, x3dh);

  final aliceState = await initRatchetAsSender(
    x3dh.sharedSecret,
    alice.identity.dh.publicKey,
    x3dh.theirIdentityDHPublicKey,
    bob.signedPrekey.keyPair.publicKey,
  );

  OneTimePrekey? usedOtp;
  if (header.oneTimePrekeyId != null) {
    for (final k in bob.oneTimePrekeys) {
      if (k.keyId == header.oneTimePrekeyId) {
        usedOtp = k;
        break;
      }
    }
  }

  final bobSharedSecret = await receiveX3DH(
    bob.identity,
    bob.signedPrekey.keyPair,
    usedOtp?.keyPair,
    alice.identity.dh.publicKey,
    fromBase64(header.ephemeralKey),
  );

  final bobState = initRatchetAsReceiver(
    bobSharedSecret,
    bob.identity.dh.publicKey,
    alice.identity.dh.publicKey,
    bob.signedPrekey.keyPair,
  );

  return _Session(aliceState: aliceState, bobState: bobState, header: header);
}

void main() {
  group('X3DH handshake', () {
    test('both sides derive the same shared secret (with a one-time prekey)', () async {
      final alice = await _makeDevice();
      final bob = await _makeDevice();
      final bundle = _publishBundle(bob, true);
      final x3dh = await initiateX3DH(alice.identity, bundle);
      final header = buildPrekeyMessageHeader(alice.identity, x3dh);
      final usedOtp = bob.oneTimePrekeys.firstWhere((k) => k.keyId == header.oneTimePrekeyId);

      final bobSecret = await receiveX3DH(
        bob.identity,
        bob.signedPrekey.keyPair,
        usedOtp.keyPair,
        alice.identity.dh.publicKey,
        x3dh.ephemeralPublicKey,
      );

      expect(toBase64(bobSecret), equals(toBase64(x3dh.sharedSecret)));
    });

    test('both sides derive the same shared secret without a one-time prekey', () async {
      final alice = await _makeDevice();
      final bob = await _makeDevice();
      final bundle = _publishBundle(bob, false);
      final x3dh = await initiateX3DH(alice.identity, bundle);

      final bobSecret = await receiveX3DH(
        bob.identity,
        bob.signedPrekey.keyPair,
        null,
        alice.identity.dh.publicKey,
        x3dh.ephemeralPublicKey,
      );

      expect(toBase64(bobSecret), equals(toBase64(x3dh.sharedSecret)));
    });

    test('rejects a bundle with an invalid signed-prekey signature', () async {
      final alice = await _makeDevice();
      final bob = await _makeDevice();
      final bundle = _publishBundle(bob, false);
      final tampered = PrekeyBundle(
        deviceId: bundle.deviceId,
        identityKey: bundle.identityKey,
        signedPrekeyId: bundle.signedPrekeyId,
        signedPrekey: bundle.signedPrekey,
        signedPrekeySig: toBase64(Uint8List(64)), // garbage signature
      );
      expect(() => initiateX3DH(alice.identity, tampered), throwsA(anything));
    });
  });

  group('Double Ratchet session', () {
    test('Alice can send and Bob can decrypt the first message', () async {
      final alice = await _makeDevice();
      final bob = await _makeDevice();
      final session = await _establishSession(alice, bob, true);

      final msg = await ratchetEncrypt(session.aliceState, utf8ToBytes('hello bob'));
      final plaintext = await ratchetDecrypt(session.bobState, msg);
      expect(bytesToUtf8(plaintext), equals('hello bob'));
    });

    test('supports a full back-and-forth conversation', () async {
      final alice = await _makeDevice();
      final bob = await _makeDevice();
      final session = await _establishSession(alice, bob, true);

      final m1 = await ratchetEncrypt(session.aliceState, utf8ToBytes('hi bob'));
      expect(bytesToUtf8(await ratchetDecrypt(session.bobState, m1)), equals('hi bob'));

      final m2 = await ratchetEncrypt(session.bobState, utf8ToBytes('hi alice'));
      expect(bytesToUtf8(await ratchetDecrypt(session.aliceState, m2)), equals('hi alice'));

      final m3 = await ratchetEncrypt(session.aliceState, utf8ToBytes('how are you'));
      expect(bytesToUtf8(await ratchetDecrypt(session.bobState, m3)), equals('how are you'));

      final m4 = await ratchetEncrypt(session.bobState, utf8ToBytes('good, you?'));
      expect(bytesToUtf8(await ratchetDecrypt(session.aliceState, m4)), equals('good, you?'));
    });

    test('handles several messages within the same sending chain', () async {
      final alice = await _makeDevice();
      final bob = await _makeDevice();
      final session = await _establishSession(alice, bob, true);

      for (var i = 0; i < 5; i++) {
        final msg = await ratchetEncrypt(session.aliceState, utf8ToBytes('message $i'));
        expect(bytesToUtf8(await ratchetDecrypt(session.bobState, msg)), equals('message $i'));
      }
    });

    test('decrypts out-of-order messages within a chain via the skipped-key cache', () async {
      final alice = await _makeDevice();
      final bob = await _makeDevice();
      final session = await _establishSession(alice, bob, true);

      final m0 = await ratchetEncrypt(session.aliceState, utf8ToBytes('zero'));
      final m1 = await ratchetEncrypt(session.aliceState, utf8ToBytes('one'));
      final m2 = await ratchetEncrypt(session.aliceState, utf8ToBytes('two'));

      expect(bytesToUtf8(await ratchetDecrypt(session.bobState, m2)), equals('two'));
      expect(bytesToUtf8(await ratchetDecrypt(session.bobState, m0)), equals('zero'));
      expect(bytesToUtf8(await ratchetDecrypt(session.bobState, m1)), equals('one'));
    });

    test('decrypts messages out of order across a ratchet direction change', () async {
      final alice = await _makeDevice();
      final bob = await _makeDevice();
      final session = await _establishSession(alice, bob, true);

      final a1 = await ratchetEncrypt(session.aliceState, utf8ToBytes('a1'));
      final a2 = await ratchetEncrypt(session.aliceState, utf8ToBytes('a2'));
      expect(bytesToUtf8(await ratchetDecrypt(session.bobState, a1)), equals('a1'));

      final b1 = await ratchetEncrypt(session.bobState, utf8ToBytes('b1'));
      expect(bytesToUtf8(await ratchetDecrypt(session.aliceState, b1)), equals('b1'));

      final a3 = await ratchetEncrypt(session.aliceState, utf8ToBytes('a3'));
      expect(bytesToUtf8(await ratchetDecrypt(session.bobState, a3)), equals('a3'));

      // The late a2 (from Alice's OLD chain, before the ratchet) must still
      // decrypt via Bob's skipped-key cache, recorded when he ratcheted on a3.
      expect(bytesToUtf8(await ratchetDecrypt(session.bobState, a2)), equals('a2'));
    });

    test('fails closed on a tampered ciphertext', () async {
      final alice = await _makeDevice();
      final bob = await _makeDevice();
      final session = await _establishSession(alice, bob, true);

      final msg = await ratchetEncrypt(session.aliceState, utf8ToBytes('secret'));
      final tampered = RatchetMessage(
        header: msg.header,
        ciphertext: '${msg.ciphertext.substring(0, msg.ciphertext.length - 4)}AAAA',
      );
      expect(() => ratchetDecrypt(session.bobState, tampered), throwsA(anything));
    });

    test('a third party without the session keys cannot decrypt', () async {
      final alice = await _makeDevice();
      final bob = await _makeDevice();
      final eve = await _makeDevice();
      final session = await _establishSession(alice, bob, true);
      final eveSession = await _establishSession(eve, bob, true);

      final msg = await ratchetEncrypt(session.aliceState, utf8ToBytes('for bob only'));
      expect(() => ratchetDecrypt(eveSession.bobState, msg), throwsA(anything));
    });
  });
}
