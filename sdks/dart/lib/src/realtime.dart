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
  final Future<void> Function() _onCancel;

  RealtimeSubscription(this.channel, this._onCancel);

  /// Cancel this subscription. Sends an unsubscribe frame once no callbacks
  /// remain on the channel.
  Future<void> cancel() => _onCancel();
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
///
/// Robust against ordering and disconnects:
///  - subscriptions issued before the socket opens are buffered and flushed
///    once it is ready, so `connect()` then `subscribe()` works without a race;
///  - dropped connections are re-established with capped exponential backoff
///    and every active channel is re-subscribed on reopen;
///  - [disconnect] / [dispose] stop reconnection.
///
/// Data (row) channels require an authenticated connection. When a session has
/// been set on the client (via `setJWT`/`setSession`), the token is forwarded
/// as `?token=` on the WebSocket URL.
class Realtime {
  final String _endpoint;
  final String _projectId;
  String? _token;
  WebSocketChannel? _channel;
  final _controller = StreamController<RealtimeEvent>.broadcast();
  bool _connected = false;
  bool _closedByUser = false;
  int _reconnectAttempts = 0;
  Timer? _reconnectTimer;

  /// Active channels with a reference count of live subscriptions, so the set
  /// can be re-subscribed after a reconnect and unsubscribed when the last
  /// callback is cancelled.
  final Map<String, int> _channels = {};

  Realtime({required String endpoint, required String projectId})
      : _endpoint = endpoint.replaceFirst(RegExp(r'^http'), 'ws'),
        _projectId = projectId;

  /// Whether the client is currently connected.
  bool get isConnected => _connected;

  /// Stream of all incoming realtime events.
  Stream<RealtimeEvent> get stream => _controller.stream;

  /// Set (or clear) the session token forwarded on the realtime connection.
  /// Called automatically by the parent client's `setJWT`/`setSession`. If a
  /// connection is already open it is re-established so the token takes effect.
  void setToken(String? token) {
    _token = token;
    final channel = _channel;
    if (channel != null) {
      // Reconnect to apply the new token; the old socket's handlers are scoped
      // and will no-op once `_channel` no longer points at it.
      _channel = null;
      _connected = false;
      channel.sink.close();
      connect();
    }
  }

  /// Connect to the realtime server.
  void connect() {
    if (_channel != null) return;
    _closedByUser = false;

    final params = <String, String>{'project': _projectId};
    if (_token != null) params['token'] = _token!;
    final uri =
        Uri.parse('$_endpoint/v1/realtime').replace(queryParameters: params);

    final channel = WebSocketChannel.connect(uri);
    _channel = channel;

    channel.ready.then((_) {
      if (_channel != channel) return; // superseded
      _connected = true;
      _reconnectAttempts = 0;
      // (Re-)subscribe every active channel: covers subscriptions buffered
      // before open and re-subscription after a reconnect.
      if (_channels.isNotEmpty) {
        _rawSend(channel, {
          'type': 'subscribe',
          'channels': _channels.keys.toList(),
        });
      }
    }).catchError((_) {
      if (_channel != channel) return;
      _connected = false;
      _channel = null;
      if (!_closedByUser) _scheduleReconnect();
    });

    channel.stream.listen(
      (data) {
        try {
          final json = jsonDecode(data as String) as Map<String, dynamic>;
          _controller.add(RealtimeEvent.fromJson(json));
        } catch (_) {}
      },
      onDone: () {
        if (_channel != channel) return;
        _connected = false;
        _channel = null;
        if (!_closedByUser) _scheduleReconnect();
      },
      onError: (_) {
        if (_channel != channel) return;
        _connected = false;
        _channel = null;
        if (!_closedByUser) _scheduleReconnect();
      },
    );
  }

  void _scheduleReconnect() {
    if (_reconnectTimer != null) return;
    final attempt = _reconnectAttempts > 20 ? 20 : _reconnectAttempts;
    final delayMs = (1000 * (1 << attempt)).clamp(1000, 30000).toInt();
    _reconnectAttempts++;
    _reconnectTimer = Timer(Duration(milliseconds: delayMs), () {
      _reconnectTimer = null;
      if (!_closedByUser) connect();
    });
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
    _channels[channel] = (_channels[channel] ?? 0) + 1;
    // Send now if connected; otherwise it is flushed once ready.
    if (_connected && _channel != null) {
      _rawSend(_channel!, {'type': 'subscribe', 'channels': [channel]});
    }

    final sub =
        _controller.stream.where((e) => e.channel == channel).listen(callback);

    return RealtimeSubscription(channel, () async {
      await sub.cancel();
      final remaining = (_channels[channel] ?? 1) - 1;
      if (remaining <= 0) {
        _channels.remove(channel);
        if (_connected && _channel != null) {
          _rawSend(_channel!, {'type': 'unsubscribe', 'channels': [channel]});
        }
      } else {
        _channels[channel] = remaining;
      }
    });
  }

  /// Unsubscribe from a channel entirely (drops all local refcounts).
  void unsubscribe(String channel) {
    _channels.remove(channel);
    if (_connected && _channel != null) {
      _rawSend(_channel!, {'type': 'unsubscribe', 'channels': [channel]});
    }
  }

  /// Disconnect from the realtime server and stop reconnection.
  void disconnect() {
    _closedByUser = true;
    _reconnectTimer?.cancel();
    _reconnectTimer = null;
    _reconnectAttempts = 0;
    _channel?.sink.close();
    _channel = null;
    _connected = false;
  }

  /// Disconnect and release the event stream. Use when the client is discarded.
  void dispose() {
    disconnect();
    _controller.close();
  }

  void _rawSend(WebSocketChannel channel, Map<String, dynamic> message) {
    channel.sink.add(jsonEncode(message));
  }
}
