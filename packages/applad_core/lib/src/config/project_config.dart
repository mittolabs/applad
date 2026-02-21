library;

import '../models/environment.dart';

/// Project-level configuration (`project.yaml`).
final class ProjectConfig {
  const ProjectConfig({
    required this.id,
    required this.name,
    required this.orgId,
    this.environments = const {},
    this.defaultEnvironment = Environment.development,
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

    return ProjectConfig(
      id: map['id']?.toString() ?? '',
      name: map['name']?.toString() ?? '',
      orgId: (map['org_id'] ?? map['org'])?.toString() ?? '',
      environments: envMap,
      defaultEnvironment: map['default_environment'] != null
          ? Environment.fromString(map['default_environment'].toString())
          : Environment.development,
    );
  }

  final String id;
  final String name;
  final String orgId;
  final Map<Environment, ProjectEnvironmentConfig> environments;
  final Environment defaultEnvironment;

  Map<String, dynamic> toJson() => {
        'id': id,
        'name': name,
        'org_id': orgId,
        'environments':
            environments.map((k, v) => MapEntry(k.name, v.toJson())),
        'default_environment': defaultEnvironment.name,
      };
}

final class ProjectEnvironmentConfig {
  const ProjectEnvironmentConfig({
    this.infraTarget,
    this.host,
    this.user,
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

    final rawVars = map['variables'];
    Map<String, String> parsedVars = {};
    if (rawVars is Map) {
      for (final entry in rawVars.entries) {
        parsedVars[entry.key.toString()] = entry.value?.toString() ?? '';
      }
    }

    return ProjectEnvironmentConfig(
      infraTarget: map['infra_target']?.toString() ?? expectedType,
      host: expectedHost,
      user: expectedUser,
      variables: parsedVars,
    );
  }

  final String? infraTarget;
  final String? host;
  final String? user;
  final Map<String, String> variables;

  Map<String, dynamic> toJson() => {
        'infra_target': infraTarget,
        'host': host,
        'user': user,
        'variables': variables,
      };
}
