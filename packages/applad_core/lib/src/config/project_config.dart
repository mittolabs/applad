library;

import '../models/environment.dart';
import '../utils/env_parser.dart';

/// Project-level configuration (`project.yaml`).
final class ProjectConfig {
  const ProjectConfig({
    required this.id,
    required this.name,
    required this.orgId,
    this.environments = const {},
    this.defaultEnvironment = Environment.development,
    this.enabledFeatures = const [],
  });

  factory ProjectConfig.fromMap(Map<String, dynamic> map) {
    final envMap = <Environment, ProjectEnvironmentConfig>{};
    final rawEnvs = map['environments'];

    if (rawEnvs is List) {
      for (final item in rawEnvs) {
        if (item is Map) {
          final name = item['name']?.toString();
          if (name != null) {
            final env = Environment.fromString(name);
            envMap[env] = ProjectEnvironmentConfig.fromMap(
                Map<String, dynamic>.from(item));
          }
        }
      }
    } else if (rawEnvs is Map) {
      for (final entry in rawEnvs.entries) {
        final env = Environment.fromString(entry.key.toString());
        if (entry.value is Map) {
          envMap[env] = ProjectEnvironmentConfig.fromMap(
              Map<String, dynamic>.from(entry.value));
        }
      }
    }

    final rawFeatures = map['features'] ?? map['enabled_features'];
    final featuresList = <String>[];
    if (rawFeatures is Map) {
      for (final entry in rawFeatures.entries) {
        if (entry.value == true) {
          featuresList.add(entry.key.toString());
        }
      }
    } else if (rawFeatures is List) {
      featuresList.addAll(rawFeatures.map((e) => e.toString()));
    }

    return ProjectConfig(
      id: map['id']?.toString() ?? '',
      name: map['name']?.toString() ?? '',
      orgId: (map['org_id'] ?? map['org'])?.toString() ?? '',
      environments: envMap,
      defaultEnvironment: map['default_environment'] != null
          ? Environment.fromString(map['default_environment'].toString())
          : Environment.development,
      enabledFeatures: featuresList,
    );
  }

  final String id;
  final String name;
  final String orgId;
  final Map<Environment, ProjectEnvironmentConfig> environments;
  final Environment defaultEnvironment;
  final List<String> enabledFeatures;

  Map<String, dynamic> toJson() => {
        'id': id,
        'name': name,
        'org_id': orgId,
        'environments':
            environments.map((k, v) => MapEntry(k.name, v.toJson())),
        'default_environment': defaultEnvironment.name,
        'features': enabledFeatures,
      };
}

final class ProjectEnvironmentConfig {
  const ProjectEnvironmentConfig({
    this.infraTarget,
    this.host,
    this.user,
    this.engineVersion = 'latest',
    this.variables = const {},
  });

  factory ProjectEnvironmentConfig.fromMap(Map<String, dynamic> map) {
    final rawInfra = map['infrastructure'];
    String? expectedType;
    String? expectedHost;
    String? expectedUser;

    if (rawInfra is Map) {
      expectedType = rawInfra['type']?.toString();
      expectedHost = rawInfra['host']?.toString();
      expectedUser = rawInfra['user']?.toString();
    } else if (rawInfra is String) {
      expectedType = rawInfra;
    }

    return ProjectEnvironmentConfig(
      infraTarget: map['infra_target']?.toString() ?? expectedType,
      host: expectedHost,
      user: expectedUser,
      engineVersion: map['engine_version']?.toString() ?? 'latest',
      variables: parseEnvironment(map['variables'] ?? map['environment']),
    );
  }

  final String? infraTarget;
  final String? host;
  final String? user;
  final String engineVersion;
  final Map<String, String> variables;

  Map<String, dynamic> toJson() => {
        'infra_target': infraTarget,
        'host': host,
        'user': user,
        'engine_version': engineVersion,
        'variables': variables,
      };
}
