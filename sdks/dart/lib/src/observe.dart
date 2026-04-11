import 'package:dio/dio.dart';

/// Observe service — error tracking, logs, performance, releases, replays,
/// uptime monitors, cron monitors, and alert rules.
class Observe {
  final Dio _dio;

  Observe(this._dio);

  // ── Overview ──────────────────────────────────────────────────────────────

  Future<Map<String, dynamic>> getOverview() async {
    final res = await _dio.get('/v1/observe/overview');
    return Map<String, dynamic>.from(res.data);
  }

  // ── Errors ────────────────────────────────────────────────────────────────

  Future<Map<String, dynamic>> captureError(
    String title, {
    String? errorType,
    String level = 'error',
    String? stackTrace,
    List<Map<String, dynamic>>? breadcrumbs,
    Map<String, dynamic>? userContext,
    Map<String, dynamic>? requestContext,
    Map<String, dynamic>? runtimeContext,
    Map<String, dynamic>? tags,
    String environment = 'production',
    String? release,
  }) async {
    final res = await _dio.post('/v1/observe/errors', data: {
      'title': title,
      if (errorType != null) 'errorType': errorType,
      'level': level,
      if (stackTrace != null) 'stackTrace': stackTrace,
      if (breadcrumbs != null) 'breadcrumbs': breadcrumbs,
      if (userContext != null) 'userContext': userContext,
      if (requestContext != null) 'requestContext': requestContext,
      if (runtimeContext != null) 'runtimeContext': runtimeContext,
      if (tags != null) 'tags': tags,
      'environment': environment,
      if (release != null) 'release': release,
    });
    return Map<String, dynamic>.from(res.data);
  }

  Future<Map<String, dynamic>> listErrors({
    String? status,
    String? level,
    int? limit,
  }) async {
    final res = await _dio.get('/v1/observe/errors', queryParameters: {
      if (status != null) 'status': status,
      if (level != null) 'level': level,
      if (limit != null) 'limit': limit,
    });
    return Map<String, dynamic>.from(res.data);
  }

  Future<Map<String, dynamic>> getError(String errorId) async {
    final res = await _dio.get('/v1/observe/errors/$errorId');
    return Map<String, dynamic>.from(res.data);
  }

  Future<void> resolveError(String errorId) async {
    await _dio.patch('/v1/observe/errors/$errorId/resolve');
  }

  Future<void> ignoreError(String errorId) async {
    await _dio.patch('/v1/observe/errors/$errorId/ignore');
  }

  Future<void> unresolveError(String errorId) async {
    await _dio.patch('/v1/observe/errors/$errorId/unresolve');
  }

  Future<void> setErrorPriority(String errorId, String priority) async {
    await _dio.patch('/v1/observe/errors/$errorId/priority',
        data: {'priority': priority});
  }

  Future<void> assignError(String errorId, String assignee) async {
    await _dio.patch('/v1/observe/errors/$errorId/assign',
        data: {'assignee': assignee});
  }

  Future<void> addNote(String errorId, String text) async {
    await _dio.post('/v1/observe/errors/$errorId/activity', data: {'text': text});
  }

  // ── Logs ──────────────────────────────────────────────────────────────────

  Future<Map<String, dynamic>> captureLog(
    String message, {
    String level = 'info',
    String? source,
    String? environment,
    String? release,
    Map<String, dynamic>? meta,
    String? traceId,
    String? spanId,
  }) async {
    final res = await _dio.post('/v1/observe/logs', data: {
      'message': message,
      'level': level,
      if (source != null) 'source': source,
      if (environment != null) 'environment': environment,
      if (release != null) 'release': release,
      if (meta != null) 'meta': meta,
      if (traceId != null) 'traceId': traceId,
      if (spanId != null) 'spanId': spanId,
    });
    return Map<String, dynamic>.from(res.data);
  }

  Future<Map<String, dynamic>> listLogs({
    String? level,
    String? source,
    int? limit,
  }) async {
    final res = await _dio.get('/v1/observe/logs', queryParameters: {
      if (level != null) 'level': level,
      if (source != null) 'source': source,
      if (limit != null) 'limit': limit,
    });
    return Map<String, dynamic>.from(res.data);
  }

  // ── Performance ───────────────────────────────────────────────────────────

  Future<Map<String, dynamic>> getPerformance() async {
    final res = await _dio.get('/v1/observe/performance');
    return Map<String, dynamic>.from(res.data);
  }

  Future<void> recordPerf(
    String path, {
    String method = 'GET',
    double p50Ms = 0,
    double p75Ms = 0,
    double p95Ms = 0,
    double p99Ms = 0,
    double rps = 0,
    double errorPct = 0,
    int reqCount = 1,
  }) async {
    await _dio.post('/v1/observe/performance', data: {
      'path': path,
      'method': method,
      'p50Ms': p50Ms,
      'p75Ms': p75Ms,
      'p95Ms': p95Ms,
      'p99Ms': p99Ms,
      'rps': rps,
      'errorPct': errorPct,
      'reqCount': reqCount,
    });
  }

  Future<void> reportWebVitals({
    String? pageUrl,
    double? lcp,
    double? fid,
    double? cls,
    double? ttfb,
    double? fcp,
    double? inp,
  }) async {
    await _dio.post('/v1/observe/performance/vitals', data: {
      if (pageUrl != null) 'pageUrl': pageUrl,
      if (lcp != null) 'lcp': lcp,
      if (fid != null) 'fid': fid,
      if (cls != null) 'cls': cls,
      if (ttfb != null) 'ttfb': ttfb,
      if (fcp != null) 'fcp': fcp,
      if (inp != null) 'inp': inp,
    });
  }

  // ── Releases ──────────────────────────────────────────────────────────────

  Future<Map<String, dynamic>> listReleases() async {
    final res = await _dio.get('/v1/observe/releases');
    return Map<String, dynamic>.from(res.data);
  }

  Future<Map<String, dynamic>> createRelease(
    String version, {
    String environment = 'production',
    List<Map<String, dynamic>>? commits,
  }) async {
    final res = await _dio.post('/v1/observe/releases', data: {
      'version': version,
      'environment': environment,
      if (commits != null) 'commits': commits,
    });
    return Map<String, dynamic>.from(res.data);
  }

  Future<Map<String, dynamic>> getRelease(String releaseId) async {
    final res = await _dio.get('/v1/observe/releases/$releaseId');
    return Map<String, dynamic>.from(res.data);
  }

  // ── Replays ───────────────────────────────────────────────────────────────

  Future<Map<String, dynamic>> listReplays({int? limit}) async {
    final res = await _dio.get('/v1/observe/replays', queryParameters: {
      if (limit != null) 'limit': limit,
    });
    return Map<String, dynamic>.from(res.data);
  }

  Future<Map<String, dynamic>> createReplay({
    String? sessionId,
    String? userId,
    String? user,
    String? url,
    String? browser,
    String? os,
    String? country,
    int durationSecs = 0,
    int errorCount = 0,
    bool hasRageClick = false,
    bool hasDeadClick = false,
    List<Map<String, dynamic>>? events,
    List<Map<String, dynamic>>? network,
    List<Map<String, dynamic>>? console,
  }) async {
    final res = await _dio.post('/v1/observe/replays', data: {
      if (sessionId != null) 'sessionId': sessionId,
      if (userId != null) 'userId': userId,
      if (user != null) 'user': user,
      if (url != null) 'url': url,
      if (browser != null) 'browser': browser,
      if (os != null) 'os': os,
      if (country != null) 'country': country,
      'durationSecs': durationSecs,
      'errorCount': errorCount,
      'hasRageClick': hasRageClick,
      'hasDeadClick': hasDeadClick,
      if (events != null) 'events': events,
      if (network != null) 'network': network,
      if (console != null) 'console': console,
    });
    return Map<String, dynamic>.from(res.data);
  }

  Future<Map<String, dynamic>> getReplay(String replayId) async {
    final res = await _dio.get('/v1/observe/replays/$replayId');
    return Map<String, dynamic>.from(res.data);
  }

  // ── Uptime monitors ───────────────────────────────────────────────────────

  Future<Map<String, dynamic>> listMonitors() async {
    final res = await _dio.get('/v1/observe/uptime');
    return Map<String, dynamic>.from(res.data);
  }

  Future<Map<String, dynamic>> createMonitor(
    String name,
    String url, {
    String checkType = 'http',
    int intervalSecs = 60,
    String? keyword,
  }) async {
    final res = await _dio.post('/v1/observe/uptime', data: {
      'name': name,
      'url': url,
      'checkType': checkType,
      'intervalSecs': intervalSecs,
      if (keyword != null) 'keyword': keyword,
    });
    return Map<String, dynamic>.from(res.data);
  }

  Future<void> deleteMonitor(String monitorId) async {
    await _dio.delete('/v1/observe/uptime/$monitorId');
  }

  // ── Cron monitors ─────────────────────────────────────────────────────────

  Future<Map<String, dynamic>> listCronMonitors() async {
    final res = await _dio.get('/v1/observe/crons');
    return Map<String, dynamic>.from(res.data);
  }

  Future<Map<String, dynamic>> createCronMonitor(
    String name,
    String schedule, {
    String timezone = 'UTC',
    int gracePeriod = 5,
  }) async {
    final res = await _dio.post('/v1/observe/crons', data: {
      'name': name,
      'schedule': schedule,
      'timezone': timezone,
      'gracePeriod': gracePeriod,
    });
    return Map<String, dynamic>.from(res.data);
  }

  Future<void> toggleCronMonitor(String monitorId) async {
    await _dio.patch('/v1/observe/crons/$monitorId/toggle');
  }

  Future<void> deleteCronMonitor(String monitorId) async {
    await _dio.delete('/v1/observe/crons/$monitorId');
  }

  Future<Map<String, dynamic>> cronCheckin(
    String monitorId, {
    String status = 'ok',
    int? durationMs,
    String? errorMsg,
  }) async {
    final res = await _dio.post('/v1/observe/crons/$monitorId/checkin', data: {
      'status': status,
      if (durationMs != null) 'durationMs': durationMs,
      if (errorMsg != null) 'errorMsg': errorMsg,
    });
    return Map<String, dynamic>.from(res.data);
  }

  // ── Alert rules ───────────────────────────────────────────────────────────

  Future<Map<String, dynamic>> listAlerts() async {
    final res = await _dio.get('/v1/observe/alerts');
    return Map<String, dynamic>.from(res.data);
  }

  Future<Map<String, dynamic>> createAlertRule(
    String name, {
    String metric = 'error_rate',
    String operator = 'gt',
    double threshold = 0,
    String window = '5m',
    String severity = 'warning',
    String channel = 'email',
  }) async {
    final res = await _dio.post('/v1/observe/alerts', data: {
      'name': name,
      'metric': metric,
      'operator': operator,
      'threshold': threshold,
      'window': window,
      'severity': severity,
      'channel': channel,
    });
    return Map<String, dynamic>.from(res.data);
  }

  Future<void> toggleAlertRule(String ruleId) async {
    await _dio.patch('/v1/observe/alerts/$ruleId/toggle');
  }

  Future<void> deleteAlertRule(String ruleId) async {
    await _dio.delete('/v1/observe/alerts/$ruleId');
  }
}
