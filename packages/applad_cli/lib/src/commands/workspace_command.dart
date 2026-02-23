import 'dart:io';
import 'package:args/command_runner.dart';
import '../utils/output.dart';
import '../utils/workspace_manager.dart';

/// `applad workspace` — manage and discover Applad workspaces across the machine.
final class WorkspaceCommand extends Command<void> {
  WorkspaceCommand() {
    addSubcommand(WorkspaceListCommand());
    addSubcommand(WorkspaceAddCommand());
    addSubcommand(WorkspaceRemoveCommand());
    addSubcommand(WorkspaceDiscoverCommand());
  }

  @override
  String get name => 'workspace';

  @override
  String get description =>
      'Manage and discover Applad workspaces on this machine.';
}

class WorkspaceListCommand extends Command<void> {
  @override
  String get name => 'list';

  @override
  String get description => 'List all cached Applad workspaces.';

  @override
  void run() {
    final workspaces = WorkspaceManager.list();
    if (workspaces.isEmpty) {
      Output.info(
          'No cached workspaces found. Use `applad workspace discover` to scan your machine.');
      return;
    }

    Output.header('Cached Workspaces');
    for (final path in workspaces) {
      final isCurrent = Directory.current.path == path;
      final prefix = isCurrent ? '\x1b[32m* ' : '  ';
      stdout.writeln('$prefix$path\x1b[0m');
    }
  }
}

class WorkspaceAddCommand extends Command<void> {
  @override
  String get name => 'add';

  @override
  String get description => 'Manually add a workspace to the cache.';

  @override
  void run() {
    final path = Directory.current.path;
    if (!File('$path/applad.yaml').existsSync()) {
      Output.error(
          'Current directory is not an Applad project (no applad.yaml found).');
      return;
    }

    WorkspaceManager.register(path);
    Output.success('Workspace added to cache: $path');
  }
}

class WorkspaceRemoveCommand extends Command<void> {
  @override
  String get name => 'remove';

  @override
  String get description => 'Remove a workspace from the cache.';

  @override
  void run() {
    final path = Directory.current.path;
    WorkspaceManager.unregister(path);
    Output.success('Workspace removed from cache: $path');
  }
}

class WorkspaceDiscoverCommand extends Command<void> {
  @override
  String get name => 'discover';

  @override
  String get description => 'Scan the filesystem for Applad projects.';

  @override
  Future<void> run() async {
    final root = Output.prompt('Enter root directory to scan',
        defaultValue: Platform.environment['HOME'] ?? '');

    Output.info('Scanning $root for Applad projects...');
    final results = await WorkspaceManager.discover(root);

    if (results.isEmpty) {
      Output.info('No new projects found.');
      return;
    }

    Output.success('Found ${results.length} project(s):');
    for (final path in results) {
      stdout.writeln('  $path');
      WorkspaceManager.register(path);
    }
  }
}
