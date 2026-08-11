// Device identity: the long-term key material a device generates once and
// publishes (public halves only) so other devices can start an encrypted
// session with it. Two separate keypairs stand in for Signal's single
// XEdDSA identity key (which reuses one Curve25519 key for both signing and
// DH via a birational map): an X25519 pair for Diffie-Hellman agreement and
// an Ed25519 pair for signing the signed prekey. Mirrors
// sdks/js/src/crypto/identity.ts exactly — see that file for the rationale.
import 'dart:typed_data';

import 'bytes.dart';
import 'primitives.dart';

class IdentityKeyPair {
  final KeyPairBytes dh;
  final KeyPairBytes sign;
  const IdentityKeyPair({required this.dh, required this.sign});
}

class SignedPrekey {
  final int keyId;
  final KeyPairBytes keyPair;
  final Uint8List signature;
  const SignedPrekey({required this.keyId, required this.keyPair, required this.signature});
}

class OneTimePrekey {
  final int keyId;
  final KeyPairBytes keyPair;
  const OneTimePrekey({required this.keyId, required this.keyPair});
}

/// Generates a new device identity. Call this once per device and persist
/// the private halves in platform secure storage — they must never leave
/// the device or be sent to the server.
Future<IdentityKeyPair> generateIdentityKeyPair() async {
  final dh = await generateX25519KeyPair();
  final sign = await generateEd25519KeyPair();
  return IdentityKeyPair(dh: dh, sign: sign);
}

/// Generates a new signed prekey, rotated periodically (recommend every
/// 7-30 days) and re-published via the device's registration endpoint.
Future<SignedPrekey> generateSignedPrekey(IdentityKeyPair identity, int keyId) async {
  final keyPair = await generateX25519KeyPair();
  final signature = await ed25519Sign(keyPair.publicKey, identity.sign);
  return SignedPrekey(keyId: keyId, keyPair: keyPair, signature: signature);
}

/// Generates a batch of one-time prekeys with sequential ids starting at
/// startId, to upload alongside registration or a top-up.
Future<List<OneTimePrekey>> generateOneTimePrekeys(int startId, int count) async {
  final out = <OneTimePrekey>[];
  for (var i = 0; i < count; i++) {
    out.add(OneTimePrekey(keyId: startId + i, keyPair: await generateX25519KeyPair()));
  }
  return out;
}

/// Verifies a signed prekey's signature against the claimed identity's
/// signing key, before it's trusted for X3DH.
Future<bool> verifySignedPrekey(Uint8List identityPublicSign, Uint8List signedPrekeyPublic, Uint8List signature) {
  return ed25519Verify(signature, signedPrekeyPublic, identityPublicSign);
}

// ── Wire encoding ────────────────────────────────────────────────────────────
//
// The backend's `identityKey` field is an opaque string as far as the server
// is concerned — it never decodes it. The client packs both public halves
// (DH + sign, 32 bytes each) into one base64 blob so no schema/migration
// change was needed to add the signing key.

String encodeIdentityPublicKey(IdentityKeyPair identity) {
  return toBase64(concatBytes([identity.dh.publicKey, identity.sign.publicKey]));
}

class IdentityPublicKeys {
  final Uint8List dh;
  final Uint8List sign;
  const IdentityPublicKeys({required this.dh, required this.sign});
}

IdentityPublicKeys decodeIdentityPublicKey(String encoded) {
  final bytes = fromBase64(encoded);
  if (bytes.length != 64) {
    throw ArgumentError('chat-crypto: malformed identity key (expected 64 bytes: dh(32) + sign(32))');
  }
  return IdentityPublicKeys(dh: bytes.sublist(0, 32), sign: bytes.sublist(32, 64));
}
