library;

import 'dart:io';
import 'package:path/path.dart' as p;

import '../config/instance_config.dart';
import '../config/org_config.dart';
import '../config/project_config.dart';
import '../config/auth_config.dart';
import '../config/database_config.dart';
import '../config/table_config.dart';
import '../config/storage_config.dart';
import '../config/function_config.dart';
import '../config/workflow_config.dart';
import '../config/messaging_config.dart';
import '../config/flag_config.dart';

import '../config/deployment_config.dart';
import '../config/realtime_config.dart';
import '../config/analytics_config.dart';
import '../config/observability_config.dart';
import '../config/security_config.dart';
import 'config_loader.dart';

/// The fully resolved Applad configuration tree.
final class ApplAdConfig {
  const ApplAdConfig({
    required this.instance,
    required this.org,
    required this.project,
    this.auth,
    this.database,
    this.tables = const [],
    this.storage,
    this.functions = const [],
    this.workflows = const [],
    this.messaging,
    this.flags = const [],
    this.deployments = const [],
    this.realtime,
    this.analytics,
    this.observability,
    this.security,
    required this.rootPath,
  });

  final InstanceConfig instance;
  final OrgConfig org;
  final ProjectConfig project;
  final AuthConfig? auth;
  final DatabaseConfig? database;
  final List<TableConfig> tables;
  final StorageConfig? storage;
  final List<FunctionConfig> functions;
  final List<WorkflowConfig> workflows;
  final MessagingConfig? messaging;
  final List<FlagConfig> flags;

  final List<DeploymentConfig> deployments;
  final RealtimeConfig? realtime;
  final AnalyticsConfig? analytics;
  final ObservabilityConfig? observability;
  final SecurityConfig? security;
  final String rootPath;

  Map<String, dynamic> toJson() => {
        'instance': instance.toJson(),
        'org': org.toJson(),
        'project': project.toJson(),
        'auth': auth?.toJson(),
        'database': database?.toJson(),
        'tables': tables.map((t) => t.toJson()).toList(),
        'storage': storage?.toJson(),
        'functions': functions.map((f) => f.toJson()).toList(),
        'workflows': workflows.map((w) => w.toJson()).toList(),
        'messaging': messaging?.toJson(),
        'flags': flags.map((f) => f.toJson()).toList(),
        'deployments': deployments.map((d) => d.toJson()).toList(),
        'realtime': realtime?.toJson(),
        'analytics': analytics?.toJson(),
        'observability': observability?.toJson(),
        'security': security?.toJson(),
        'root_path': rootPath,
      };
}

/// Merges all YAML config files into a single [ApplAdConfig] tree.
final class ConfigMerger {
  ConfigMerger({ConfigLoader? loader})
      : _loader = loader ?? const ConfigLoader();

  final ConfigLoader _loader;

  ApplAdConfig merge(String rootPath) {
    final instanceMap = _loader.loadFile(p.join(rootPath, 'applad.yaml'));
    final instance = InstanceConfig.fromMap(instanceMap);

    final orgsDir = p.join(rootPath, 'orgs');
    final orgDirs = _listSubdirs(orgsDir);
    if (orgDirs.isEmpty) {
      throw StateError('No org directories found in $orgsDir');
    }

    final orgDir = orgDirs.first;
    final org = OrgConfig.fromMap(_loader.loadFile(p.join(orgDir, 'org.yaml')));

    final projectDirs = _listSubdirs(orgDir).where((dir) {
      final projectName = p.basename(dir);
      return !projectName.startsWith('.') &&
          File(p.join(dir, 'project.yaml')).existsSync();
    }).toList();
    if (projectDirs.isEmpty) {
      throw StateError('No project directories found in $orgDir');
    }

    final projectDir = projectDirs.first;
    final project = ProjectConfig.fromMap(
        _loader.loadFile(p.join(projectDir, 'project.yaml')));

    return ApplAdConfig(
      instance: instance,
      org: org,
      project: project,
      auth: _opt(p.join(projectDir, 'auth', 'auth.yaml'), AuthConfig.fromMap),
      database: _opt(p.join(projectDir, 'database', 'database.yaml'),
          DatabaseConfig.fromMap),
      storage: _opt(
          p.join(projectDir, 'storage', 'storage.yaml'), StorageConfig.fromMap),
      tables: _loadNamedFiles(
          p.join(projectDir, 'database', 'tables'), TableConfig.fromMap),
      functions: _loadNamedFiles(
          p.join(projectDir, 'functions'), FunctionConfig.fromMap),
      workflows: _loadNamedFiles(
          p.join(projectDir, 'workflows'), WorkflowConfig.fromMap),
      messaging: _opt(p.join(projectDir, 'messaging', 'messaging.yaml'),
          MessagingConfig.fromMap),
      flags: _loadNamedFiles(p.join(projectDir, 'flags'), FlagConfig.fromMap),
      deployments: _loadNamedFiles(
          p.join(projectDir, 'deployments'), DeploymentConfig.fromMap),
      realtime: _opt(p.join(projectDir, 'realtime', 'realtime.yaml'),
          RealtimeConfig.fromMap),
      analytics: _opt(p.join(projectDir, 'analytics', 'analytics.yaml'),
          AnalyticsConfig.fromMap),
      observability: _opt(
          p.join(projectDir, 'observability', 'observability.yaml'),
          ObservabilityConfig.fromMap),
      security: _opt(p.join(projectDir, 'security', 'security.yaml'),
          SecurityConfig.fromMap),
      rootPath: rootPath,
    );
  }

  T? _opt<T>(String path, T Function(Map<String, dynamic>) factory) {
    try {
      final file = File(path);
      if (!file.existsSync()) return null;
      return factory(_loader.loadFile(path));
    } catch (_) {
      return null;
    }
  }

  List<T> _loadNamedFiles<T>(
      String dirPath, T Function(Map<String, dynamic>) factory) {
    final dir = Directory(dirPath);
    if (!dir.existsSync()) return [];

    return _loader.loadDirectory(dirPath).entries.map((e) {
      final map = e.value;
      map['name'] ??= e.key;
      return factory(map);
    }).toList();
  }

  List<String> _listSubdirs(String dirPath) {
    final dir = Directory(dirPath);
    if (!dir.existsSync()) return [];
    return dir.listSync().whereType<Directory>().map((d) => d.path).toList()
      ..sort();
  }
}
