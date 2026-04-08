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
}
