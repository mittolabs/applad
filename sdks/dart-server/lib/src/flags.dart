import 'client.dart';

/// Flags service — manage and evaluate feature flags.
class Flags {
  final ApplAdServer _client;

  Flags(this._client);

  // --- CRUD ---

  /// List all flags.
  Future<Map<String, dynamic>> list() async {
    return _client.get('/v1/flags');
  }

  /// Create a new flag.
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

  /// Get a flag by key.
  Future<Map<String, dynamic>> get(String key) async {
    return _client.get('/v1/flags/$key');
  }

  /// Update a flag.
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

  /// Delete a flag.
  Future<Map<String, dynamic>> delete(String key) async {
    return _client.delete('/v1/flags/$key');
  }

  /// Toggle a flag on or off.
  Future<Map<String, dynamic>> toggle(String key, bool enabled) async {
    return _client.patch('/v1/flags/$key/toggle', data: {
      'enabled': enabled,
    });
  }

  // --- Evaluation ---

  /// Get a single flag evaluation by key.
  Future<Map<String, dynamic>> getFlag(String key) async {
    return _client.get('/v1/flags/evaluate/$key');
  }

  /// Get all flag evaluations with optional context.
  Future<Map<String, dynamic>> getAllFlags({
    Map<String, dynamic>? context,
  }) async {
    return _client.post('/v1/flags/evaluate/all', data: {
      if (context != null) 'context': context,
    });
  }

  /// Evaluate a flag with key and context.
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
