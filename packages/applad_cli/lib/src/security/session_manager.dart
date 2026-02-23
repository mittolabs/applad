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

  /// Returns the current user's email if logged in.
  static String? get currentUserEmail {
    final file = File(_sessionFile);
    if (!file.existsSync()) return null;
    final content = file.readAsStringSync();
    if (content == 'authenticated') return 'guest@applad.dev';
    return content;
  }

  /// Creates a mock session file.
  static void login({String email = 'guest@applad.dev'}) {
    final file = File(_sessionFile);
    if (!file.parent.existsSync()) {
      file.parent.createSync(recursive: true);
    }
    file.writeAsStringSync(email);
  }

  /// Deletes the session file.
  static void logout() {
    final file = File(_sessionFile);
    if (file.existsSync()) {
      file.deleteSync();
    }
  }
}
