import 'dart:io';
import 'package:path/path.dart' as p;

/// Manages local session state for the Applad CLI.
final class SessionManager {
  static final String _sessionFile = p.join(
    Platform.environment['HOME'] ?? '',
    '.applad',
    'session',
  );

  /// Checks if a session file exists.
  static bool isLoggedIn() {
    return File(_sessionFile).existsSync();
  }

  /// Creates a mock session file.
  static void login() {
    final file = File(_sessionFile);
    if (!file.parent.existsSync()) {
      file.parent.createSync(recursive: true);
    }
    file.writeAsStringSync('authenticated');
  }

  /// Deletes the session file.
  static void logout() {
    final file = File(_sessionFile);
    if (file.existsSync()) {
      file.deleteSync();
    }
  }
}
