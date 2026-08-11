// Byte/encoding helpers shared by the chat crypto layer. Dart's dart:convert
// and dart:typed_data already give us a real cross-platform base64 and
// random source, so this file is much thinner than its JS counterpart (which
// has to paper over browser/Node/React Native differences by hand).
import 'dart:convert';
import 'dart:math';
import 'dart:typed_data';

String toBase64(List<int> bytes) => base64.encode(bytes);

Uint8List fromBase64(String encoded) => base64.decode(encoded);

Uint8List utf8ToBytes(String s) => Uint8List.fromList(utf8.encode(s));

String bytesToUtf8(List<int> bytes) => utf8.decode(bytes);

/// Concatenates any number of byte lists into one Uint8List.
Uint8List concatBytes(List<List<int>> parts) {
  final total = parts.fold<int>(0, (n, p) => n + p.length);
  final out = Uint8List(total);
  var offset = 0;
  for (final p in parts) {
    out.setRange(offset, offset + p.length, p);
    offset += p.length;
  }
  return out;
}

final _secureRandom = Random.secure();

/// Cryptographically random bytes.
Uint8List randomBytes(int length) {
  final out = Uint8List(length);
  for (var i = 0; i < length; i++) {
    out[i] = _secureRandom.nextInt(256);
  }
  return out;
}

/// Encodes a non-negative integer as 4 big-endian bytes — used to bind a
/// counter (e.g. a ratchet chain/message index) into associated data.
Uint8List u32be(int n) {
  final out = Uint8List(4);
  final view = ByteData.view(out.buffer);
  view.setUint32(0, n, Endian.big);
  return out;
}

/// Constant-structure lexicographic comparison of two byte lists — used to
/// pick a side-independent canonical order (e.g. for associated data), not
/// for anything requiring timing-safety.
int compareBytes(List<int> a, List<int> b) {
  final len = a.length < b.length ? a.length : b.length;
  for (var i = 0; i < len; i++) {
    if (a[i] != b[i]) return a[i] - b[i];
  }
  return a.length - b.length;
}
