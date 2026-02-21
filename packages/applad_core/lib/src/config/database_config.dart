library;

import '../models/secret_ref.dart';

/// Database configuration (`database/database.yaml`).
final class DatabaseConfig {
  const DatabaseConfig({
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

  factory DatabaseConfig.fromMap(Map<String, dynamic> map) {
    // If it has connections list, pick the primary one or the first one
    Map<String, dynamic> target = map;
    if (map['connections'] is List && (map['connections'] as List).isNotEmpty) {
      final connections = map['connections'] as List;
      final primary = connections.firstWhere(
        (c) => c is Map && (c['id'] == 'primary' || c['id'] == map['default']),
        orElse: () => connections.first,
      );
      if (primary is Map) {
        target = Map<String, dynamic>.from(primary);
      }
    }

    return DatabaseConfig(
      adapter:
          DatabaseAdapter.fromString(target['adapter']?.toString() ?? 'sqlite'),
      connectionStringRef:
          _maybeSecretRef(target['connection_string'] ?? target['url']),
      host: target['host']?.toString(),
      port: target['port'] as int?,
      database:
          (target['database'] ?? target['db'] ?? target['path'])?.toString(),
      username: (target['username'] ?? target['user'])?.toString(),
      passwordRef: _maybeSecretRef(target['password'] ?? target['pass']),
      pool: target['pool'] != null
          ? PoolConfig.fromMap(Map<String, dynamic>.from(target['pool'] as Map))
          : null,
      migrations: target['migrations'] != null
          ? MigrationsConfig.fromMap(
              Map<String, dynamic>.from(target['migrations'] as Map))
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
