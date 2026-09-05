import 'package:dio/dio.dart';

class Analytics {
  final Dio _dio;

  Analytics(this._dio);

  Future<Map<String, dynamic>> trackEvent({
    required String event,
    Map<String, dynamic>? properties,
  }) async {
    final res = await _dio.post('/v1/analytics/events', data: {
      'event': event,
      if (properties != null) 'properties': properties,
    });
    return res.data;
  }

  Future<Map<String, dynamic>> listEvents({
    String? event,
    String? startDate,
    String? endDate,
    int? limit,
  }) async {
    final res = await _dio.get('/v1/analytics/events', queryParameters: {
      if (event != null) 'event': event,
      if (startDate != null) 'startDate': startDate,
      if (endDate != null) 'endDate': endDate,
      if (limit != null) 'limit': limit,
    });
    return res.data;
  }

  Future<Map<String, dynamic>> getStats({
    String? event,
    String? startDate,
    String? endDate,
    String? interval,
  }) async {
    final res = await _dio.get('/v1/analytics/stats', queryParameters: {
      if (event != null) 'event': event,
      if (startDate != null) 'startDate': startDate,
      if (endDate != null) 'endDate': endDate,
      if (interval != null) 'interval': interval,
    });
    return res.data;
  }

  Future<Map<String, dynamic>> getRealtimeCount() async {
    final res = await _dio.get('/v1/analytics/realtime');
    return res.data;
  }

  /// Summary for the Analytics landing page: events, active users, request
  /// latency and average uptime, over the last 24 hours.
  Future<Map<String, dynamic>> getOverview() async {
    final res = await _dio.get('/v1/analytics/overview');
    return res.data;
  }

  /// Per-route request latency measured by the platform over the last 24 hours.
  Future<Map<String, dynamic>> getPerformance() async {
    final res = await _dio.get('/v1/analytics/performance');
    return res.data;
  }

  // ── Uptime monitors ───────────────────────────────────────────────────────

  Future<Map<String, dynamic>> listMonitors() async {
    final res = await _dio.get('/v1/analytics/uptime');
    return res.data;
  }

  Future<Map<String, dynamic>> createMonitor({
    required String name,
    required String url,
    String checkType = 'http',
    int intervalSecs = 60,
    String? keyword,
  }) async {
    final res = await _dio.post('/v1/analytics/uptime', data: {
      'name': name,
      'url': url,
      'checkType': checkType,
      'intervalSecs': intervalSecs,
      if (keyword != null) 'keyword': keyword,
    });
    return res.data;
  }

  Future<void> deleteMonitor(String monitorId) async {
    await _dio.delete('/v1/analytics/uptime/$monitorId');
  }

  // ── Cron monitors ─────────────────────────────────────────────────────────

  Future<Map<String, dynamic>> listCronMonitors() async {
    final res = await _dio.get('/v1/analytics/crons');
    return res.data;
  }

  Future<Map<String, dynamic>> createCronMonitor({
    required String name,
    required String schedule,
    String timezone = 'UTC',
    int gracePeriod = 5,
  }) async {
    final res = await _dio.post('/v1/analytics/crons', data: {
      'name': name,
      'schedule': schedule,
      'timezone': timezone,
      'gracePeriod': gracePeriod,
    });
    return res.data;
  }

  Future<void> deleteCronMonitor(String monitorId) async {
    await _dio.delete('/v1/analytics/crons/$monitorId');
  }

  /// Report a run of a scheduled job. Call it when the job finishes; a monitor
  /// that hears nothing within its grace period is marked missed.
  Future<Map<String, dynamic>> cronCheckin(
    String monitorId, {
    String status = 'ok',
    int? durationMs,
    String? errorMsg,
  }) async {
    final res = await _dio.post('/v1/analytics/crons/$monitorId/checkin', data: {
      'status': status,
      if (durationMs != null) 'durationMs': durationMs,
      if (errorMsg != null) 'errorMsg': errorMsg,
    });
    return res.data;
  }
}
