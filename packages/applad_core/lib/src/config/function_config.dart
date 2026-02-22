library;

import '../models/source_block.dart';
import '../utils/env_parser.dart';

/// Serverless function configuration (`functions/*/function.yaml`).
final class FunctionConfig {
  const FunctionConfig({
    required this.name,
    this.runtime = 'dart',
    this.trigger = FunctionTrigger.http,
    this.memory = 128,
    this.timeoutSeconds = 30,
    this.environment = const {},
    this.schedule,
    this.source,
  });

  factory FunctionConfig.fromMap(Map<String, dynamic> map) {
    return FunctionConfig(
      name: map['name'] as String,
      runtime: map['runtime'] as String? ?? 'dart',
      trigger: FunctionTrigger.fromString(map['trigger'] as String? ?? 'http'),
      memory: int.tryParse(map['memory']?.toString() ?? '') ?? 128,
      timeoutSeconds:
          int.tryParse(map['timeout_seconds']?.toString() ?? '') ?? 30,
      environment: parseEnvironment(map['environment']),
      schedule: map['schedule'] as String?,
      source: map['source'] != null
          ? SourceBlock.fromMap(Map<String, dynamic>.from(map['source'] as Map))
          : null,
    );
  }

  final String name;
  final String runtime;
  final FunctionTrigger trigger;
  final int memory; // MB
  final int timeoutSeconds;
  final Map<String, String> environment;
  final String? schedule; // cron expression for scheduled functions
  final SourceBlock? source;

  Map<String, dynamic> toJson() => {
        'name': name,
        'runtime': runtime,
        'trigger': trigger.name,
        'memory': memory,
        'timeout_seconds': timeoutSeconds,
        'environment': environment,
        'schedule': schedule,
        'source': source?.toJson(),
      };
}

enum FunctionTrigger {
  http,
  schedule,
  event,
  webhook;

  static FunctionTrigger fromString(String value) {
    return switch (value.toLowerCase()) {
      'http' => FunctionTrigger.http,
      'schedule' || 'cron' => FunctionTrigger.schedule,
      'event' => FunctionTrigger.event,
      'webhook' => FunctionTrigger.webhook,
      _ => throw ArgumentError('Unknown function trigger: $value'),
    };
  }
}
