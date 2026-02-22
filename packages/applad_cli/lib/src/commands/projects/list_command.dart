import 'dart:io';
import 'package:args/command_runner.dart';
import 'package:path/path.dart' as p;
import '../../utils/output.dart';
import '../../utils/config_finder.dart';

/// `applad projects list` — List all projects in the current workspace.
final class ProjectsListCommand extends Command<void> {
  @override
  String get name => 'list';

  @override
  String get description => 'List all projects in the current workspace.';

  @override
  Future<void> run() async {
    final finder = const ConfigFinder();
    final String rootPath;
    try {
      rootPath = finder.requireRoot();
    } catch (e) {
      Output.error(e.toString());
      return;
    }

    final orgsDir = Directory(p.join(rootPath, 'orgs'));
    if (!orgsDir.existsSync()) {
      Output.info('No projects found (orgs/ directory missing).');
      return;
    }

    final projects = <_ProjectInfo>[];

    final orgDirs = orgsDir
        .listSync()
        .whereType<Directory>()
        .where((d) => File(p.join(d.path, 'org.yaml')).existsSync());

    for (final orgDir in orgDirs) {
      final orgId = p.basename(orgDir.path);
      final projectsDir = Directory(p.join(orgDir.path, 'projects'));

      if (!projectsDir.existsSync()) continue;

      final projectDirs = projectsDir
          .listSync()
          .whereType<Directory>()
          .where((d) => File(p.join(d.path, 'project.yaml')).existsSync());

      for (final projectDir in projectDirs) {
        projects.add(_ProjectInfo(
          orgId: orgId,
          projectId: p.basename(projectDir.path),
        ));
      }
    }

    if (projects.isEmpty) {
      Output.info('No projects found.');
      return;
    }

    Output.header('Projects');
    for (final project in projects) {
      Output.info('• ${project.orgId} / ${project.projectId}');
    }
    Output.info('');
    Output.info('Total: ${projects.length} projects');
  }
}

class _ProjectInfo {
  final String orgId;
  final String projectId;

  _ProjectInfo({required this.orgId, required this.projectId});
}
