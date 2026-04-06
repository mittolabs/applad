import 'client.dart';

/// Teams service — manage teams and memberships.
class Teams {
  final ApplAdServer _client;

  Teams(this._client);

  /// Create a team.
  Future<Map<String, dynamic>> create({
    required String name,
    String? teamId,
    List<String>? roles,
  }) async {
    return _client.post('/v1/teams', data: {
      'name': name,
      'teamId': teamId ?? 'unique()',
      if (roles != null) 'roles': roles,
    });
  }

  /// List teams.
  Future<Map<String, dynamic>> list({
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
    return _client.get('/v1/teams$qs');
  }

  /// Get a team by ID.
  Future<Map<String, dynamic>> get(String teamId) async {
    return _client.get('/v1/teams/$teamId');
  }

  /// Update a team.
  Future<Map<String, dynamic>> update(String teamId, String name) async {
    return _client.put('/v1/teams/$teamId', data: {
      'name': name,
    });
  }

  /// Delete a team.
  Future<Map<String, dynamic>> delete(String teamId) async {
    return _client.delete('/v1/teams/$teamId');
  }

  // --- Memberships ---

  /// Create a membership.
  Future<Map<String, dynamic>> createMembership({
    required String teamId,
    required String email,
    List<String>? roles,
  }) async {
    return _client.post('/v1/teams/$teamId/memberships', data: {
      'email': email,
      'roles': roles ?? [],
    });
  }

  /// List memberships for a team.
  Future<Map<String, dynamic>> listMemberships(String teamId) async {
    return _client.get('/v1/teams/$teamId/memberships');
  }

  /// Delete a membership.
  Future<Map<String, dynamic>> deleteMembership(
      String teamId, String membershipId) async {
    return _client.delete('/v1/teams/$teamId/memberships/$membershipId');
  }
}
