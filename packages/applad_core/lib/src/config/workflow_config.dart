library;

/// Workflow configuration (`workflows/*.yaml`).
final class WorkflowConfig {
  const WorkflowConfig({
    required this.name,
    required this.steps,
    this.triggerMap,
    this.retry,
    this.timeout,
  });

  factory WorkflowConfig.fromMap(Map<String, dynamic> map) {
    return WorkflowConfig(
      name: map['name']?.toString() ?? '',
      steps: (map['steps'] as List?)
              ?.map((s) =>
                  WorkflowStep.fromMap(Map<String, dynamic>.from(s as Map)))
              .toList() ??
          [],
      triggerMap: map['trigger'] is Map
          ? Map<String, dynamic>.from(map['trigger'] as Map)
          : null,
      retry: map['retry'] != null
          ? RetryConfig.fromMap(Map<String, dynamic>.from(map['retry'] as Map))
          : null,
      timeout: map['timeout_seconds'] as int?,
    );
  }

  final String name;
  final List<WorkflowStep> steps;
  final Map<String, dynamic>?
      triggerMap; // e.g. {type: "event", event: "auth.user.created"}
  final RetryConfig? retry;
  final int? timeout;
}

final class WorkflowStep {
  const WorkflowStep({
    required this.name,
    required this.type,
    this.channel,
    this.template,
    this.functionName,
    this.dependsOn = const [],
    this.params = const {},
    this.condition,
  });

  factory WorkflowStep.fromMap(Map<String, dynamic> map) {
    return WorkflowStep(
      name: map['name']?.toString() ?? '',
      type: map['type']?.toString() ??
          map['action']?.toString() ??
          '', // fallback to legacy action
      channel: map['channel']?.toString(),
      template: map['template']?.toString(),
      functionName: map['function']?.toString(),
      dependsOn:
          (map['depends_on'] as List?)?.map((e) => e.toString()).toList() ?? [],
      params: (map['params'] as Map?)?.cast<String, dynamic>() ?? {},
      condition: map['condition']?.toString(),
    );
  }

  final String name;
  final String type; // e.g. "message", "function"
  final String? channel;
  final String? template;
  final String? functionName;
  final List<String> dependsOn;
  final Map<String, dynamic> params;
  final String? condition;
}

final class RetryConfig {
  const RetryConfig({this.maxAttempts = 3, this.delaySeconds = 5});

  factory RetryConfig.fromMap(Map<String, dynamic> map) {
    return RetryConfig(
      maxAttempts: map['max_attempts'] as int? ?? 3,
      delaySeconds: map['delay_seconds'] as int? ?? 5,
    );
  }

  final int maxAttempts;
  final int delaySeconds;
}
