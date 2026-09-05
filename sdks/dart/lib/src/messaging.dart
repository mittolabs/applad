import 'package:dio/dio.dart';

/// Messaging service — send emails.
class Messaging {
  final Dio _dio;

  Messaging(this._dio);

  /// Send an email.
  Future<Map<String, dynamic>> sendEmail({
    required List<String> to,
    required String subject,
    String? html,
  }) async {
    final res = await _dio.post('/v1/messaging/email', data: {
      'to': to,
      'subject': subject,
      if (html != null) 'html': html,
    });
    return res.data;
  }

  // --- Templates ---

  /// Create a reusable message template with {{variable}} placeholders.
  Future<Map<String, dynamic>> createTemplate({
    required String name,
    required String type,
    required String subject,
    required String body,
    List<String>? variables,
    String templateId = 'unique()',
  }) async {
    final res = await _dio.post('/v1/messaging/templates', data: {
      'templateId': templateId,
      'name': name,
      'type': type,
      'subject': subject,
      'body': body,
      'variables': variables ?? [],
    });
    return Map<String, dynamic>.from(res.data as Map);
  }

  Future<Map<String, dynamic>> listTemplates() async {
    final res = await _dio.get('/v1/messaging/templates');
    return Map<String, dynamic>.from(res.data as Map);
  }

  Future<Map<String, dynamic>> getTemplate(String templateId) async {
    final res = await _dio.get('/v1/messaging/templates/$templateId');
    return Map<String, dynamic>.from(res.data as Map);
  }

  Future<Map<String, dynamic>> deleteTemplate(String templateId) async {
    final res = await _dio.delete('/v1/messaging/templates/$templateId');
    return res.data ?? {};
  }

  /// Render the template with [variables] and send to [to].
  Future<Map<String, dynamic>> sendTemplate({
    required String templateId,
    required List<String> to,
    Map<String, String>? variables,
  }) async {
    final res = await _dio.post('/v1/messaging/templates/$templateId/send',
        data: {
          'to': to,
          'variables': variables ?? {},
        });
    return Map<String, dynamic>.from(res.data as Map);
  }

  // ── Topics ──────────────────────────────────────────────────────────────────
  //
  // A topic is a group of targets a message goes to at once — "everyone who
  // wants notifications" rather than a list the app has to keep and iterate.

  /// Creates a topic, or returns the existing one with this id.
  Future<Map<String, dynamic>> createTopic({
    required String name,
    String? topicId,
  }) async {
    final res = await _dio.post('/v1/messaging/topics', data: {
      'name': name,
      if (topicId != null) 'topicId': topicId,
    });
    return res.data;
  }

  Future<Map<String, dynamic>> listTopics() async {
    final res = await _dio.get('/v1/messaging/topics');
    return res.data;
  }

  Future<Map<String, dynamic>> getTopic(String topicId) async {
    final res = await _dio.get('/v1/messaging/topics/$topicId');
    return res.data;
  }

  /// Adds [target] to a topic.
  Future<Map<String, dynamic>> subscribe({
    required String topicId,
    required String target,
  }) async {
    final res = await _dio.post(
      '/v1/messaging/topics/$topicId/subscribers',
      data: {'target': target},
    );
    return res.data;
  }

  /// Removes [target] from a topic. Unsubscribing something that was never
  /// subscribed is not an error.
  Future<Map<String, dynamic>> unsubscribe({
    required String topicId,
    required String target,
  }) async {
    final res = await _dio.delete(
      '/v1/messaging/topics/$topicId/subscribers/${Uri.encodeComponent(target)}',
    );
    return res.data;
  }

  /// Sends a message to everyone subscribed to a topic.
  Future<Map<String, dynamic>> sendToTopic({
    required String topicId,
    required String subject,
    required String body,
  }) async {
    final res = await _dio.post(
      '/v1/messaging/topics/$topicId/messages',
      data: {'subject': subject, 'body': body},
    );
    return res.data;
  }
}
