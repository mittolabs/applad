library;

import '../models/secret_ref.dart';

/// Root-level Applad instance configuration (`applad.yaml`).
final class InstanceConfig {
  const InstanceConfig({
    required this.version,
    this.ai,
    this.observability,
    this.enabledFeatures = const [],
  });

  factory InstanceConfig.fromMap(Map<String, dynamic> map) {
    return InstanceConfig(
      version: map['version'] as String? ?? '1',
      ai: map['ai'] != null
          ? AiConfig.fromMap(map['ai'] as Map<String, dynamic>)
          : null,
      observability: map['observability'] != null
          ? ObservabilityRef.fromMap(
              map['observability'] as Map<String, dynamic>)
          : null,
      enabledFeatures: (map['enabled_features'] as List?)?.cast<String>() ?? [],
    );
  }

  final String version;
  final AiConfig? ai;
  final ObservabilityRef? observability;
  final List<String> enabledFeatures;
}

final class AiConfig {
  const AiConfig({
    required this.provider,
    this.model,
    this.apiKeyRef,
  });

  factory AiConfig.fromMap(Map<String, dynamic> map) {
    return AiConfig(
      provider: map['provider'] as String,
      model: map['model'] as String?,
      apiKeyRef: map['api_key'] is String &&
              SecretRef.isSecretRef(map['api_key'] as String)
          ? SecretRef.parse(map['api_key'] as String)
          : null,
    );
  }

  final String provider;
  final String? model;
  final SecretRef? apiKeyRef;
}

final class ObservabilityRef {
  const ObservabilityRef({this.logLevel, this.tracing});

  factory ObservabilityRef.fromMap(Map<String, dynamic> map) {
    bool? parseTracing(dynamic val) {
      if (val is bool) return val;
      if (val is Map) return val['enabled'] as bool?;
      return null;
    }

    String? parseLogLevel(Map<String, dynamic> m) {
      if (m['log_level'] is String) return m['log_level'] as String;
      if (m['logging'] is Map) return (m['logging'] as Map)['level'] as String?;
      return null;
    }

    return ObservabilityRef(
      logLevel: parseLogLevel(map),
      tracing: parseTracing(map['tracing']),
    );
  }

  final String? logLevel;
  final bool? tracing;
}
