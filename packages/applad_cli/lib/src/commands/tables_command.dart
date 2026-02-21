import 'package:args/command_runner.dart';
import '../utils/output.dart';

/// `applad tables` — Manage database tables and schemas.
final class TablesCommand extends Command<void> {
  @override
  String get name => 'tables';

  @override
  String get description => 'Manage database tables and schemas.';

  TablesCommand() {
    // Add subcommands here later, e.g., list, generate, validate, show, diff
  }

  @override
  Future<void> run() async {
    // If no subcommand is provided, this will print help by default if we don't have run implemented,
    // but the subcommands would be required usually. If no subcommands, just print a placeholder.
    Output.info(
        'applad tables - Manage database tables and schemas (coming in Phase 2)');
  }
}
