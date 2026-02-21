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
    return DatabaseConfig(
      adapter: DatabaseAdapter.fromString(map['adapter'] as String? ?? 'sqlite'),
      connectionStringRef: _maybeSecretRef(map['connection_string']),
      host: map['host'] as String?,
      port: map['port'] as int?,
      database: map['database'] as String?,
      username: map['username'] as String?,
      passwordRef: _maybeSecretRef(map['password']),
      pool: map['pool'] != null
          ? PoolConfig.fromMap(map['pool'] as Map<String, dynamic>)
          : null,
      migrations: map['migrations'] != null
          ? MigrationsConfig.fromMap(map['migrations'] as Map<String, dynamic>)
          : null,
    );
  }

  static SecretRef? _maybeSecretRef(dynamic value) {
    if (value is String && SecretRef.isSecretRef(value)) return SecretRef.parse(value);
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
}
