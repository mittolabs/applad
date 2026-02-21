library;

import '../errors/applad_error.dart';
import '../config/database_config.dart';
import 'config_merger.dart';

/// Validates a fully merged [ApplAdConfig] tree.
final class ConfigValidator {
  const ConfigValidator();

  /// Validates the config and returns a list of violations.
  /// Throws [ValidationError] if there are any error-level violations.
  List<ValidationViolation> validate(ApplAdConfig config) {
    final violations = <ValidationViolation>[];

    // Instance validations
    _validateInstance(config, violations);

    // Org validations
    _validateOrg(config, violations);

    // Project validations
    _validateProject(config, violations);

    // Table validations
    _validateTables(config, violations);

    // Auth validations
    if (config.auth != null) _validateAuth(config, violations);

    // Database validations
    if (config.database != null) _validateDatabase(config, violations);

    // Check for error-level violations
    final errors =
        violations.where((v) => v.severity == ViolationSeverity.error).toList();
    if (errors.isNotEmpty) {
      throw ValidationError(
        'Config validation failed with ${errors.length} error(s)',
        violations: violations,
      );
    }

    return violations;
  }

  void _validateInstance(
      ApplAdConfig config, List<ValidationViolation> violations) {
    if (config.instance.version.isEmpty) {
      violations.add(const ValidationViolation(
        path: 'applad.yaml > version',
        message: 'Version is required',
      ));
    }
  }

  void _validateOrg(ApplAdConfig config, List<ValidationViolation> violations) {
    if (config.org.id.isEmpty) {
      violations.add(const ValidationViolation(
        path: 'org.yaml > id',
        message: 'Org ID is required',
      ));
    }
    if (config.org.name.isEmpty) {
      violations.add(const ValidationViolation(
        path: 'org.yaml > name',
        message: 'Org name is required',
      ));
    }
  }

  void _validateProject(
      ApplAdConfig config, List<ValidationViolation> violations) {
    if (config.project.id.isEmpty) {
      violations.add(const ValidationViolation(
        path: 'project.yaml > id',
        message: 'Project ID is required',
      ));
    }
    if (config.project.name.isEmpty) {
      violations.add(const ValidationViolation(
        path: 'project.yaml > name',
        message: 'Project name is required',
      ));
    }
  }

  void _validateTables(
      ApplAdConfig config, List<ValidationViolation> violations) {
    for (final table in config.tables) {
      if (table.fields.isEmpty) {
        violations.add(ValidationViolation(
          path: 'tables/${table.name} > fields',
          message: 'Table "${table.name}" has no fields defined',
          severity: ViolationSeverity.warning,
        ));
      }
      for (final field in table.fields) {
        if (field.name.isEmpty || field.type.isEmpty) {
          violations.add(ValidationViolation(
            path: 'tables/${table.name} > fields',
            message: 'Field in table "${table.name}" is missing name or type',
          ));
        }
      }
    }
  }

  void _validateAuth(
      ApplAdConfig config, List<ValidationViolation> violations) {
    final auth = config.auth!;
    if (auth.providers.isEmpty) {
      violations.add(const ValidationViolation(
        path: 'auth/auth.yaml > providers',
        message: 'No auth providers configured',
        severity: ViolationSeverity.warning,
      ));
    }
  }

  void _validateDatabase(
      ApplAdConfig config, List<ValidationViolation> violations) {
    final db = config.database!;
    if (db.adapter != DatabaseAdapter.sqlite) {
      if (db.connectionStringRef == null && db.host == null) {
        violations.add(const ValidationViolation(
          path: 'database/database.yaml',
          message: 'Non-SQLite database requires connection_string or host',
        ));
      }
    }
  }
}
