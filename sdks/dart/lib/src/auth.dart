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
  ///
  /// On success the returned session carries a `secret` (a JWT). This applies it
  /// as the bearer token for subsequent requests, so auth works the same on
  /// mobile and web without depending on a cookie the platform may not keep.
  /// Persist `secret` yourself to stay signed in across launches, and restore it
  /// with [Applad.setJWT].
  Future<Map<String, dynamic>> createEmailSession({
    required String email,
    required String password,
  }) async {
    final res = await _dio.post('/v1/account/sessions/email', data: {
      'email': email,
      'password': password,
    });
    _applySecret(res.data);
    return res.data;
  }

  /// Create an anonymous session.
  Future<Map<String, dynamic>> createAnonymousSession() async {
    final res = await _dio.post('/v1/account/sessions/anonymous');
    _applySecret(res.data);
    return res.data;
  }

  /// The URL that starts an OAuth2 sign-in for [provider].
  ///
  /// Open it in a browser or an in-app web view. Applad handles the provider
  /// handshake and redirects to [success] or [failure] with the session
  /// established; both must be relative paths, which is what stops this being
  /// an open redirect. Providers are configured per project, so one that is not
  /// configured fails at the redirect rather than here.
  String oauth2SessionUrl(
    String provider, {
    String? success,
    String? failure,
  }) {
    final query = <String, String>{
      if (success != null) 'success': success,
      if (failure != null) 'failure': failure,
    };
    return _url('/v1/account/sessions/oauth/$provider', query);
  }

  /// Adopts a session secret obtained outside the SDK — the value an OAuth
  /// redirect hands back, for instance — so later calls are authenticated.
  void setSession(String secret) {
    _dio.options.headers['Authorization'] = 'Bearer $secret';
  }

  /// Joins the client's base URL to [path] with an optional query.
  String _url(String path, Map<String, String> query) {
    var base = _dio.options.baseUrl;
    while (base.endsWith('/')) {
      base = base.substring(0, base.length - 1);
    }
    if (query.isEmpty) return '$base$path';
    final encoded = query.entries
        .map((e) => '${e.key}=${Uri.encodeQueryComponent(e.value)}')
        .join('&');
    return '$base$path?$encoded';
  }

  /// Applies a freshly minted session secret as the bearer token, if present.
  void _applySecret(dynamic data) {
    if (data is Map && data['secret'] is String && (data['secret'] as String).isNotEmpty) {
      _dio.options.headers['Authorization'] = 'Bearer ${data['secret']}';
    }
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

  /// Delete all sessions for the current user (logout everywhere).
  Future<void> deleteSessions() async {
    await _dio.delete('/v1/account/sessions');
    _dio.options.headers.remove('Authorization');
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
