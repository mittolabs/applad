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

  /// The row data from the event payload (for database change events).
  Map<String, dynamic>? get row {
    if (payload is Map) {
      final p = Map<String, dynamic>.from(payload as Map);
      final data = p['new'] ?? p['old'];
      if (data is Map) return Map<String, dynamic>.from(data);
    }
    return null;
  }

  /// The previous row data (only populated for update events).
  Map<String, dynamic>? get oldRow {
    if (payload is Map) {
      final p = Map<String, dynamic>.from(payload as Map);
      final data = p['old'];
      if (data is Map) return Map<String, dynamic>.from(data);
    }
    return null;
  }
}

/// Realtime subscription handle — call [cancel] to unsubscribe.
class RealtimeSubscription {
  final String channel;
  final StreamSubscription _sub;

  RealtimeSubscription(this.channel, this._sub);

  /// Cancel this subscription.
  void cancel() => _sub.cancel();
}

/// A fluent builder for subscribing to database table change events.
///
/// Created via [Realtime.database] — do not instantiate directly.
///
/// ```dart
/// final sub = client.realtime
///     .database('myDb', 'posts')
///     .onInsert((row) => print('new post: $row'))
///     .onUpdate((row) => print('updated: $row'))
///     .onDelete((row) => print('deleted: $row'))
///     .subscribe();
///
/// // Later:
/// sub.cancel();
/// ```
class DatabaseChannel {
  final Realtime _realtime;
  final String _databaseId;
  final String _tableName;
  void Function(Map<String, dynamic> row)? _onInsert;
  void Function(Map<String, dynamic> row)? _onUpdate;
  void Function(Map<String, dynamic> row)? _onDelete;

  DatabaseChannel._(this._realtime, this._databaseId, this._tableName);

  /// Called when a row is inserted into the table.
  DatabaseChannel onInsert(void Function(Map<String, dynamic> row) callback) {
    _onInsert = callback;
    return this;
  }

  /// Called when a row is updated.
  DatabaseChannel onUpdate(void Function(Map<String, dynamic> row) callback) {
    _onUpdate = callback;
    return this;
  }

  /// Called when a row is deleted.
  DatabaseChannel onDelete(void Function(Map<String, dynamic> row) callback) {
    _onDelete = callback;
    return this;
  }

  /// Subscribe and start receiving events. Returns a handle to cancel.
  RealtimeSubscription subscribe() {
    final channel =
        'databases.${_realtime._projectId}.$_databaseId.$_tableName';
    return _realtime.subscribe(channel, (event) {
      final row = event.row ?? {};
      switch (event.type) {
        case 'databases.rows.create':
          _onInsert?.call(row);
        case 'databases.rows.update':
          _onUpdate?.call(row);
        case 'databases.rows.delete':
          _onDelete?.call(event.oldRow ?? row);
      }
    });
  }
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

  /// Returns a [DatabaseChannel] builder for subscribing to table-level
  /// INSERT / UPDATE / DELETE events.
  ///
  /// ```dart
  /// final sub = client.realtime
  ///     .database('myDb', 'posts')
  ///     .onInsert((row) => setState(() => posts.add(row)))
  ///     .subscribe();
  /// ```
  DatabaseChannel database(String databaseId, String tableName) {
    return DatabaseChannel._(this, databaseId, tableName);
  }

  /// Subscribe to a raw channel and receive events via callback.
  RealtimeSubscription subscribe(
    String channel,
    void Function(RealtimeEvent event) callback,
  ) {
    _send({'type': 'subscribe', 'channels': [channel]});
    final sub =
        _controller.stream.where((e) => e.channel == channel).listen(callback);
    return RealtimeSubscription(channel, sub);
  }

  /// Unsubscribe from a channel.
  void unsubscribe(String channel) {
    _send({'type': 'unsubscribe', 'channels': [channel]});
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
