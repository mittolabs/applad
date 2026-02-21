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
import '../config/hosting_config.dart';
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
    this.hosting = const [],
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
  final List<HostingConfig> hosting;
  final List<DeploymentConfig> deployments;
  final RealtimeConfig? realtime;
  final AnalyticsConfig? analytics;
  final ObservabilityConfig? observability;
  final SecurityConfig? security;
  final String rootPath;
}

/// Merges all YAML config files into a single [ApplAdConfig] tree.
final class ConfigMerger {
  ConfigMerger({ConfigLoader? loader}) : _loader = loader ?? const ConfigLoader();

  final ConfigLoader _loader;

  ApplAdConfig merge(String rootPath) {
    final instanceMap = _loader.loadFile(p.join(rootPath, 'applad.yaml'));
    final instance = InstanceConfig.fromMap(instanceMap);

    final orgsDir = p.join(rootPath, 'orgs');
    final orgDirs = _listSubdirs(orgsDir);
    if (orgDirs.isEmpty) throw StateError('No org directories found in $orgsDir');

    final orgDir = orgDirs.first;
    final org = OrgConfig.fromMap(_loader.loadFile(p.join(orgDir, 'org.yaml')));

    final projectsDir = p.join(orgDir, 'projects');
    final projectDirs = _listSubdirs(projectsDir);
    if (projectDirs.isEmpty) throw StateError('No project directories found in $projectsDir');

    final projectDir = projectDirs.first;
    final project = ProjectConfig.fromMap(_loader.loadFile(p.join(projectDir, 'project.yaml')));

    return ApplAdConfig(
      instance: instance,
      org: org,
      project: project,
      auth: _opt(p.join(projectDir, 'auth', 'auth.yaml'), AuthConfig.fromMap),
      database: _opt(p.join(projectDir, 'database', 'database.yaml'), DatabaseConfig.fromMap),
      tables: _loadTables(p.join(projectDir, 'tables')),
      storage: _opt(p.join(projectDir, 'storage', 'storage.yaml'), StorageConfig.fromMap),
      functions: _loadFunctions(p.join(projectDir, 'functions')),
      workflows: _loadDir(p.join(projectDir, 'workflows'), WorkflowConfig.fromMap),
      messaging: _opt(p.join(projectDir, 'messaging', 'messaging.yaml'), MessagingConfig.fromMap),
      flags: _loadDir(p.join(projectDir, 'flags'), FlagConfig.fromMap),
      hosting: _loadDir(p.join(projectDir, 'hosting'), HostingConfig.fromMap),
      deployments: _loadDir(p.join(projectDir, 'deployments'), DeploymentConfig.fromMap),
      realtime: _opt(p.join(projectDir, 'realtime', 'realtime.yaml'), RealtimeConfig.fromMap),
      analytics: _opt(p.join(projectDir, 'analytics', 'analytics.yaml'), AnalyticsConfig.fromMap),
      observability: _opt(p.join(projectDir, 'observability', 'observability.yaml'), ObservabilityConfig.fromMap),
      security: _opt(p.join(projectDir, 'security', 'security.yaml'), SecurityConfig.fromMap),
      rootPath: rootPath,
    );
  }

  T? _opt<T>(String path, T Function(Map<String, dynamic>) factory) {
    try {
      return factory(_loader.loadFile(path));
    } catch (_) {
      return null;
    }
  }

  List<TableConfig> _loadTables(String dir) {
    return _loader.loadDirectory(dir).entries.map((e) {
      final map = e.value;
      map['name'] ??= e.key;
      return TableConfig.fromMap(map);
    }).toList();
  }

  List<FunctionConfig> _loadFunctions(String dir) {
    return _listSubdirs(dir).map((funcDir) {
      return _opt(p.join(funcDir, 'function.yaml'), FunctionConfig.fromMap);
    }).whereType<FunctionConfig>().toList();
  }

  List<T> _loadDir<T>(String dir, T Function(Map<String, dynamic>) factory) {
    return _loader.loadDirectory(dir).values.map(factory).toList();
  }

  List<String> _listSubdirs(String dirPath) {
    final dir = Directory(dirPath);
    if (!dir.existsSync()) return [];
    return dir
        .listSync()
        .whereType<Directory>()
        .map((d) => d.path)
        .toList()
      ..sort();
  }
}
