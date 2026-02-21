/// Base error class and typed subclasses for Applad.
library;

/// Base class for all Applad errors.
sealed class ApplAdError implements Exception {
  const ApplAdError(this.message, {this.cause});

  final String message;
  final Object? cause;

  @override
  String toString() => 'ApplAdError: $message${cause != null ? '\nCaused by: $cause' : ''}';
}

/// Raised when a config file cannot be found or parsed.
final class ConfigError extends ApplAdError {
  const ConfigError(super.message, {super.cause, this.filePath, this.lineNumber});

  final String? filePath;
  final int? lineNumber;

  @override
  String toString() {
    final location = [
      if (filePath != null) filePath,
      if (lineNumber != null) 'line $lineNumber',
    ].join(':');
    return 'ConfigError${location.isNotEmpty ? ' ($location)' : ''}: $message';
  }
}

/// Raised when config validation fails.
final class ValidationError extends ApplAdError {
  const ValidationError(super.message, {super.cause, this.violations = const []});

  final List<ValidationViolation> violations;

  @override
  String toString() {
    if (violations.isEmpty) return 'ValidationError: $message';
    final details = violations.map((v) => '  - ${v.path}: ${v.message}').join('\n');
    return 'ValidationError: $message\n$details';
  }
}

/// A single validation rule violation.
final class ValidationViolation {
  const ValidationViolation({required this.path, required this.message, this.severity = ViolationSeverity.error});

  final String path;
  final String message;
  final ViolationSeverity severity;

  @override
  String toString() => '[$severity] $path: $message';
}

enum ViolationSeverity { error, warning, info }

/// Raised when SSH identity or key operations fail.
final class SSHError extends ApplAdError {
  const SSHError(super.message, {super.cause});
}

/// Raised when a secret reference cannot be resolved.
final class SecretError extends ApplAdError {
  const SecretError(super.message, {super.cause, this.secretName});

  final String? secretName;
}

/// Raised when the server fails to start or process a request.
final class ServerError extends ApplAdError {
  const ServerError(super.message, {super.cause, this.statusCode});

  final int? statusCode;
}

/// Raised when a CLI command is used incorrectly.
final class UsageError extends ApplAdError {
  const UsageError(super.message, {super.cause});
}
