library;

import 'dart:io';
import 'package:yaml/yaml.dart';
import 'package:path/path.dart' as p;
import '../errors/applad_error.dart';

/// Walks a config directory tree and loads all YAML files.
final class ConfigLoader {
  const ConfigLoader();

  /// Loads a single YAML file and returns its content as a Map.
  Map<String, dynamic> loadFile(String filePath) {
    final file = File(filePath);
    if (!file.existsSync()) {
      throw ConfigError('Config file not found: $filePath', filePath: filePath);
    }
    try {
      final content = file.readAsStringSync();
      if (content.trim().isEmpty) return {};
      final yaml = loadYaml(content);
      if (yaml == null) return {};
      if (yaml is! Map) {
        throw ConfigError('Expected a YAML map, got ${yaml.runtimeType}',
            filePath: filePath);
      }
      return _deepConvert(yaml) as Map<String, dynamic>;
    } on YamlException catch (e) {
      throw ConfigError('Failed to parse YAML: ${e.message}',
          filePath: filePath, cause: e);
    }
  }

  /// Loads all YAML files from a directory (non-recursive).
  Map<String, Map<String, dynamic>> loadDirectory(String dirPath) {
    final dir = Directory(dirPath);
    if (!dir.existsSync()) return {};

    final results = <String, Map<String, dynamic>>{};
    for (final entity in dir.listSync()) {
      if (entity is File &&
          (entity.path.endsWith('.yaml') || entity.path.endsWith('.yml'))) {
        final name = p.basenameWithoutExtension(entity.path);
        results[name] = loadFile(entity.path);
      }
    }
    return results;
  }

  /// Recursively loads all YAML files from a directory, pruning hidden paths.
  Map<String, Map<String, dynamic>> loadDirectoryRecursive(String dirPath) {
    final rootDir = Directory(dirPath);
    if (!rootDir.existsSync()) return {};

    final results = <String, Map<String, dynamic>>{};
    final directoriesToVisit = <Directory>[rootDir];

    while (directoriesToVisit.isNotEmpty) {
      final currentDir = directoriesToVisit.removeLast();
      File('debug_traverse.txt').writeAsStringSync(
          'VISITING: \${currentDir.path}\\n',
          mode: FileMode.append);

      try {
        for (final entity in currentDir.listSync(followLinks: false)) {
          final name = p.basename(entity.path);

          // Prune standard caches and hidden folders from traversal entirely!
          if (name.startsWith('.') || name == 'build') {
            continue;
          }

          if (entity is Directory) {
            directoriesToVisit.add(entity);
          } else if (entity is File &&
              (name.endsWith('.yaml') || name.endsWith('.yml'))) {
            final relativePath = p.relative(entity.path, from: dirPath);
            final keyName = relativePath.replaceAll(RegExp(r'\.ya?ml$'), '');
            results[keyName] = loadFile(entity.path);
          }
        }
      } catch (_) {
        // Skip inaccessible directories during traversal
      }
    }

    return results;
  }

  dynamic _deepConvert(dynamic value) {
    if (value is YamlMap) {
      return {
        for (final e in value.entries) e.key.toString(): _deepConvert(e.value)
      };
    } else if (value is YamlList) {
      return [for (final item in value) _deepConvert(item)];
    }
    return value;
  }
}
