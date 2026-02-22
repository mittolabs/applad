library;

import '../models/secret_ref.dart';
import '../models/source_block.dart';
import '../utils/env_parser.dart';

/// Deployment configuration (`deployments/*.yaml`).
final class DeploymentConfig {
  const DeploymentConfig({
    required this.name,
    required this.platform,
    this.buildCommand,
    this.credentialsRef,
    this.region,
    this.environment = const {},
    this.source,
  });

  factory DeploymentConfig.fromMap(Map<String, dynamic> map) {
    final buildMap = map['build'] is Map ? map['build'] as Map : null;
    final signingMap = map['signing'] is Map ? map['signing'] as Map : null;

    return DeploymentConfig(
      name: map['name']?.toString() ?? '',
      platform: map['platform']?.toString() ?? 'unknown',
      buildCommand: (map['build_command'] ?? buildMap?['command'])?.toString(),
      credentialsRef:
          (map['credentials'] ?? signingMap?['keystore']) is String &&
                  SecretRef.isSecretRef(
                      (map['credentials'] ?? signingMap?['keystore']) as String)
              ? SecretRef.parse(
                  (map['credentials'] ?? signingMap?['keystore']) as String)
              : null,
      region: map['region']?.toString(),
      environment: parseEnvironment(map['environment']),
      source: map['source'] != null
          ? SourceBlock.fromMap(Map<String, dynamic>.from(map['source'] as Map))
          : null,
    );
  }

  final String name;
  final String platform; // fly, railway, render, aws, gcp, azure
  final String? buildCommand;
  final SecretRef? credentialsRef;
  final String? region;
  final Map<String, String> environment;
  final SourceBlock? source;

  Map<String, dynamic> toJson() => {
        'name': name,
        'platform': platform,
        'build_command': buildCommand,
        'credentials': credentialsRef?.toString(),
        'region': region,
        'environment': environment,
        'source': source?.toJson(),
      };
}
