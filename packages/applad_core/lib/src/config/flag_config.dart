library;

import '../models/environment.dart';

/// Feature flag configuration (`flags/*.yaml`).
final class FlagConfig {
  const FlagConfig({
    required this.name,
    this.type = 'boolean',
    this.defaultValue,
    this.variants = const [],
    this.environmentOverrides = const {},
    this.description,
    this.rolloutPercentage,
  });

  factory FlagConfig.fromMap(Map<String, dynamic> map) {
    final overrides = <Environment, dynamic>{};
    final rawOverrides = map['environments'] as Map?;
    if (rawOverrides != null) {
      for (final entry in rawOverrides.entries) {
        final env = Environment.fromString(entry.key.toString());
        overrides[env] = entry.value;
      }
    }
    return FlagConfig(
      name: (map['name'] ?? map['key'])?.toString() ?? '',
      type: map['type']?.toString() ?? 'boolean',
      defaultValue: map['default'] ?? map['default_enabled'],
      variants:
          (map['variants'] as List?)?.map((e) => e.toString()).toList() ?? [],
      environmentOverrides: overrides,
      description: map['description']?.toString(),
      rolloutPercentage: map['rollout_percentage'] as int?,
    );
  }

  final String name;
  final String type; // boolean, multivariate
  final dynamic defaultValue; // bool or String
  final List<String> variants;
  final Map<Environment, dynamic> environmentOverrides;
  final String? description;
  final int? rolloutPercentage; // 0-100

  Map<String, dynamic> toJson() => {
        'name': name,
        'type': type,
        'default': defaultValue,
        'variants': variants,
        'environments': environmentOverrides.map((k, v) => MapEntry(k.name, v)),
        'description': description,
        'rollout_percentage': rolloutPercentage,
      };

  dynamic valueFor(Environment env) =>
      environmentOverrides[env] ?? defaultValue;
}
