library;

/// Parses environment variables from either a Map or a List of Maps.
///
/// Supports:
/// environment:
///   KEY: VALUE
///
/// and:
/// environment:
///   - key: KEY
///     value: VALUE
Map<String, String> parseEnvironment(dynamic raw) {
  final Map<String, String> result = {};
  if (raw is Map) {
    for (final entry in raw.entries) {
      result[entry.key.toString()] = entry.value?.toString() ?? '';
    }
  } else if (raw is List) {
    for (final item in raw) {
      if (item is Map) {
        final key = (item['key'] ?? item['name'])?.toString();
        final value = (item['value'] ?? item['val'])?.toString() ?? '';
        if (key != null) {
          result[key] = value;
        }
      }
    }
  }
  return result;
}
