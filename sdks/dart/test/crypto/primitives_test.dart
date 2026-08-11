// NOTE: written without a Dart toolchain available to run it — mirrors
// sdks/js/src/crypto/__tests__/primitives.test.ts, which IS verified. Run
// `melos test` (or `dart test` from sdks/dart/) before trusting this file.
import 'dart:typed_data';

import 'package:test/test.dart';
import 'package:applad/src/crypto/crypto.dart';

void main() {
  group('X25519', () {
    test('two parties derive the same shared secret', () async {
      final alice = await generateX25519KeyPair();
      final bob = await generateX25519KeyPair();
      final aliceShared = await x25519SharedSecret(alice, bob.publicKey);
      final bobShared = await x25519SharedSecret(bob, alice.publicKey);
      expect(toBase64(aliceShared), equals(toBase64(bobShared)));
    });

    test('produces 32-byte keys', () async {
      final kp = await generateX25519KeyPair();
      expect(kp.publicKey.length, equals(32));
      expect(kp.secretKey.length, equals(32));
    });
  });

  group('Ed25519', () {
    test('verifies a signature made with the matching secret key', () async {
      final kp = await generateEd25519KeyPair();
      final msg = utf8ToBytes('signed prekey bytes');
      final sig = await ed25519Sign(msg, kp);
      expect(await ed25519Verify(sig, msg, kp.publicKey), isTrue);
    });

    test('rejects a signature from the wrong key', () async {
      final kp = await generateEd25519KeyPair();
      final other = await generateEd25519KeyPair();
      final msg = utf8ToBytes('signed prekey bytes');
      final sig = await ed25519Sign(msg, kp);
      expect(await ed25519Verify(sig, msg, other.publicKey), isFalse);
    });

    test('rejects a tampered message', () async {
      final kp = await generateEd25519KeyPair();
      final sig = await ed25519Sign(utf8ToBytes('original'), kp);
      expect(await ed25519Verify(sig, utf8ToBytes('tampered'), kp.publicKey), isFalse);
    });
  });

  group('HKDF', () {
    test('is deterministic for the same inputs', () async {
      final ikm = randomBytes(32);
      final salt = randomBytes(32);
      final info = utf8ToBytes('test-info');
      final a = await hkdf(ikm, salt, info, 64);
      final b = await hkdf(ikm, salt, info, 64);
      expect(toBase64(a), equals(toBase64(b)));
      expect(a.length, equals(64));
    });

    test('produces different output for different info (domain separation)', () async {
      final ikm = randomBytes(32);
      final salt = randomBytes(32);
      final a = await hkdf(ikm, salt, utf8ToBytes('info-a'), 32);
      final b = await hkdf(ikm, salt, utf8ToBytes('info-b'), 32);
      expect(toBase64(a), isNot(equals(toBase64(b))));
    });
  });

  group('HMAC-SHA256', () {
    test('is deterministic and 32 bytes', () async {
      final key = randomBytes(32);
      final msg = utf8ToBytes('chain key advance');
      final a = await hmacSha256(key, msg);
      final b = await hmacSha256(key, msg);
      expect(a.length, equals(32));
      expect(toBase64(a), equals(toBase64(b)));
    });
  });

  group('AES-256-GCM', () {
    test('round-trips plaintext', () async {
      final key = randomBytes(32);
      final aad = utf8ToBytes('header-bytes');
      final plaintext = utf8ToBytes('hello, this is a secret message');
      final sealed = await aesGcmSeal(key, plaintext, aad);
      final opened = await aesGcmOpen(key, sealed, aad);
      expect(bytesToUtf8(opened), equals('hello, this is a secret message'));
    });

    test('produces ciphertext that does not contain the plaintext', () async {
      final key = randomBytes(32);
      final plaintext = utf8ToBytes('do not leak this');
      final sealed = await aesGcmSeal(key, plaintext, Uint8List(0));
      expect(toBase64(sealed), isNot(contains('do not leak this')));
    });

    test('produces different ciphertext each time (random nonce)', () async {
      final key = randomBytes(32);
      final plaintext = utf8ToBytes('same plaintext');
      final a = await aesGcmSeal(key, plaintext, Uint8List(0));
      final b = await aesGcmSeal(key, plaintext, Uint8List(0));
      expect(toBase64(a), isNot(equals(toBase64(b))));
    });

    test('fails to open with the wrong key', () async {
      final key = randomBytes(32);
      final wrongKey = randomBytes(32);
      final sealed = await aesGcmSeal(key, utf8ToBytes('secret'), Uint8List(0));
      expect(() => aesGcmOpen(wrongKey, sealed, Uint8List(0)), throwsA(anything));
    });

    test('fails to open with mismatched aad (header tampering detected)', () async {
      final key = randomBytes(32);
      final sealed = await aesGcmSeal(key, utf8ToBytes('secret'), utf8ToBytes('header-a'));
      expect(() => aesGcmOpen(key, sealed, utf8ToBytes('header-b')), throwsA(anything));
    });

    test('fails to open tampered ciphertext', () async {
      final key = randomBytes(32);
      final sealed = await aesGcmSeal(key, utf8ToBytes('secret'), Uint8List(0));
      final tampered = Uint8List.fromList(sealed);
      tampered[tampered.length - 1] ^= 0xff;
      expect(() => aesGcmOpen(key, tampered, Uint8List(0)), throwsA(anything));
    });
  });

  group('base64 round trip', () {
    test('round-trips arbitrary bytes', () {
      final bytes = randomBytes(64);
      expect(toBase64(fromBase64(toBase64(bytes))), equals(toBase64(bytes)));
    });
  });
}
