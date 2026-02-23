import 'dart:convert';
import 'dart:io';
import 'package:path/path.dart' as p;

class WorkspaceManager {
  static final String _workspacesFile = p.join(
    Platform.environment['HOME'] ?? '',
    '.applad',
    'workspaces.json',
  );

  /// Registers a workspace path in the local cache.
  static void register(String path) {
    final workspaces = list();
    final normalized = p.normalize(p.absolute(path));
    if (!workspaces.contains(normalized)) {
      workspaces.add(normalized);
      _save(workspaces);
    }
  }

  /// Removes a workspace path from the local cache.
  static void unregister(String path) {
    final workspaces = list();
    final normalized = p.normalize(p.absolute(path));
    if (workspaces.remove(normalized)) {
      _save(workspaces);
    }
  }

  /// Returns a list of all registered workspace paths, purging invalid ones.
  static List<String> list() {
    final file = File(_workspacesFile);
    if (!file.existsSync()) return [];

    try {
      final content = file.readAsStringSync();
      final data = json.decode(content);
      if (data is List) {
        final paths = data.cast<String>();
        final validPaths = <String>[];
        bool changed = false;

        for (final path in paths) {
          final dir = Directory(path);
          final appladYaml = File(p.join(path, 'applad.yaml'));

          if (dir.existsSync() && appladYaml.existsSync()) {
            validPaths.add(path);
          } else {
            changed = true;
          }
        }

        if (changed) {
          _save(validPaths);
        }
        return validPaths;
      }
    } catch (_) {
      // Corrupted file, reset
    }
    return [];
  }

  /// Scans a directory for Applad projects (containing applad.yaml).
  static Future<List<String>> discover(String rootPath,
      {int maxDepth = 4}) async {
    final dir = Directory(rootPath);
    if (!dir.existsSync()) return [];

    final results = <String>[];
    try {
      final stream = dir.list(recursive: true, followLinks: false);
      await for (final entity in stream.handleError((error) {
        // Skip paths we don't have permission to access (common on macOS)
        if (error is FileSystemException) {
          // ignore error
        }
      })) {
        if (entity is File && p.basename(entity.path) == 'applad.yaml') {
          final absolutePath = p.absolute(entity.path);
          final segments = p.split(absolutePath);

          // Filter out hidden directories (starting with .) and common noise
          bool isNoise = segments.any((s) =>
              (s.startsWith('.') && s != '.' && s != '..') ||
              s == 'node_modules' ||
              s == 'build' ||
              s == 'dist' ||
              s == 'vendor' ||
              s == '__brick__');

          if (!isNoise) {
            results.add(p.dirname(absolutePath));
          }
        }
      }
    } catch (_) {
      // Handle any other catastrophic listing failures
    }
    return results;
  }

  static void _save(List<String> workspaces) {
    final file = File(_workspacesFile);
    if (!file.parent.existsSync()) {
      file.parent.createSync(recursive: true);
    }
    file.writeAsStringSync(json.encode(workspaces));
  }
}
