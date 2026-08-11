// Audited cryptographic primitives the chat protocol is built from. Nothing
// in this file implements protocol logic (X3DH/Double Ratchet live in their
// own files) — it only wraps the `cryptography` package's X25519 key
// agreement, Ed25519 signatures, HKDF, HMAC, and AES-256-GCM, never
// hand-rolling the underlying math. See pubspec.yaml for why: no maintained
// Dart FFI binding to libsignal-client exists, so the ratchet state machine
// is assembled here on top of primitives that are already audited, rather
// than trusting a from-scratch implementation of both the math and the
// protocol.
//
// NOTE: unlike the rest of this SDK, this file has not been compiled or run
// — the development environment that wrote it has no Dart/Flutter toolchain.
// Run `melos bootstrap && melos analyze && melos test` before trusting it.
import 'dart:typed_data';
import 'package:cryptography/cryptography.dart';

import 'bytes.dart';

class KeyPairBytes {
  final Uint8List publicKey;
  final Uint8List secretKey;
  const KeyPairBytes({required this.publicKey, required this.secretKey});
}

final X25519 _x25519 = X25519();
final Ed25519 _ed25519 = Ed25519();

// ── X25519 (Diffie-Hellman key agreement) ───────────────────────────────────

Future<KeyPairBytes> generateX25519KeyPair() async {
  final keyPair = await _x25519.newKeyPair();
  final secretKey = await keyPair.extractPrivateKeyBytes();
  final publicKey = await keyPair.extractPublicKey();
  return KeyPairBytes(publicKey: Uint8List.fromList(publicKey.bytes), secretKey: Uint8List.fromList(secretKey));
}

/// Computes the shared X25519 secret between our key pair and their public key.
Future<Uint8List> x25519SharedSecret(KeyPairBytes ourKeyPair, Uint8List theirPublicKey) async {
  final keyPair = SimpleKeyPairData(
    ourKeyPair.secretKey,
    publicKey: SimplePublicKey(ourKeyPair.publicKey, type: KeyPairType.x25519),
    type: KeyPairType.x25519,
  );
  final shared = await _x25519.sharedSecretKey(
    keyPair: keyPair,
    remotePublicKey: SimplePublicKey(theirPublicKey, type: KeyPairType.x25519),
  );
  return Uint8List.fromList(await shared.extractBytes());
}

// ── Ed25519 (signing — used only to sign a device's medium-term signed
// prekey, so a recipient can verify it actually came from the claimed
// identity before using it for X3DH) ────────────────────────────────────────

Future<KeyPairBytes> generateEd25519KeyPair() async {
  final keyPair = await _ed25519.newKeyPair();
  final secretKey = await keyPair.extractPrivateKeyBytes();
  final publicKey = await keyPair.extractPublicKey();
  return KeyPairBytes(publicKey: Uint8List.fromList(publicKey.bytes), secretKey: Uint8List.fromList(secretKey));
}

Future<Uint8List> ed25519Sign(Uint8List message, KeyPairBytes signingKeyPair) async {
  final keyPair = SimpleKeyPairData(
    signingKeyPair.secretKey,
    publicKey: SimplePublicKey(signingKeyPair.publicKey, type: KeyPairType.ed25519),
    type: KeyPairType.ed25519,
  );
  final signature = await _ed25519.sign(message, keyPair: keyPair);
  return Uint8List.fromList(signature.bytes);
}

Future<bool> ed25519Verify(Uint8List signature, Uint8List message, Uint8List publicKey) async {
  final sig = Signature(signature, publicKey: SimplePublicKey(publicKey, type: KeyPairType.ed25519));
  return _ed25519.verify(message, signature: sig);
}

// ── HKDF (RFC 5869) ──────────────────────────────────────────────────────────

/// Derives lengthBytes of key material from ikm using HKDF-SHA256.
Future<Uint8List> hkdf(Uint8List ikm, Uint8List salt, Uint8List info, int lengthBytes) async {
  final algorithm = Hkdf(hmac: Hmac.sha256(), outputLength: lengthBytes);
  final derived = await algorithm.deriveKey(secretKey: SecretKey(ikm), nonce: salt, info: info);
  return Uint8List.fromList(await derived.extractBytes());
}

// ── HMAC-SHA256 ──────────────────────────────────────────────────────────────

Future<Uint8List> hmacSha256(Uint8List key, Uint8List message) async {
  final mac = await Hmac.sha256().calculateMac(message, secretKey: SecretKey(key));
  return Uint8List.fromList(mac.bytes);
}

// ── AES-256-GCM ──────────────────────────────────────────────────────────────

const int _gcmNonceLength = 12; // bytes
const int _gcmTagLength = 16; // bytes (128-bit tag)

/// Seals plaintext with AES-256-GCM under a fresh random nonce, authenticating
/// (but not encrypting) aad. Returns nonce || ciphertext || tag — the same
/// wire layout the JS client's WebCrypto-based implementation produces.
Future<Uint8List> aesGcmSeal(Uint8List key, Uint8List plaintext, Uint8List aad) async {
  if (key.length != 32) {
    throw ArgumentError('chat-crypto: AES-256-GCM key must be 32 bytes');
  }
  final algorithm = AesGcm.with256bits();
  final secretBox = await algorithm.encrypt(plaintext, secretKey: SecretKey(key), aad: aad);
  return concatBytes([secretBox.nonce, secretBox.cipherText, secretBox.mac.bytes]);
}

/// Reverses aesGcmSeal. Throws if the tag doesn't verify (tampered
/// ciphertext, wrong key, or mismatched aad).
Future<Uint8List> aesGcmOpen(Uint8List key, Uint8List sealed, Uint8List aad) async {
  if (key.length != 32) {
    throw ArgumentError('chat-crypto: AES-256-GCM key must be 32 bytes');
  }
  if (sealed.length < _gcmNonceLength + _gcmTagLength) {
    throw ArgumentError('chat-crypto: sealed payload too short');
  }
  final nonce = sealed.sublist(0, _gcmNonceLength);
  final cipherText = sealed.sublist(_gcmNonceLength, sealed.length - _gcmTagLength);
  final tag = sealed.sublist(sealed.length - _gcmTagLength);
  final algorithm = AesGcm.with256bits();
  final secretBox = SecretBox(cipherText, nonce: nonce, mac: Mac(tag));
  final plaintext = await algorithm.decrypt(secretBox, secretKey: SecretKey(key), aad: aad);
  return Uint8List.fromList(plaintext);
}
