import 'dart:io';
import 'package:path/path.dart' as p;

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
}
