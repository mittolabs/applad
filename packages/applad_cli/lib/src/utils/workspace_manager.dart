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

  /// Returns a list of all registered workspace paths.
  static List<String> list() {
    final file = File(_workspacesFile);
    if (!file.existsSync()) return [];

    try {
      final content = file.readAsStringSync();
      final data = json.decode(content);
      if (data is List) {
        return data.cast<String>();
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
    await for (final entity in dir.list(recursive: true, followLinks: false)) {
      if (entity is File && p.basename(entity.path) == 'applad.yaml') {
        results.add(p.dirname(p.absolute(entity.path)));
      }
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
