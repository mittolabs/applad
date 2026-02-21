library;

import '../applad_client.dart';

/// Handles authentication — sign up, sign in, sign out, etc.
final class AuthClient {
  AuthClient({required this.client});

  final ApplAdClient client;

  /// Sign up with email and password.
  /// Returns user data on success.
  Future<Map<String, dynamic>> signUp({
    required String email,
    required String password,
    Map<String, dynamic>? metadata,
  }) async {
    // Phase 2 implementation
    throw UnimplementedError('signUp — available in Phase 2');
  }

  /// Sign in with email and password.
  Future<Map<String, dynamic>> signIn({
    required String email,
    required String password,
  }) async {
    // Phase 2 implementation
    throw UnimplementedError('signIn — available in Phase 2');
  }

  /// Sign out the current user.
  Future<void> signOut() async {
    client.authToken = null;
  }

  /// Get the currently authenticated user.
  Future<Map<String, dynamic>?> getUser() async {
    if (client.authToken == null) return null;
    // Phase 2 implementation
    throw UnimplementedError('getUser — available in Phase 2');
  }

  /// Refresh the current session.
  Future<void> refreshSession() async {
    // Phase 2 implementation
    throw UnimplementedError('refreshSession — available in Phase 2');
  }
}
