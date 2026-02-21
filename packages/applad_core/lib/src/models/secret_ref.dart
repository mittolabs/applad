library;

/// Represents a reference to a secret — never the value itself.
///
/// Secret references appear in config as `${SECRET_NAME}`.
final class SecretRef {
  const SecretRef(this.name);

  /// Parses a secret reference string like `${MY_SECRET}`.
  factory SecretRef.parse(String value) {
    final match = _pattern.firstMatch(value);
    if (match == null) {
      throw FormatException('Not a valid secret reference: $value');
    }
    return SecretRef(match.group(1)!);
  }

  static final _pattern = RegExp(r'^\$\{([A-Z_][A-Z0-9_]*)\}$');

  final String name;

  /// Whether a string looks like a secret reference.
  static bool isSecretRef(String value) => _pattern.hasMatch(value);

  @override
  String toString() => '\${$name}';

  @override
  bool operator ==(Object other) => other is SecretRef && other.name == name;

  @override
  int get hashCode => name.hashCode;
}
