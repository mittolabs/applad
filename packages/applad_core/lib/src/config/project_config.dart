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
    final rawEnvs = map['environments'] as Map?;
    if (rawEnvs != null) {
      for (final entry in rawEnvs.entries) {
        final env = Environment.fromString(entry.key as String);
        envMap[env] = ProjectEnvironmentConfig.fromMap(entry.value as Map<String, dynamic>);
      }
    }
    return ProjectConfig(
      id: map['id'] as String,
      name: map['name'] as String,
      orgId: map['org_id'] as String,
      environments: envMap,
      defaultEnvironment: map['default_environment'] != null
          ? Environment.fromString(map['default_environment'] as String)
          : Environment.development,
    );
  }

  final String id;
  final String name;
  final String orgId;
  final Map<Environment, ProjectEnvironmentConfig> environments;
  final Environment defaultEnvironment;
}

final class ProjectEnvironmentConfig {
  const ProjectEnvironmentConfig({
    this.infraTarget,
    this.variables = const {},
  });

  factory ProjectEnvironmentConfig.fromMap(Map<String, dynamic> map) {
    return ProjectEnvironmentConfig(
      infraTarget: map['infra_target'] as String?,
      variables: (map['variables'] as Map?)?.cast<String, String>() ?? {},
    );
  }

  final String? infraTarget;
  final Map<String, String> variables;
}
