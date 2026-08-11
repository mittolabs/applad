// X3DH (Extended Triple Diffie-Hellman) session establishment: the handshake
// that lets one device start an encrypted session with another using only
// that device's published prekey bundle — no round trip required before the
// first message. Output is a 32-byte shared secret that seeds a Double
// Ratchet session (see double_ratchet.dart). Follows the algorithm from
// https://signal.org/docs/specifications/x3dh/. Mirrors
// sdks/js/src/crypto/x3dh.ts exactly — the same constants, DH ordering, and
// header shape, so the two clients interoperate.
import 'dart:typed_data';

import 'bytes.dart';
import 'identity.dart';
import 'primitives.dart';

final Uint8List _x3dhSalt = Uint8List(32); // all-zero; the info string below provides domain separation
final Uint8List _x3dhInfo = utf8ToBytes('ApplaChatX3DH-v1');

/// A recipient device's public prekey bundle, as fetched from
/// GET /chat/devices/{deviceId}/prekey-bundle.
class PrekeyBundle {
  final String deviceId;
  final String identityKey;
  final int signedPrekeyId;
  final String signedPrekey;
  final String signedPrekeySig;
  final int? oneTimePrekeyId;
  final String? oneTimePrekey;

  const PrekeyBundle({
    required this.deviceId,
    required this.identityKey,
    required this.signedPrekeyId,
    required this.signedPrekey,
    required this.signedPrekeySig,
    this.oneTimePrekeyId,
    this.oneTimePrekey,
  });
}

class X3DHInitiatorResult {
  final Uint8List sharedSecret;
  final Uint8List ephemeralPublicKey;
  final int usedSignedPrekeyId;
  final int? usedOneTimePrekeyId;
  final Uint8List theirIdentityDHPublicKey;

  const X3DHInitiatorResult({
    required this.sharedSecret,
    required this.ephemeralPublicKey,
    required this.usedSignedPrekeyId,
    required this.theirIdentityDHPublicKey,
    this.usedOneTimePrekeyId,
  });
}

/// Initiates a session with another device from its published bundle (the
/// "Alice" role in X3DH — the side sending the first message). Throws if the
/// bundle's signed prekey signature doesn't verify against its claimed
/// identity, since trusting an unverified prekey would let a
/// man-in-the-middle substitute their own key.
Future<X3DHInitiatorResult> initiateX3DH(IdentityKeyPair ourIdentity, PrekeyBundle bundle) async {
  final theirIdentity = decodeIdentityPublicKey(bundle.identityKey);
  final theirSignedPrekey = fromBase64(bundle.signedPrekey);
  final signature = fromBase64(bundle.signedPrekeySig);
  final valid = await verifySignedPrekey(theirIdentity.sign, theirSignedPrekey, signature);
  if (!valid) {
    throw StateError('chat-crypto: signed prekey signature verification failed — refusing to establish a session');
  }

  final ephemeral = await generateX25519KeyPair();

  final dh1 = await x25519SharedSecret(ourIdentity.dh, theirSignedPrekey);
  final dh2 = await x25519SharedSecret(ephemeral, theirIdentity.dh);
  final dh3 = await x25519SharedSecret(ephemeral, theirSignedPrekey);
  final parts = [dh1, dh2, dh3];
  if (bundle.oneTimePrekey != null) {
    parts.add(await x25519SharedSecret(ephemeral, fromBase64(bundle.oneTimePrekey!)));
  }

  final sharedSecret = await hkdf(concatBytes(parts), _x3dhSalt, _x3dhInfo, 32);

  return X3DHInitiatorResult(
    sharedSecret: sharedSecret,
    ephemeralPublicKey: ephemeral.publicKey,
    usedSignedPrekeyId: bundle.signedPrekeyId,
    usedOneTimePrekeyId: bundle.oneTimePrekeyId,
    theirIdentityDHPublicKey: theirIdentity.dh,
  );
}

/// The "prekey message" header an initiator sends alongside their first
/// Double Ratchet message, carrying everything the recipient needs to derive
/// the same shared secret. Server-facing envelope_type: "prekey".
class PrekeyMessageHeader {
  final String identityKey;
  final String ephemeralKey;
  final int signedPrekeyId;
  final int? oneTimePrekeyId;

  const PrekeyMessageHeader({
    required this.identityKey,
    required this.ephemeralKey,
    required this.signedPrekeyId,
    this.oneTimePrekeyId,
  });
}

PrekeyMessageHeader buildPrekeyMessageHeader(IdentityKeyPair ourIdentity, X3DHInitiatorResult result) {
  return PrekeyMessageHeader(
    identityKey: toBase64(concatBytes([ourIdentity.dh.publicKey, ourIdentity.sign.publicKey])),
    ephemeralKey: toBase64(result.ephemeralPublicKey),
    signedPrekeyId: result.usedSignedPrekeyId,
    oneTimePrekeyId: result.usedOneTimePrekeyId,
  );
}

/// Completes the handshake on the receiving ("Bob") side: given the sender's
/// identity and ephemeral public keys from a prekey message header, and our
/// own matching private key material, derives the same shared secret the
/// initiator did. ourOneTimePrekey must be the specific key the header names
/// (null if the header carries none — X3DH degrades gracefully without one,
/// per spec, at the cost of one fewer DH term).
Future<Uint8List> receiveX3DH(
  IdentityKeyPair ourIdentity,
  KeyPairBytes ourSignedPrekey,
  KeyPairBytes? ourOneTimePrekey,
  Uint8List theirIdentityDHPublicKey,
  Uint8List theirEphemeralPublicKey,
) async {
  final dh1 = await x25519SharedSecret(ourSignedPrekey, theirIdentityDHPublicKey);
  final dh2 = await x25519SharedSecret(ourIdentity.dh, theirEphemeralPublicKey);
  final dh3 = await x25519SharedSecret(ourSignedPrekey, theirEphemeralPublicKey);
  final parts = [dh1, dh2, dh3];
  if (ourOneTimePrekey != null) {
    parts.add(await x25519SharedSecret(ourOneTimePrekey, theirEphemeralPublicKey));
  }
  return hkdf(concatBytes(parts), _x3dhSalt, _x3dhInfo, 32);
}
