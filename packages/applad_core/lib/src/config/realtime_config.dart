library;

/// Real-time messaging broker configuration (`realtime/realtime.yaml`).
final class RealtimeConfig {
  const RealtimeConfig({
    required this.broker,
    this.url,
    this.maxConnectionsPerClient = 5,
    this.presenceEnabled = false,
  });

  factory RealtimeConfig.fromMap(Map<String, dynamic> map) {
    return RealtimeConfig(
      broker: RealtimeBroker.fromString(map['broker'] as String? ?? 'internal'),
      url: map['url'] as String?,
      maxConnectionsPerClient: map['max_connections_per_client'] as int? ?? 5,
      presenceEnabled: map['presence_enabled'] as bool? ?? false,
    );
  }

  final RealtimeBroker broker;
  final String? url;
  final int maxConnectionsPerClient;
  final bool presenceEnabled;
}

enum RealtimeBroker {
  internal,
  nats,
  redis;

  static RealtimeBroker fromString(String value) {
    return switch (value.toLowerCase()) {
      'nats' => RealtimeBroker.nats,
      'redis' => RealtimeBroker.redis,
      'internal' => RealtimeBroker.internal,
      _ => throw ArgumentError('Unknown realtime broker: $value'),
    };
  }
}
