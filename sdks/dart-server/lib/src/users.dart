import 'client.dart';

/// Server-side user management service.
class Users {
  final ApplAdServer _client;

  Users(this._client);

  /// Create a user.
  Future<Map<String, dynamic>> createUser({
    String? userId,
    String? email,
    String? phone,
    String? password,
    String? name,
  }) async {
    return _client.post('/v1/users', data: {
      'userId': userId ?? 'unique()',
      if (email != null) 'email': email,
      if (phone != null) 'phone': phone,
      if (password != null) 'password': password,
      if (name != null) 'name': name,
    });
  }

  /// Get a user by ID.
  Future<Map<String, dynamic>> getUser(String userId) async {
    return _client.get('/v1/users/$userId');
  }

  /// List users with optional search and pagination.
  Future<Map<String, dynamic>> listUsers({
    int? limit,
    int? offset,
    String? search,
  }) async {
    final query = <String, String>{};
    if (limit != null) query['limit'] = limit.toString();
    if (offset != null) query['offset'] = offset.toString();
    if (search != null) query['search'] = search;
    final qs = query.isNotEmpty
        ? '?${query.entries.map((e) => '${e.key}=${Uri.encodeComponent(e.value)}').join('&')}'
        : '';
    return _client.get('/v1/users$qs');
  }

  /// Delete a user.
  Future<Map<String, dynamic>> deleteUser(String userId) async {
    return _client.delete('/v1/users/$userId');
  }

  /// Update a user's status.
  Future<Map<String, dynamic>> updateUserStatus(
      String userId, bool status) async {
    return _client.patch('/v1/users/$userId/status', data: {
      'status': status,
    });
  }

  /// Update a user's email.
  Future<Map<String, dynamic>> updateEmail(
      String userId, String email) async {
    return _client.patch('/v1/users/$userId/email', data: {
      'email': email,
    });
  }

  /// Update a user's name.
  Future<Map<String, dynamic>> updateName(
      String userId, String name) async {
    return _client.patch('/v1/users/$userId/name', data: {
      'name': name,
    });
  }

  /// List sessions for a user.
  Future<Map<String, dynamic>> listSessions(String userId) async {
    return _client.get('/v1/users/$userId/sessions');
  }

  /// Delete all sessions for a user.
  Future<Map<String, dynamic>> deleteSessions(String userId) async {
    return _client.delete('/v1/users/$userId/sessions');
  }
}
