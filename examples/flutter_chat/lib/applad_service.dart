import 'package:applad/applad.dart';
import 'package:shared_preferences/shared_preferences.dart';

import 'config.dart';

/// One Applad client for the whole app, plus the small amount of state a client
/// owns that the SDK deliberately does not: where the session secret is kept,
/// and who is signed in.
///
/// Auth is a bearer token (see the platform audit, G1). The SDK applies it on
/// login and clears it on logout; persisting it across launches is the app's
/// job, because secure storage is platform-specific. We use shared_preferences
/// for brevity — a production app would use the Keychain/Keystore on mobile.
class AppladService {
  AppladService._();
  static final AppladService instance = AppladService._();

  static const _secretKey = 'applad_session_secret';

  final Applad client = Applad(
    endpoint: Config.endpoint,
    projectId: Config.projectId,
  );

  Map<String, dynamic>? currentUser;

  String get userId => currentUser?['\$id']?.toString() ?? '';
  String get userName =>
      (currentUser?['name'] as String?)?.trim().isNotEmpty == true
          ? currentUser!['name'] as String
          : (currentUser?['email'] as String? ?? 'You');

  /// Restore a persisted session, if any, and confirm it still works. Returns
  /// true when a live session was recovered.
  Future<bool> restore() async {
    final prefs = await SharedPreferences.getInstance();
    final secret = prefs.getString(_secretKey);
    if (secret == null || secret.isEmpty) return false;
    client.setJWT(secret);
    try {
      currentUser = await client.auth.getAccount();
      client.realtime.connect();
      return true;
    } catch (_) {
      // Stale or revoked: forget it and fall back to the login screen.
      await prefs.remove(_secretKey);
      return false;
    }
  }

  Future<void> signUp(String name, String email, String password) async {
    await client.auth.createAccount(email: email, password: password, name: name);
    await signIn(email, password);
  }

  Future<void> signIn(String email, String password) async {
    final session = await client.auth.createEmailSession(email: email, password: password);
    final secret = session['secret'] as String?;
    if (secret != null && secret.isNotEmpty) {
      final prefs = await SharedPreferences.getInstance();
      await prefs.setString(_secretKey, secret);
    }
    currentUser = await client.auth.getAccount();
    client.realtime.connect();
  }

  Future<void> signOut() async {
    try {
      await client.auth.deleteSessions();
    } catch (_) {
      // Best effort — clear locally regardless.
    }
    client.realtime.disconnect();
    currentUser = null;
    final prefs = await SharedPreferences.getInstance();
    await prefs.remove(_secretKey);
  }
}
