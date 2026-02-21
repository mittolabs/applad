import 'package:args/command_runner.dart';
import '../utils/output.dart';

final class DbCommand extends Command<void> {
  DbCommand() {
    addSubcommand(_DbSubcommand('migrate', 'Run pending database migrations.'));
    addSubcommand(
        _DbSubcommand('seed', 'Seed the database with initial data.'));
    addSubcommand(_DbSubcommand(
        'reset', 'Reset the database (drop + recreate + migrate).'));
    addSubcommand(_DbSubcommand('status', 'Show migration status.'));
  }

  @override
  String get name => 'db';
  @override
  String get description => 'Database management commands.';
}

final class _DbSubcommand extends Command<void> {
  _DbSubcommand(this._name, this._description);
  final String _name;
  final String _description;

  @override
  String get name => _name;
  @override
  String get description => '$_description (Phase 2)';

  @override
  Future<void> run() async =>
      Output.info('applad db $name — coming in Phase 2');
}
