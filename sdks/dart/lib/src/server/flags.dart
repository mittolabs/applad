import '../server_client.dart';

/// Flags service — manage and evaluate feature flags (server-side).
class Flags {
  final ApplAdServer _client;

  Flags(this._client);

  // --- CRUD ---

  Future<Map<String, dynamic>> list() async {
    return _client.get('/v1/flags');
  }

  Future<Map<String, dynamic>> create({
    required String key,
    required String name,
    String? description,
    bool? enabled,
    Map<String, dynamic>? variants,
    List<Map<String, dynamic>>? rules,
  }) async {
    return _client.post('/v1/flags', data: {
      'key': key,
      'name': name,
      if (description != null) 'description': description,
      if (enabled != null) 'enabled': enabled,
      if (variants != null) 'variants': variants,
      if (rules != null) 'rules': rules,
    });
  }

  Future<Map<String, dynamic>> get(String key) async {
    return _client.get('/v1/flags/$key');
  }

  Future<Map<String, dynamic>> update({
    required String key,
    String? name,
    String? description,
    Map<String, dynamic>? variants,
    List<Map<String, dynamic>>? rules,
  }) async {
    return _client.put('/v1/flags/$key', data: {
      if (name != null) 'name': name,
      if (description != null) 'description': description,
      if (variants != null) 'variants': variants,
      if (rules != null) 'rules': rules,
    });
  }

  Future<Map<String, dynamic>> delete(String key) async {
    return _client.delete('/v1/flags/$key');
  }

  Future<Map<String, dynamic>> toggle(String key, bool enabled) async {
    return _client.patch('/v1/flags/$key/toggle', data: {
      'enabled': enabled,
    });
  }

  // --- Evaluation ---

  Future<Map<String, dynamic>> getFlag(String key) async {
    return _client.get('/v1/flags/evaluate/$key');
  }

  Future<Map<String, dynamic>> getAllFlags({
    Map<String, dynamic>? context,
  }) async {
    return _client.post('/v1/flags/evaluate/all', data: {
      if (context != null) 'context': context,
    });
  }

  Future<Map<String, dynamic>> evaluateFlag({
    required String key,
    Map<String, dynamic>? context,
  }) async {
    return _client.post('/v1/flags/evaluate', data: {
      'key': key,
      if (context != null) 'context': context,
    });
  }
}
