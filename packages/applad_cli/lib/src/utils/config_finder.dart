import 'dart:io';
import 'package:path/path.dart' as p;
import 'output.dart';

/// Walks upward from [startDir] to find the root `applad.yaml`.
final class ConfigFinder {
  const ConfigFinder();

  /// Searches from [startDir] upward until it finds `applad.yaml`.
  /// Returns the directory containing `applad.yaml`, or null if not found.
  String? findRoot({String? startDir}) {
    var current = startDir ?? Directory.current.path;

    // Walk up the tree
    while (true) {
      final candidate = p.join(current, 'applad.yaml');
      if (File(candidate).existsSync()) {
        return current;
      }

      final parent = p.dirname(current);
      if (parent == current) {
        // Reached filesystem root
        return null;
      }
      current = parent;
    }
  }

  String requireRoot({String? startDir}) {
    final root = findRoot(startDir: startDir);
    if (root == null) {
      throw StateError(
        'Could not find applad.yaml. '
        'Run this command from inside an Applad project, '
        'or run `applad init` to create one.',
      );
    }
    return root;
  }

  /// Searches from [startDir] upward until it finds `project.yaml`.
  /// Returns the directory containing `project.yaml`, or null if not found.
  static Directory? findProjectRoot({String? startDir}) {
    var current = Directory(startDir ?? Directory.current.path);

    while (true) {
      final candidate = File(p.join(current.path, 'project.yaml'));
      if (candidate.existsSync()) {
        return current;
      }

      final parent = current.parent;
      if (parent.path == current.path) {
        return null;
      }
      current = parent;
    }
  }

  /// Interactive discovery that either returns the current project root or
  /// prompts the user to select one of the projects found in sub-directories.
  static Directory? discoverProjectRoot() {
    // 1. Try local discovery (upward)
    final localRoot = findProjectRoot();
    if (localRoot != null) return localRoot;

    // 2. Scan sub-directories for projects
    Output.info('Scanning for Applad projects in ${Directory.current.path}...');
    final projects = <Directory>[];

    void scan(Directory dir, int depth) {
      if (depth > 5) return;
      final name = p.basename(dir.path);
      if (name.startsWith('.') ||
          name == 'node_modules' ||
          name == 'vendor' ||
          name == 'build' ||
          name == 'bin') {
        return;
      }

      try {
        for (final entity in dir.listSync(followLinks: false)) {
          if (entity is File && p.basename(entity.path) == 'project.yaml') {
            projects.add(dir);
            // Don't recurse further if we found a project
            return;
          } else if (entity is Directory) {
            scan(entity, depth + 1);
          }
        }
      } catch (_) {}
    }

    scan(Directory.current, 0);

    if (projects.isEmpty) return null;

    if (projects.length == 1) {
      final project = projects.first;
      final relativePath = p.relative(project.path);
      Output.blank();
      Output.info('Found 1 project: ${p.basename(project.path)}');
      Output.kv('Path', relativePath);
      final useIt = Output.confirm(
        'Use this project?',
        defaultValue: false,
      );
      return useIt ? projects.first : null;
    }

    Output.blank();
    Output.info('Select a project (or [0] to cancel):');
    for (var i = 0; i < projects.length; i++) {
      final project = projects[i];
      final label = p.basename(project.path);
      final path = p.relative(project.path);
      stdout.writeln('  [${i + 1}] $label (${_dim(path)})');
    }

    final choiceInput = Output.prompt('Selection', defaultValue: '0');
    final choice = int.tryParse(choiceInput) ?? 0;

    if (choice < 1 || choice > projects.length) {
      Output.info('Operation cancelled.');
      return null;
    }

    return projects[choice - 1];
  }

  static String _dim(String s) {
    return stdout.supportsAnsiEscapes ? '\x1B[2m$s\x1B[0m' : s;
  }
}
