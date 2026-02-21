library;

import '../models/secret_ref.dart';

/// Deployment configuration (`deployments/*.yaml`).
final class DeploymentConfig {
  const DeploymentConfig({
    required this.name,
    required this.platform,
    this.buildCommand,
    this.credentialsRef,
    this.region,
    this.environment = const {},
  });

  factory DeploymentConfig.fromMap(Map<String, dynamic> map) {
    return DeploymentConfig(
      name: map['name'] as String,
      platform: map['platform'] as String,
      buildCommand: map['build_command'] as String?,
      credentialsRef: map['credentials'] is String && SecretRef.isSecretRef(map['credentials'] as String)
          ? SecretRef.parse(map['credentials'] as String)
          : null,
      region: map['region'] as String?,
      environment: (map['environment'] as Map?)?.cast<String, String>() ?? {},
    );
  }

  final String name;
  final String platform; // fly, railway, render, aws, gcp, azure
  final String? buildCommand;
  final SecretRef? credentialsRef;
  final String? region;
  final Map<String, String> environment;
}
