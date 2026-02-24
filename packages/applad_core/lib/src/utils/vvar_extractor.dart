library;

import 'dart:io';
import 'package:path/path.dart' as p;

/// A utility to extract `${VAR_NAME}` references from Applad config files.
final class VVarExtractor {
  const VVarExtractor();

  /// Scans all `.yaml` files in the given [rootPath] and returns a unique set
  /// of all environment variable names referenced via `${VAR}`.
  Set<String> extractFromPath(String rootPath) {
    final vars = <String>{};
    final dir = Directory(rootPath);
    if (!dir.existsSync()) return vars;

    final regExp = RegExp(r'\$\{([A-Z0-9_]+)\}');

    for (final file in dir.listSync(recursive: true)) {
      if (file is File && p.extension(file.path) == '.yaml') {
        final content = file.readAsStringSync();
        final matches = regExp.allMatches(content);
        for (final match in matches) {
          final varName = match.group(1);
          if (varName != null) {
            vars.add(varName);
          }
        }
      }
    }

    return vars;
  }
}
