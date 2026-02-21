library;

/// Real-time messaging broker configuration (`realtime/realtime.yaml`).
final class RealtimeConfig {
  const RealtimeConfig({
    required this.broker,
    this.url,
    this.maxConnectionsPerClient = 5,
    this.presenceEnabled = false,
    this.channels = const [],
  });

  factory RealtimeConfig.fromMap(Map<String, dynamic> map) {
    return RealtimeConfig(
      broker: RealtimeBroker.fromString(
          (map['broker'] ?? map['adapter'])?.toString() ?? 'internal'),
      url: map['url']?.toString(),
      maxConnectionsPerClient: map['max_connections_per_client'] as int? ?? 5,
      presenceEnabled: map['presence_enabled'] as bool? ?? false,
      channels: (map['channels'] as List?)
              ?.map((c) =>
                  RealtimeChannel.fromMap(Map<String, dynamic>.from(c as Map)))
              .toList() ??
          [],
    );
  }

  final RealtimeBroker broker;
  final String? url;
  final int maxConnectionsPerClient;
  final bool presenceEnabled;
  final List<RealtimeChannel> channels;

  Map<String, dynamic> toJson() => {
        'broker': broker.name,
        'url': url,
        'max_connections_per_client': maxConnectionsPerClient,
        'presence_enabled': presenceEnabled,
        'channels': channels.map((c) => c.toJson()).toList(),
      };
}

final class RealtimeChannel {
  const RealtimeChannel({
    required this.name,
    this.table,
    this.events = const ['create', 'update', 'delete'],
    this.permissions = const [],
  });

  factory RealtimeChannel.fromMap(Map<String, dynamic> map) {
    return RealtimeChannel(
      name: map['name']?.toString() ?? '',
      table: map['table']?.toString(),
      events: (map['events'] as List?)?.cast<String>() ??
          ['create', 'update', 'delete'],
      permissions: (map['permissions'] as List?)
              ?.map((p) => RealtimeChannelPermission.fromMap(
                  Map<String, dynamic>.from(p as Map)))
              .toList() ??
          [],
    );
  }

  final String name;
  final String? table;
  final List<String> events;
  final List<RealtimeChannelPermission> permissions;

  Map<String, dynamic> toJson() => {
        'name': name,
        'table': table,
        'events': events,
        'permissions': permissions.map((p) => p.toJson()).toList(),
      };
}

final class RealtimeChannelPermission {
  const RealtimeChannelPermission({required this.role, this.filter});

  factory RealtimeChannelPermission.fromMap(Map<String, dynamic> map) {
    return RealtimeChannelPermission(
      role: map['role']?.toString() ?? '*',
      filter: map['filter']?.toString(),
    );
  }

  final String role;
  final String? filter;

  Map<String, dynamic> toJson() => {
        'role': role,
        'filter': filter,
      };
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
