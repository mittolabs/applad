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
    return ObservabilityConfig(
      logLevel: LogLevel.fromString(map['log_level'] as String? ?? 'info'),
      tracing: map['tracing'] as bool? ?? false,
      metricsEnabled: map['metrics_enabled'] as bool? ?? false,
      exporters: (map['exporters'] as List?)
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
