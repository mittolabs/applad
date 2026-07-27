import 'package:dio/dio.dart';

/// Client-side teams service — create teams and manage their memberships as the
/// signed-in user.
///
/// A team is Applad's unit of shared access: a row permissioned `read("team:X")`
/// is visible to exactly the members of team `X`. That makes teams the natural
/// backing for anything group-scoped — a workspace, a channel, a shared
/// document — where "these people, and only these people, may see this."
class Teams {
  final Dio _dio;

  Teams(this._dio);

  /// Create a team. The caller becomes its first, already-joined owner.
  Future<Map<String, dynamic>> create({
    required String name,
    String? teamId,
    List<String>? roles,
  }) async {
    final res = await _dio.post('/v1/teams', data: {
      'name': name,
      'teamId': teamId ?? 'unique()',
      if (roles != null) 'roles': roles,
    });
    return res.data;
  }

  /// List teams the current user can see.
  Future<Map<String, dynamic>> list({int? limit, int? offset, String? search}) async {
    final res = await _dio.get('/v1/teams', queryParameters: {
      if (limit != null) 'limit': limit,
      if (offset != null) 'offset': offset,
      if (search != null) 'search': search,
    });
    return res.data;
  }

  /// Get a team by ID.
  Future<Map<String, dynamic>> get(String teamId) async {
    final res = await _dio.get('/v1/teams/$teamId');
    return res.data;
  }

  /// Rename a team.
  Future<Map<String, dynamic>> update(String teamId, String name) async {
    final res = await _dio.put('/v1/teams/$teamId', data: {'name': name});
    return res.data;
  }

  /// Delete a team.
  Future<void> delete(String teamId) async {
    await _dio.delete('/v1/teams/$teamId');
  }

  /// Invite someone to a team by email. The returned membership carries a
  /// one-time `secret`; hand it to the invitee (e.g. as a join link) and they
  /// redeem it with [acceptMembership]. A membership is not active until joined.
  Future<Map<String, dynamic>> createMembership(
    String teamID, {
    required String email,
    List<String>? roles,
  }) async {
    final res = await _dio.post('/v1/teams/$teamID/memberships', data: {
      'email': email,
      if (roles != null) 'roles': roles,
    });
    return res.data;
  }

  /// List a team's memberships.
  Future<Map<String, dynamic>> listMemberships(String teamID) async {
    final res = await _dio.get('/v1/teams/$teamID/memberships');
    return res.data;
  }

  /// Accept an invite: binds the current user to the membership and marks it
  /// joined. The joining identity comes from the session, so the `secret` is all
  /// the invitee needs to supply.
  Future<Map<String, dynamic>> acceptMembership(
    String teamID,
    String membershipID,
    String secret,
  ) async {
    final res = await _dio.patch(
      '/v1/teams/$teamID/memberships/$membershipID/status',
      data: {'secret': secret},
    );
    return res.data;
  }

  /// Remove a membership from a team.
  Future<void> deleteMembership(String teamID, String membershipID) async {
    await _dio.delete('/v1/teams/$teamID/memberships/$membershipID');
  }
}
