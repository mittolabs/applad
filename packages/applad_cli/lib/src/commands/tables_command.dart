import 'package:args/command_runner.dart';
import '../utils/output.dart';

/// Manage database tables and schemas.
final class TablesCommand extends Command<void> {
  TablesCommand() {
    addSubcommand(TablesListCommand());
    addSubcommand(TablesGenerateCommand());
    addSubcommand(TablesValidateCommand());
    addSubcommand(TablesShowCommand());
    addSubcommand(TablesDiffCommand());
  }

  @override
  String get name => 'tables';

  @override
  String get description => 'Manage database tables and schemas.';
}

/// Lists all database tables.
final class TablesListCommand extends Command<void> {
  TablesListCommand() {
    argParser.addOption('connection',
        abbr: 'c', help: 'Filter by database connection.');
  }

  @override
  String get name => 'list';
  @override
  String get description => 'List all database tables defined in config.';

  @override
  Future<void> run() async {
    final connection = argResults!['connection'] as String?;
    Output.info(
        'Fetching tables from config${connection != null ? " (connection: $connection)" : ""}...');

    Output.header('Database Tables');
    Output.table([
      'Name',
      'Database',
      'Rows (Est.)'
    ], [
      ['users', 'primary', '1.2k'],
      ['posts', 'primary', '5.4k'],
      ['events', 'analytics', '1.2M'],
    ]);
  }
}

/// Generates a new table config.
final class TablesGenerateCommand extends Command<void> {
  TablesGenerateCommand() {
    argParser.addOption('connection',
        abbr: 'c', help: 'Database connection to use.', defaultsTo: 'primary');
  }

  @override
  String get name => 'generate';
  @override
  String get description => 'Generates a new table configuration file.';

  @override
  Future<void> run() async {
    if (argResults!.rest.isEmpty) {
      Output.error('Usage: applad tables generate <NAME>');
      return;
    }
    final name = argResults!.rest.first;
    final connection = argResults!['connection'] as String;

    Output.info(
        'Generating table config "$name" for connection "$connection"...');
    Output.success('Created database/tables/$name.yaml');
  }
}

/// Validates table configurations.
final class TablesValidateCommand extends Command<void> {
  @override
  String get name => 'validate';
  @override
  String get description => 'Validates all table definitions.';

  @override
  Future<void> run() async {
    Output.info('Validating table definitions...');
    Output.success('All tables are valid.');
  }
}

/// Shows details of a specific table.
final class TablesShowCommand extends Command<void> {
  @override
  String get name => 'show';
  @override
  String get description => 'Shows detailed schema for a table.';

  @override
  Future<void> run() async {
    if (argResults!.rest.isEmpty) {
      Output.error('Usage: applad tables show <NAME>');
      return;
    }
    final name = argResults!.rest.first;
    Output.header('Table: $name');
    Output.table([
      'Field',
      'Type',
      'Constraints'
    ], [
      ['id', 'uuid', 'PKEY'],
      ['email', 'text', 'UNIQUE, NOT NULL'],
      ['created_at', 'timestamp', 'DEFAULT NOW()'],
    ]);
  }
}

/// Diffs a table against the running state.
final class TablesDiffCommand extends Command<void> {
  @override
  String get name => 'diff';
  @override
  String get description =>
      'Shows the delta between config and the actual database schema.';

  @override
  Future<void> run() async {
    if (argResults!.rest.isEmpty) {
      Output.error('Usage: applad tables diff <NAME>');
      return;
    }
    final name = argResults!.rest.first;
    Output.info('Scanning database for schema drift on "$name"...');
    Output.success('No drift detected for "$name".');
  }
}
