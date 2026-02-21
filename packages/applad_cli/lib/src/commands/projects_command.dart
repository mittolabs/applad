import 'package:args/command_runner.dart';
import '../utils/output.dart';

/// `applad projects` — Manage projects within organizations.
final class ProjectsCommand extends Command<void> {
  @override
  String get name => 'projects';

  @override
  String get description => 'Manage projects for an organization.';

  ProjectsCommand() {
    // Add subcommands here later, e.g., list, create, delete, switch, info, clone
  }

  @override
  Future<void> run() async {
    Output.info('applad projects - Manage projects (coming in Phase 2)');
  }
}
