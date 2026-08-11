// Double Ratchet: per-message forward secrecy and post-compromise security
// for an established session, seeded by X3DH's shared secret. Follows
// https://signal.org/docs/specifications/doubleratchet/ — symmetric-key
// ratchet (KDF_CK) advances every message, Diffie-Hellman ratchet (KDF_RK)
// advances whenever the peer starts using a new ratchet key, and out-of-order
// delivery is handled by caching skipped message keys rather than requiring
// in-order arrival. Mirrors sdks/js/src/crypto/doubleRatchet.ts exactly (same
// KDF construction, same associated-data scheme, same skip bound) — that
// file's test suite (session.test.ts) is the closest thing this algorithm
// has to verification, since no Dart toolchain was available to run this
// port directly.
import 'dart:typed_data';

import 'bytes.dart';
import 'primitives.dart';

final Uint8List _ratchetInfo = utf8ToBytes('ApplaChatRatchet-v1');
// Bounds how many message keys one DH ratchet step will derive-and-cache
// before giving up, so a forged/corrupt "n" (message index) in a header
// cannot force unbounded work or memory growth.
const int _maxSkip = 1000;

class RatchetState {
  // Fixed for the life of the session — used only to bind ciphertexts to
  // this specific pair of identities (see _associatedData below).
  final Uint8List ourIdentityDHPublicKey;
  final Uint8List theirIdentityDHPublicKey;

  Uint8List rootKey;
  Uint8List dhSelfPublicKey;
  Uint8List dhSelfSecretKey;
  Uint8List? dhRemotePublicKey;
  Uint8List? chainKeySend;
  Uint8List? chainKeyRecv;
  int nSend;
  int nRecv;

  /// Length of the previous sending chain — tells the peer how many
  /// messages of the OLD chain to expect at most, so they know how far to
  /// skip before ratcheting.
  int pn;

  /// Cached message keys for messages that arrived out of order, keyed by
  /// "{base64 dhRemotePublicKey}:{n}".
  final Map<String, Uint8List> skippedMessageKeys;

  RatchetState({
    required this.ourIdentityDHPublicKey,
    required this.theirIdentityDHPublicKey,
    required this.rootKey,
    required this.dhSelfPublicKey,
    required this.dhSelfSecretKey,
    this.dhRemotePublicKey,
    this.chainKeySend,
    this.chainKeyRecv,
    this.nSend = 0,
    this.nRecv = 0,
    this.pn = 0,
    Map<String, Uint8List>? skippedMessageKeys,
  }) : skippedMessageKeys = skippedMessageKeys ?? {};

  KeyPairBytes get _dhSelf => KeyPairBytes(publicKey: dhSelfPublicKey, secretKey: dhSelfSecretKey);
}

class RatchetHeader {
  final String dh; // base64 sender's current ratchet public key
  final int pn;
  final int n;
  const RatchetHeader({required this.dh, required this.pn, required this.n});
}

class RatchetMessage {
  final RatchetHeader header;
  final String ciphertext; // base64 AES-256-GCM output (aesGcmSeal's nonce||ct||tag)
  const RatchetMessage({required this.header, required this.ciphertext});
}

class _RootChain {
  final Uint8List rootKey;
  final Uint8List chainKey;
  const _RootChain(this.rootKey, this.chainKey);
}

Future<_RootChain> _kdfRK(Uint8List rootKey, Uint8List dhOut) async {
  final out = await hkdf(dhOut, rootKey, _ratchetInfo, 64);
  return _RootChain(out.sublist(0, 32), out.sublist(32, 64));
}

class _ChainMessageKey {
  final Uint8List chainKey;
  final Uint8List messageKey;
  const _ChainMessageKey(this.chainKey, this.messageKey);
}

Future<_ChainMessageKey> _kdfCK(Uint8List chainKey) async {
  // Signal spec: message key from HMAC(chain_key, 0x01), next chain key from
  // HMAC(chain_key, 0x02) — two single-byte-constant HMACs over the same key.
  final results = await Future.wait([
    hmacSha256(chainKey, Uint8List.fromList([0x01])),
    hmacSha256(chainKey, Uint8List.fromList([0x02])),
  ]);
  return _ChainMessageKey(results[1], results[0]);
}

/// Initializes ratchet state as the party who sends the first message (the
/// X3DH initiator). theirSignedPrekeyPublicKey is the recipient's signed
/// prekey — used as their initial ratchet public key, exactly as the Signal
/// spec's reference X3DH-to-Ratchet handoff does.
Future<RatchetState> initRatchetAsSender(
  Uint8List sharedSecret,
  Uint8List ourIdentityDHPublicKey,
  Uint8List theirIdentityDHPublicKey,
  Uint8List theirSignedPrekeyPublicKey,
) async {
  final dh = await generateX25519KeyPair();
  final dhOut = await x25519SharedSecret(dh, theirSignedPrekeyPublicKey);
  final derived = await _kdfRK(sharedSecret, dhOut);
  return RatchetState(
    ourIdentityDHPublicKey: ourIdentityDHPublicKey,
    theirIdentityDHPublicKey: theirIdentityDHPublicKey,
    rootKey: derived.rootKey,
    dhSelfPublicKey: dh.publicKey,
    dhSelfSecretKey: dh.secretKey,
    dhRemotePublicKey: theirSignedPrekeyPublicKey,
    chainKeySend: derived.chainKey,
  );
}

/// Initializes ratchet state as the party who receives the first message.
/// ourSignedPrekey must be the same signed prekey (keyId) the sender's
/// prekey message header names — it doubles as our initial ratchet key pair.
RatchetState initRatchetAsReceiver(
  Uint8List sharedSecret,
  Uint8List ourIdentityDHPublicKey,
  Uint8List theirIdentityDHPublicKey,
  KeyPairBytes ourSignedPrekey,
) {
  return RatchetState(
    ourIdentityDHPublicKey: ourIdentityDHPublicKey,
    theirIdentityDHPublicKey: theirIdentityDHPublicKey,
    rootKey: sharedSecret,
    dhSelfPublicKey: ourSignedPrekey.publicKey,
    dhSelfSecretKey: ourSignedPrekey.secretKey,
  );
}

/// Encrypts plaintext, advancing the sending chain by one message. Mutates
/// state.
Future<RatchetMessage> ratchetEncrypt(RatchetState state, Uint8List plaintext) async {
  final sendChain = state.chainKeySend;
  if (sendChain == null) {
    throw StateError('chat-crypto: no sending chain established yet (has a message been received to complete the handshake?)');
  }
  final derived = await _kdfCK(sendChain);
  final header = RatchetHeader(dh: toBase64(state.dhSelfPublicKey), pn: state.pn, n: state.nSend);
  state.chainKeySend = derived.chainKey;
  state.nSend += 1;

  final sealed = await aesGcmSeal(derived.messageKey, plaintext, _associatedData(state, header));
  return RatchetMessage(header: header, ciphertext: toBase64(sealed));
}

/// Decrypts a received message, performing a DH ratchet step first if the
/// message's header carries a new ratchet public key, and deriving (and
/// caching) any skipped message keys along the way so later out-of-order
/// messages still decrypt. Mutates state. Throws on a forged/corrupt header
/// or on AEAD authentication failure — never returns partial or garbage
/// plaintext.
Future<Uint8List> ratchetDecrypt(RatchetState state, RatchetMessage message) async {
  final theirDh = fromBase64(message.header.dh);

  final cacheKey = _skippedKeyId(theirDh, message.header.n);
  final cached = state.skippedMessageKeys[cacheKey];
  if (cached != null) {
    state.skippedMessageKeys.remove(cacheKey);
    return aesGcmOpen(cached, fromBase64(message.ciphertext), _associatedData(state, message.header));
  }

  if (state.dhRemotePublicKey == null || toBase64(state.dhRemotePublicKey!) != toBase64(theirDh)) {
    await _skipMessageKeys(state, message.header.pn);
    await _dhRatchetStep(state, theirDh);
  }
  await _skipMessageKeys(state, message.header.n);

  final recvChain = state.chainKeyRecv;
  if (recvChain == null) {
    throw StateError('chat-crypto: no receiving chain established (unexpected message header)');
  }
  final derived = await _kdfCK(recvChain);
  state.chainKeyRecv = derived.chainKey;
  state.nRecv += 1;

  return aesGcmOpen(derived.messageKey, fromBase64(message.ciphertext), _associatedData(state, message.header));
}

Future<void> _skipMessageKeys(RatchetState state, int until) async {
  if (state.chainKeyRecv == null) return; // no chain yet to skip within
  if (until - state.nRecv > _maxSkip) {
    throw StateError('chat-crypto: refusing to skip more than $_maxSkip message keys');
  }
  while (state.nRecv < until) {
    final derived = await _kdfCK(state.chainKeyRecv!);
    state.skippedMessageKeys[_skippedKeyId(state.dhRemotePublicKey!, state.nRecv)] = derived.messageKey;
    state.chainKeyRecv = derived.chainKey;
    state.nRecv += 1;
  }
}

Future<void> _dhRatchetStep(RatchetState state, Uint8List theirNewDhPublicKey) async {
  state.pn = state.nSend;
  state.nSend = 0;
  state.nRecv = 0;
  state.dhRemotePublicKey = theirNewDhPublicKey;

  final dhOutRecv = await x25519SharedSecret(state._dhSelf, theirNewDhPublicKey);
  final recv = await _kdfRK(state.rootKey, dhOutRecv);
  state.rootKey = recv.rootKey;
  state.chainKeyRecv = recv.chainKey;

  final newDh = await generateX25519KeyPair();
  state.dhSelfPublicKey = newDh.publicKey;
  state.dhSelfSecretKey = newDh.secretKey;
  final dhOutSend = await x25519SharedSecret(state._dhSelf, theirNewDhPublicKey);
  final send = await _kdfRK(state.rootKey, dhOutSend);
  state.rootKey = send.rootKey;
  state.chainKeySend = send.chainKey;
}

String _skippedKeyId(Uint8List dhPub, int n) => '${toBase64(dhPub)}:$n';

// Binds each ciphertext to this session's identity pair (in a fixed,
// side-independent byte order) plus the header, so a header can't be
// replayed against a different session and its integrity is authenticated
// alongside the ciphertext, not just carried alongside it.
Uint8List _associatedData(RatchetState state, RatchetHeader header) {
  final a = state.ourIdentityDHPublicKey;
  final b = state.theirIdentityDHPublicKey;
  final ordered = compareBytes(a, b) <= 0 ? [a, b] : [b, a];
  return concatBytes([...ordered, fromBase64(header.dh), u32be(header.pn), u32be(header.n)]);
}
