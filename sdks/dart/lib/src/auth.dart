import 'package:dio/dio.dart';

/// Client-side authentication service.
class Auth {
  final Dio _dio;

  Auth(this._dio);

  /// Create a new user account.
  Future<Map<String, dynamic>> createAccount({
    required String email,
    required String password,
    String? userId,
    String? name,
  }) async {
    final res = await _dio.post('/v1/account', data: {
      'userId': userId ?? 'unique()',
      'email': email,
      'password': password,
      if (name != null) 'name': name,
    });
    return res.data;
  }

  /// Get the currently logged-in user.
  Future<Map<String, dynamic>> getAccount() async {
    final res = await _dio.get('/v1/account');
    return res.data;
  }

  /// Update the current user's display name.
  Future<Map<String, dynamic>> updateName(String name) async {
    final res = await _dio.patch('/v1/account/name', data: {'name': name});
    return res.data;
  }

  /// Update the current user's email (requires password).
  Future<Map<String, dynamic>> updateEmail({
    required String email,
    required String password,
  }) async {
    final res = await _dio.patch('/v1/account/email', data: {
      'email': email,
      'password': password,
    });
    return res.data;
  }

  /// Update the current user's password.
  Future<Map<String, dynamic>> updatePassword({
    required String password,
    String? oldPassword,
  }) async {
    final res = await _dio.patch('/v1/account/password', data: {
      'password': password,
      if (oldPassword != null) 'oldPassword': oldPassword,
    });
    return res.data;
  }

  /// Update the current user's preferences.
  Future<Map<String, dynamic>> updatePrefs(Map<String, dynamic> prefs) async {
    final res = await _dio.patch('/v1/account/prefs', data: {'prefs': prefs});
    return res.data;
  }

  /// Delete the current user's account.
  Future<void> deleteAccount() async {
    await _dio.delete('/v1/account');
  }

  /// Create an email session (login).
  Future<Map<String, dynamic>> createEmailSession({
    required String email,
    required String password,
  }) async {
    final res = await _dio.post('/v1/account/sessions/email', data: {
      'email': email,
      'password': password,
    });
    return res.data;
  }

  /// Create an anonymous session.
  Future<Map<String, dynamic>> createAnonymousSession() async {
    final res = await _dio.post('/v1/account/sessions/anonymous');
    return res.data;
  }

  /// List all sessions for the current user.
  Future<Map<String, dynamic>> listSessions() async {
    final res = await _dio.get('/v1/account/sessions');
    return res.data;
  }

  /// Get a session by ID.
  Future<Map<String, dynamic>> getSession(String sessionId) async {
    final res = await _dio.get('/v1/account/sessions/$sessionId');
    return res.data;
  }

  /// Delete a specific session.
  Future<void> deleteSession(String sessionId) async {
    await _dio.delete('/v1/account/sessions/$sessionId');
  }

  /// Delete all sessions for the current user.
  Future<void> deleteSessions() async {
    await _dio.delete('/v1/account/sessions');
  }

  /// Get a short-lived JWT for the current user.
  Future<String> getJWT() async {
    final res = await _dio.post('/v1/account/jwt');
    return res.data['jwt'];
  }
}

/// Server-side user management service.
class Users {
  final Dio _dio;

  Users(this._dio);

  /// Create a user (server-side).
  Future<Map<String, dynamic>> create({
    String? userId,
    String? email,
    String? phone,
    String? password,
    String? name,
  }) async {
    final res = await _dio.post('/v1/users', data: {
      'userId': userId ?? 'unique()',
      if (email != null) 'email': email,
      if (phone != null) 'phone': phone,
      if (password != null) 'password': password,
      if (name != null) 'name': name,
    });
    return res.data;
  }

  /// List users with optional search and pagination.
  Future<Map<String, dynamic>> list({
    int? limit,
    int? offset,
    String? search,
  }) async {
    final res = await _dio.get('/v1/users', queryParameters: {
      if (limit != null) 'limit': limit,
      if (offset != null) 'offset': offset,
      if (search != null) 'search': search,
    });
    return res.data;
  }

  /// Get a user by ID.
  Future<Map<String, dynamic>> get(String userId) async {
    final res = await _dio.get('/v1/users/$userId');
    return res.data;
  }

  /// Delete a user.
  Future<void> delete(String userId) async {
    await _dio.delete('/v1/users/$userId');
  }

  /// Update a user's status.
  Future<Map<String, dynamic>> updateStatus(String userId, bool status) async {
    final res = await _dio.patch('/v1/users/$userId/status', data: {
      'status': status,
    });
    return res.data;
  }

  /// Update a user's email (admin, no password required).
  Future<Map<String, dynamic>> updateEmail(String userId, String email) async {
    final res = await _dio.patch('/v1/users/$userId/email', data: {
      'email': email,
    });
    return res.data;
  }

  /// Update a user's name.
  Future<Map<String, dynamic>> updateName(String userId, String name) async {
    final res = await _dio.patch('/v1/users/$userId/name', data: {
      'name': name,
    });
    return res.data;
  }

  /// List sessions for a user.
  Future<Map<String, dynamic>> listSessions(String userId) async {
    final res = await _dio.get('/v1/users/$userId/sessions');
    return res.data;
  }

  /// Delete all sessions for a user.
  Future<void> deleteSessions(String userId) async {
    await _dio.delete('/v1/users/$userId/sessions');
  }
}
