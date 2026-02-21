library;

/// Workflow configuration (`workflows/*.yaml`).
final class WorkflowConfig {
  const WorkflowConfig({
    required this.name,
    required this.steps,
    this.trigger,
    this.retry,
    this.timeout,
  });

  factory WorkflowConfig.fromMap(Map<String, dynamic> map) {
    return WorkflowConfig(
      name: map['name'] as String,
      steps: (map['steps'] as List)
          .map((s) => WorkflowStep.fromMap(s as Map<String, dynamic>))
          .toList(),
      trigger: map['trigger'] as String?,
      retry: map['retry'] != null
          ? RetryConfig.fromMap(map['retry'] as Map<String, dynamic>)
          : null,
      timeout: map['timeout_seconds'] as int?,
    );
  }

  final String name;
  final List<WorkflowStep> steps;
  final String? trigger; // event name or cron
  final RetryConfig? retry;
  final int? timeout;
}

final class WorkflowStep {
  const WorkflowStep({
    required this.name,
    required this.action,
    this.params = const {},
    this.condition,
  });

  factory WorkflowStep.fromMap(Map<String, dynamic> map) {
    return WorkflowStep(
      name: map['name'] as String,
      action: map['action'] as String,
      params: (map['params'] as Map?)?.cast<String, dynamic>() ?? {},
      condition: map['condition'] as String?,
    );
  }

  final String name;
  final String action; // e.g. "function.invoke", "email.send"
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
