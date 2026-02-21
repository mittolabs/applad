library;

import '../applad_client.dart';

/// Client for Applad real-time subscriptions.
final class RealtimeClient {
  RealtimeClient({required this.client});

  final ApplAdClient client;

  /// Connect to the real-time broker.
  Future<void> connect() async {
    throw UnimplementedError('Realtime — available in Phase 3');
  }

  /// Disconnect from the real-time broker.
  Future<void> disconnect() async {
    throw UnimplementedError('Realtime — available in Phase 3');
  }

  /// Subscribe to a channel.
  RealtimeSubscription channel(String name) {
    throw UnimplementedError('Realtime channels — available in Phase 3');
  }
}

/// A real-time subscription handle.
final class RealtimeSubscription {
  RealtimeSubscription(this.name);

  final String name;

  RealtimeSubscription on(
      String event, void Function(Map<String, dynamic>) callback) {
    return this;
  }

  Future<RealtimeSubscription> subscribe() async {
    throw UnimplementedError('subscribe — available in Phase 3');
  }

  Future<void> unsubscribe() async {
    throw UnimplementedError('unsubscribe — available in Phase 3');
  }
}
