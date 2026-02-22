import 'dart:convert';
import 'dart:io';
import 'package:path/path.dart' as p;
import '../utils/output.dart';

/// Manages workspace trust state for the Applad CLI.
final class TrustManager {
  static final String _trustFile = p.join(
    Platform.environment['HOME'] ?? '',
    '.applad',
    'trust.json',
  );

  /// Checks if the current directory or any of its parents are trusted.
  static bool isTrusted() {
    final trustedPaths = _loadTrust();
    var current = Directory.current.absolute.path;

    while (true) {
      if (trustedPaths.contains(current)) return true;
      final parent = p.dirname(current);
      if (parent == current) break;
      current = parent;
    }

    return false;
  }

  /// Adds a path to the trust list.
  static void trustPath(String path) {
    final trustedPaths = _loadTrust();
    trustedPaths.add(Directory(path).absolute.path);
    _saveTrust(trustedPaths);
  }

  /// Interactive check that prompts for trust if not already trusted.
  static bool ensureTrusted() {
    if (isTrusted()) return true;

    final currentDir = Directory.current.absolute.path;
    final parentDir = p.dirname(currentDir);
    final dirName = p.basename(currentDir);
    final parentName = p.basename(parentDir);

    Output.blank();
    Output.header('Security Check');
    stdout.writeln(_yellow('Do you trust this folder?'));
    stdout.writeln(_dim(
        'Trusting a folder allows Applad to execute resource creation and coordination commands.'));
    stdout.writeln(_dim(
        'This is a security feature to prevent accidental execution in untrusted directories.'));
    Output.blank();

    stdout.writeln('  [1] Trust this folder ($dirName)');
    if (parentName.isNotEmpty && parentDir != parentDir.substring(0, 1)) {
      stdout.writeln('  [2] Trust parent folder ($parentName)');
    }
    stdout.writeln('  [3] Don\'t trust (Limited mode)');
    Output.blank();

    final choice = Output.prompt('Selection', defaultValue: '3');

    if (choice == '1') {
      trustPath(currentDir);
      Output.success('Folder trusted: $dirName');
      return true;
    } else if (choice == '2') {
      trustPath(parentDir);
      Output.success('Folder trusted: $parentName');
      return true;
    }

    Output.warning('Running in untrusted/limited mode.');
    return false;
  }

  static Set<String> _loadTrust() {
    final file = File(_trustFile);
    if (!file.existsSync()) return {};

    try {
      final content = file.readAsStringSync();
      final List<dynamic> json = jsonDecode(content);
      return json.cast<String>().toSet();
    } catch (_) {
      return {};
    }
  }

  static void _saveTrust(Set<String> paths) {
    try {
      final file = File(_trustFile);
      if (!file.parent.existsSync()) {
        file.parent.createSync(recursive: true);
      }
      file.writeAsStringSync(jsonEncode(paths.toList()));
    } catch (e) {
      Output.error('Failed to save trust state: $e');
    }
  }

  static String _yellow(String s) {
    return stdout.supportsAnsiEscapes ? '\x1B[33m$s\x1B[0m' : s;
  }

  static String _dim(String s) {
    return stdout.supportsAnsiEscapes ? '\x1B[2m$s\x1B[0m' : s;
  }
}
