import '../server_client.dart';

class Analytics {
  final ApplAdServer _client;

  Analytics(this._client);

  Future<Map<String, dynamic>> trackEvent({
    required String event,
    Map<String, dynamic>? properties,
  }) async {
    return _client.post('/v1/analytics/events', data: {
      'event': event,
      if (properties != null) 'properties': properties,
    });
  }

  Future<Map<String, dynamic>> listEvents({
    String? event,
    String? startDate,
    String? endDate,
    int? limit,
  }) async {
    final query = <String, String>{};
    if (event != null) query['event'] = event;
    if (startDate != null) query['startDate'] = startDate;
    if (endDate != null) query['endDate'] = endDate;
    if (limit != null) query['limit'] = limit.toString();
    final qs = query.isNotEmpty
        ? '?${query.entries.map((e) => '${e.key}=${Uri.encodeComponent(e.value)}').join('&')}'
        : '';
    return _client.get('/v1/analytics/events$qs');
  }

  Future<Map<String, dynamic>> getStats({
    String? event,
    String? startDate,
    String? endDate,
    String? interval,
  }) async {
    final query = <String, String>{};
    if (event != null) query['event'] = event;
    if (startDate != null) query['startDate'] = startDate;
    if (endDate != null) query['endDate'] = endDate;
    if (interval != null) query['interval'] = interval;
    final qs = query.isNotEmpty
        ? '?${query.entries.map((e) => '${e.key}=${Uri.encodeComponent(e.value)}').join('&')}'
        : '';
    return _client.get('/v1/analytics/stats$qs');
  }

  Future<Map<String, dynamic>> getRealtimeCount() async {
    return _client.get('/v1/analytics/realtime');
  }
}
