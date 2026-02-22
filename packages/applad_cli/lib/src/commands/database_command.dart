import 'dart:io';
import 'package:args/command_runner.dart';
import 'package:path/path.dart' as p;
import '../utils/output.dart';
import '../utils/config_finder.dart';
import 'tables/list_command.dart';

/// `applad database` — group for database management.
final class DatabaseCommand extends Command<void> {
  DatabaseCommand() {
    addSubcommand(DatabaseTablesCommand());
  }

  @override
  String get name => 'database';
  @override
  String get description => 'Database management.';
}

/// `applad database tables` — group for table management.
final class DatabaseTablesCommand extends Command<void> {
  DatabaseTablesCommand() {
    addSubcommand(DatabaseTablesCreateCommand());
    addSubcommand(TablesListCommand());
  }

  @override
  String get name => 'tables';
  @override
  String get description => 'Table management.';
}

/// `applad database tables create` — guided table creation.
final class DatabaseTablesCreateCommand extends Command<void> {
  @override
  String get name => 'create';
  @override
  String get description => 'Guided creation of a new database table.';

  @override
  Future<void> run() async {
    Output.header('Create Table');

    final projectDir = ConfigFinder.discoverProjectRoot();
    if (projectDir == null) {
      Output.error('No Applad project found.');
      return;
    }

    final projectName = p.basename(projectDir.path);
    Output.info('Selected project: $projectName');

    final tablesDir = Directory('${projectDir.path}/database/tables');
    if (!tablesDir.existsSync()) {
      Output.info(
          'Database tables namespace is not enabled. Creating directory...');
      tablesDir.createSync(recursive: true);
    }

    final name = Output.prompt('Table name', defaultValue: 'users');
    final dbConnection =
        Output.prompt('Database connection', defaultValue: 'primary');

    final yamlFile = File('${tablesDir.path}/$name.yaml');
    if (yamlFile.existsSync()) {
      Output.error('Table "$name" already exists.');
      return;
    }

    final content = '''
# ============================================================
# TABLE: $name
# Generated via applad database tables create
# ============================================================

name: "$name"
database: "$dbConnection"

# Table schema
schema:
  columns:
    - name: "id"
      type: "uuid"
      primary_key: true
      default: "gen_random_uuid()"
    - name: "created_at"
      type: "timestamptz"
      default: "now()"
    - name: "updated_at"
      type: "timestamptz"
      default: "now()"

# Security policies
policies:
  - role: "authenticated"
    allow: ["select"]
    check: "user_id = auth.uid()"
''';

    yamlFile.writeAsStringSync(content);

    Output.blank();
    Output.success('Created table config: ${yamlFile.path}');
    Output.blank();
    Output.nextSteps([
      'Edit ${yamlFile.path} to add columns and define security policies.',
      'Run `applad up` to deploy and run migrations.'
    ]);
  }
}
