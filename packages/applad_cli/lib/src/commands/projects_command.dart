import 'package:args/command_runner.dart';
import 'projects/list_command.dart';

/// `applad projects` — Manage projects within organizations.
final class ProjectsCommand extends Command<void> {
  @override
  String get name => 'projects';

  @override
  String get description => 'Manage projects for an organization.';

  ProjectsCommand() {
    addSubcommand(ProjectsListCommand());
  }

  @override
  Future<void> run() async {
    printUsage();
  }
}
