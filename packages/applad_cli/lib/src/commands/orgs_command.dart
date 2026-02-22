import 'package:args/command_runner.dart';
import 'orgs/list_command.dart';

/// `applad orgs` — Manage organizations on this instance.
final class OrgsCommand extends Command<void> {
  @override
  String get name => 'orgs';

  @override
  String get description => 'Manage organizations on this instance.';

  OrgsCommand() {
    addSubcommand(OrgsListCommand());
  }

  @override
  Future<void> run() async {
    printUsage();
  }
}
