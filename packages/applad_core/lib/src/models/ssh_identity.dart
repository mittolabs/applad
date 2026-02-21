library;

/// Represents a registered SSH key identity.
final class SshIdentity {
  const SshIdentity({
    required this.fingerprint,
    required this.label,
    required this.identityString,
    this.publicKey,
  });

  factory SshIdentity.fromMap(Map<String, dynamic> map) {
    return SshIdentity(
      fingerprint: map['fingerprint'] as String,
      label: map['label'] as String,
      identityString: map['identity_string'] as String,
      publicKey: map['public_key'] as String?,
    );
  }

  /// SHA-256 fingerprint of the public key.
  final String fingerprint;

  /// Human-readable label (e.g. "mitto@macbook").
  final String label;

  /// The full identity string (e.g. "mitto@macbook-pro.local").
  final String identityString;

  /// The raw public key (optional, stored for verification).
  final String? publicKey;

  Map<String, dynamic> toMap() => {
    'fingerprint': fingerprint,
    'label': label,
    'identity_string': identityString,
    if (publicKey != null) 'public_key': publicKey,
  };

  @override
  String toString() => 'SshIdentity($label [$fingerprint])';
}
