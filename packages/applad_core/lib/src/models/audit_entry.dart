library;

import 'ssh_identity.dart';

/// A full audit log entry capturing who did what, when, and why.
final class AuditEntry {
  const AuditEntry({
    required this.id,
    required this.timestamp,
    required this.action,
    required this.actor,
    this.via,
    this.diff,
    this.instructionPrompt,
    this.metadata = const {},
  });

  factory AuditEntry.fromMap(Map<String, dynamic> map) {
    return AuditEntry(
      id: map['id'] as String,
      timestamp: DateTime.parse(map['timestamp'] as String),
      action: map['action'] as String,
      actor: SshIdentity.fromMap(map['actor'] as Map<String, dynamic>),
      via: map['via'] as String?,
      diff: map['diff'] as String?,
      instructionPrompt: map['instruction_prompt'] as String?,
      metadata: (map['metadata'] as Map?)?.cast<String, dynamic>() ?? {},
    );
  }

  /// Unique ID for this audit entry.
  final String id;

  /// When the action occurred.
  final DateTime timestamp;

  /// What action was performed (e.g. "config.push", "db.migrate", "instruct.apply").
  final String action;

  /// The SSH identity of the person who performed the action.
  final SshIdentity actor;

  /// How the action was performed (e.g. "cli", "admin", "api").
  final String? via;

  /// Unified diff of config changes (if applicable).
  final String? diff;

  /// The natural-language instruction that triggered this action (if via AI).
  final String? instructionPrompt;

  /// Arbitrary metadata.
  final Map<String, dynamic> metadata;

  Map<String, dynamic> toMap() => {
    'id': id,
    'timestamp': timestamp.toIso8601String(),
    'action': action,
    'actor': actor.toMap(),
    if (via != null) 'via': via,
    if (diff != null) 'diff': diff,
    if (instructionPrompt != null) 'instruction_prompt': instructionPrompt,
    if (metadata.isNotEmpty) 'metadata': metadata,
  };
}
