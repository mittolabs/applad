library;

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
  });

  factory FunctionConfig.fromMap(Map<String, dynamic> map) {
    return FunctionConfig(
      name: map['name'] as String,
      runtime: map['runtime'] as String? ?? 'dart',
      trigger: FunctionTrigger.fromString(map['trigger'] as String? ?? 'http'),
      memory: map['memory'] as int? ?? 128,
      timeoutSeconds: map['timeout_seconds'] as int? ?? 30,
      environment: (map['environment'] as Map?)?.cast<String, String>() ?? {},
      schedule: map['schedule'] as String?,
    );
  }

  final String name;
  final String runtime;
  final FunctionTrigger trigger;
  final int memory; // MB
  final int timeoutSeconds;
  final Map<String, String> environment;
  final String? schedule; // cron expression for scheduled functions

  Map<String, dynamic> toJson() => {
        'name': name,
        'runtime': runtime,
        'trigger': trigger.name,
        'memory': memory,
        'timeout_seconds': timeoutSeconds,
        'environment': environment,
        'schedule': schedule,
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
