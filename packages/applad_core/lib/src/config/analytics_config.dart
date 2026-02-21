library;

/// Analytics configuration (`analytics/analytics.yaml`).
final class AnalyticsConfig {
  const AnalyticsConfig({
    this.retentionDays = 90,
    this.capturedEvents = const [],
    this.enabled = true,
    this.anonymizeIps = true,
  });

  factory AnalyticsConfig.fromMap(Map<String, dynamic> map) {
    return AnalyticsConfig(
      retentionDays: map['retention_days'] as int? ?? 90,
      capturedEvents: (map['captured_events'] as List?)?.cast<String>() ?? [],
      enabled: map['enabled'] as bool? ?? true,
      anonymizeIps: map['anonymize_ips'] as bool? ?? true,
    );
  }

  final int retentionDays;
  final List<String> capturedEvents;
  final bool enabled;
  final bool anonymizeIps;

  Map<String, dynamic> toJson() => {
        'retention_days': retentionDays,
        'captured_events': capturedEvents,
        'enabled': enabled,
        'anonymize_ips': anonymizeIps,
      };
}
