import 'package:args/command_runner.dart';
import 'tables/list_command.dart';

/// `applad tables` — Manage database tables and schemas.
final class TablesCommand extends Command<void> {
  @override
  String get name => 'tables';

  @override
  String get description => 'Manage database tables and schemas.';

  TablesCommand() {
    addSubcommand(TablesListCommand());
  }

  @override
  Future<void> run() async {
    printUsage();
  }
}
