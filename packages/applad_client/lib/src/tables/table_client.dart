library;

import '../applad_client.dart';

/// Client for a specific table — supports CRUD operations.
final class TableClient {
  TableClient({required this.client, required this.table});

  final ApplAdClient client;
  final String table;

  /// Insert a row into the table.
  Future<Map<String, dynamic>> insert(Map<String, dynamic> data) async {
    throw UnimplementedError('insert — available in Phase 2');
  }

  /// Update rows matching the given ID.
  Future<Map<String, dynamic>> update(String id, Map<String, dynamic> data) async {
    throw UnimplementedError('update — available in Phase 2');
  }

  /// Delete a row by ID.
  Future<void> delete(String id) async {
    throw UnimplementedError('delete — available in Phase 2');
  }

  /// Subscribe to real-time changes on this table.
  void on(RealtimeEvent event, void Function(Map<String, dynamic> payload) callback) {
    throw UnimplementedError('Realtime — available in Phase 3');
  }
}

/// Real-time event types.
enum RealtimeEvent {
  insert,
  update,
  delete,
  all,
}
