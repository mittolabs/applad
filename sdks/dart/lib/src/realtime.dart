import 'dart:async';
import 'dart:convert';
import 'package:web_socket_channel/web_socket_channel.dart';

/// Realtime event received from the server.
class RealtimeEvent {
  final String type;
  final String channel;
  final String timestamp;
  final dynamic payload;

  RealtimeEvent({
    required this.type,
    required this.channel,
    required this.timestamp,
    this.payload,
  });

  factory RealtimeEvent.fromJson(Map<String, dynamic> json) {
    return RealtimeEvent(
      type: json['type'] ?? '',
      channel: json['channel'] ?? '',
      timestamp: json['timestamp'] ?? '',
      payload: json['payload'],
    );
  }
}

/// Realtime subscription handle.
class RealtimeSubscription {
  final String channel;
  final StreamSubscription _sub;

  RealtimeSubscription(this.channel, this._sub);

  /// Cancel this subscription.
  void cancel() => _sub.cancel();
}

/// Realtime WebSocket client for subscribing to live events.
class Realtime {
  final String _endpoint;
  final String _projectId;
  WebSocketChannel? _channel;
  final _controller = StreamController<RealtimeEvent>.broadcast();
  bool _connected = false;

  Realtime({required String endpoint, required String projectId})
      : _endpoint = endpoint.replaceFirst(RegExp(r'^http'), 'ws'),
        _projectId = projectId;

  /// Whether the client is currently connected.
  bool get isConnected => _connected;

  /// Stream of all incoming realtime events.
  Stream<RealtimeEvent> get stream => _controller.stream;

  /// Connect to the realtime server.
  void connect() {
    if (_connected) return;

    final uri = Uri.parse('$_endpoint/v1/realtime')
        .replace(queryParameters: {'project': _projectId});

    _channel = WebSocketChannel.connect(uri);
    _connected = true;

    _channel!.stream.listen(
      (data) {
        try {
          final json = jsonDecode(data as String) as Map<String, dynamic>;
          _controller.add(RealtimeEvent.fromJson(json));
        } catch (_) {}
      },
      onDone: () {
        _connected = false;
      },
      onError: (error) {
        _connected = false;
      },
    );
  }

  /// Subscribe to a channel and receive events via callback.
  RealtimeSubscription subscribe(
    String channel,
    void Function(RealtimeEvent event) callback,
  ) {
    // Send subscribe message
    _send({
      'type': 'subscribe',
      'channels': [channel]
    });

    // Filter stream for this channel
    final sub =
        _controller.stream.where((e) => e.channel == channel).listen(callback);

    return RealtimeSubscription(channel, sub);
  }

  /// Unsubscribe from a channel.
  void unsubscribe(String channel) {
    _send({
      'type': 'unsubscribe',
      'channels': [channel]
    });
  }

  /// Disconnect from the realtime server.
  void disconnect() {
    _channel?.sink.close();
    _connected = false;
  }

  void _send(Map<String, dynamic> message) {
    if (_channel != null && _connected) {
      _channel!.sink.add(jsonEncode(message));
    }
  }
}
