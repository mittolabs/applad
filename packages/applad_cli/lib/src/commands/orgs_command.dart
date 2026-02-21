import 'package:args/command_runner.dart';
import '../utils/output.dart';

/// `applad orgs` — Manage organizations on this instance.
final class OrgsCommand extends Command<void> {
  @override
  String get name => 'orgs';

  @override
  String get description => 'Manage organizations on this instance.';

  OrgsCommand() {
    // Add subcommands here later, e.g., list, create, delete, switch, members, keys
  }

  @override
  Future<void> run() async {
    Output.info('applad orgs - Manage organizations (coming in Phase 2)');
  }
}
