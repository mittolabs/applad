library;

import '../models/environment.dart';

/// Feature flag configuration (`flags/*.yaml`).
final class FlagConfig {
  const FlagConfig({
    required this.name,
    this.defaultEnabled = false,
    this.environmentOverrides = const {},
    this.description,
    this.rolloutPercentage,
  });

  factory FlagConfig.fromMap(Map<String, dynamic> map) {
    final overrides = <Environment, bool>{};
    final rawOverrides = map['environments'] as Map?;
    if (rawOverrides != null) {
      for (final entry in rawOverrides.entries) {
        final env = Environment.fromString(entry.key as String);
        overrides[env] = entry.value as bool;
      }
    }
    return FlagConfig(
      name: map['name'] as String,
      defaultEnabled: map['default_enabled'] as bool? ?? false,
      environmentOverrides: overrides,
      description: map['description'] as String?,
      rolloutPercentage: map['rollout_percentage'] as int?,
    );
  }

  final String name;
  final bool defaultEnabled;
  final Map<Environment, bool> environmentOverrides;
  final String? description;
  final int? rolloutPercentage; // 0-100

  bool isEnabledFor(Environment env) => environmentOverrides[env] ?? defaultEnabled;
}
