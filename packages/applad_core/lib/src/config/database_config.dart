library;

import '../models/secret_ref.dart';

/// Database configuration (`database/database.yaml`).
final class DatabaseConfig {
  const DatabaseConfig({
    this.connections = const {},
    this.defaultConnection = 'default',
  });

  factory DatabaseConfig.fromMap(Map<String, dynamic> map) {
    String defConn = map['default']?.toString() ?? 'default';
    Map<String, DatabaseConnection> conns = {};

    if (map['connections'] is Map) {
      final cmap = map['connections'] as Map;
      for (final entry in cmap.entries) {
        final val = entry.value;
        if (val is Map) {
          conns[entry.key.toString()] =
              DatabaseConnection.fromMap(Map<String, dynamic>.from(val));
        }
      }
    } else if (map['connections'] is List) {
      final clist = map['connections'] as List;
      for (final item in clist) {
        if (item is Map) {
          final id = item['id']?.toString() ?? 'default';
          conns[id] =
              DatabaseConnection.fromMap(Map<String, dynamic>.from(item));
        }
      }
    } else {
      // Fallback: If it's a flat struct or old format
      conns[defConn] = DatabaseConnection.fromMap(map);
    }

    if (conns.isEmpty) {
      conns[defConn] = DatabaseConnection.fromMap(map);
    }

    return DatabaseConfig(
      connections: conns,
      defaultConnection: defConn,
    );
  }

  final Map<String, DatabaseConnection> connections;
  final String defaultConnection;

  Map<String, dynamic> toJson() => {
        'default': defaultConnection,
        'connections': connections.map((k, v) => MapEntry(k, v.toJson())),
      };
}

final class DatabaseConnection {
  const DatabaseConnection({
    required this.adapter,
    this.connectionStringRef,
    this.host,
    this.port,
    this.database,
    this.username,
    this.passwordRef,
    this.pool,
    this.migrations,
  });

  factory DatabaseConnection.fromMap(Map<String, dynamic> map) {
    return DatabaseConnection(
      adapter:
          DatabaseAdapter.fromString(map['adapter']?.toString() ?? 'sqlite'),
      connectionStringRef:
          _maybeSecretRef(map['connection_string'] ?? map['url']),
      host: map['host']?.toString(),
      port: map['port'] as int?,
      database: (map['database'] ?? map['db'] ?? map['path'])?.toString(),
      username: (map['username'] ?? map['user'])?.toString(),
      passwordRef: _maybeSecretRef(map['password'] ?? map['pass']),
      pool: map['pool'] != null
          ? PoolConfig.fromMap(Map<String, dynamic>.from(map['pool'] as Map))
          : null,
      migrations: map['migrations'] != null
          ? MigrationsConfig.fromMap(
              Map<String, dynamic>.from(map['migrations'] as Map))
          : null,
    );
  }

  static SecretRef? _maybeSecretRef(dynamic value) {
    if (value is String && SecretRef.isSecretRef(value)) {
      return SecretRef.parse(value);
    }
    return null;
  }

  final DatabaseAdapter adapter;
  final SecretRef? connectionStringRef;
  final String? host;
  final int? port;
  final String? database;
  final String? username;
  final SecretRef? passwordRef;
  final PoolConfig? pool;
  final MigrationsConfig? migrations;

  Map<String, dynamic> toJson() => {
        'adapter': adapter.name,
        'connection_string': connectionStringRef?.toString(),
        'host': host,
        'port': port,
        'database': database,
        'username': username,
        'password': passwordRef?.toString(),
        'pool': pool?.toJson(),
        'migrations': migrations?.toJson(),
      };
}

enum DatabaseAdapter {
  sqlite,
  postgres,
  mysql,
  turso;

  static DatabaseAdapter fromString(String value) {
    return switch (value.toLowerCase()) {
      'sqlite' => DatabaseAdapter.sqlite,
      'postgres' || 'postgresql' => DatabaseAdapter.postgres,
      'mysql' => DatabaseAdapter.mysql,
      'turso' || 'libsql' => DatabaseAdapter.turso,
      _ => throw ArgumentError('Unknown database adapter: $value'),
    };
  }
}

final class PoolConfig {
  const PoolConfig({this.minConnections = 1, this.maxConnections = 10});

  factory PoolConfig.fromMap(Map<String, dynamic> map) {
    return PoolConfig(
      minConnections: map['min'] as int? ?? 1,
      maxConnections: map['max'] as int? ?? 10,
    );
  }

  final int minConnections;
  final int maxConnections;

  Map<String, dynamic> toJson() => {
        'min': minConnections,
        'max': maxConnections,
      };
}

final class MigrationsConfig {
  const MigrationsConfig({this.directory = 'migrations', this.autoRun = false});

  factory MigrationsConfig.fromMap(Map<String, dynamic> map) {
    return MigrationsConfig(
      directory: map['directory'] as String? ?? 'migrations',
      autoRun: map['auto_run'] as bool? ?? false,
    );
  }

  final String directory;
  final bool autoRun;

  Map<String, dynamic> toJson() => {
        'directory': directory,
        'auto_run': autoRun,
      };
}
