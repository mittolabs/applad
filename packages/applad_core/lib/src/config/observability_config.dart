library;

/// Observability configuration (`observability/observability.yaml`).
final class ObservabilityConfig {
  const ObservabilityConfig({
    this.logLevel = LogLevel.info,
    this.tracing = false,
    this.metricsEnabled = false,
    this.exporters = const [],
  });

  factory ObservabilityConfig.fromMap(Map<String, dynamic> map) {
    bool parseTracing(dynamic val) {
      if (val is bool) return val;
      if (val is Map) return val['enabled'] as bool? ?? false;
      return false;
    }

    String parseLogLevel(Map<String, dynamic> m) {
      if (m['log_level'] is String) return m['log_level'] as String;
      if (m['logging'] is Map)
        return (m['logging'] as Map)['level'] as String? ?? 'info';
      return 'info';
    }

    final rawExporters = map['exporters'] ?? map['export'];

    return ObservabilityConfig(
      logLevel: LogLevel.fromString(parseLogLevel(map)),
      tracing: parseTracing(map['tracing']),
      metricsEnabled: map['metrics_enabled'] as bool? ?? false,
      exporters: (rawExporters as List?)
              ?.map((e) =>
                  ObservabilityExporter.fromMap(e as Map<String, dynamic>))
              .toList() ??
          [],
    );
  }

  final LogLevel logLevel;
  final bool tracing;
  final bool metricsEnabled;
  final List<ObservabilityExporter> exporters;
}

enum LogLevel {
  debug,
  info,
  warning,
  error;

  static LogLevel fromString(String value) {
    return switch (value.toLowerCase()) {
      'debug' => LogLevel.debug,
      'info' => LogLevel.info,
      'warning' || 'warn' => LogLevel.warning,
      'error' => LogLevel.error,
      _ => LogLevel.info,
    };
  }
}

final class ObservabilityExporter {
  const ObservabilityExporter({required this.type, required this.endpoint});

  factory ObservabilityExporter.fromMap(Map<String, dynamic> map) {
    return ObservabilityExporter(
      type: map['type'] as String,
      endpoint: map['endpoint'] as String,
    );
  }

  final String type; // otlp, prometheus, etc.
  final String endpoint;
}
