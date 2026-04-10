import '../server_client.dart';

/// Messaging service — send emails, SMS, and push notifications (server-side).
class Messaging {
  final ApplAdServer _client;

  Messaging(this._client);

  Future<Map<String, dynamic>> sendEmail({
    required List<String> to,
    required String subject,
    String? html,
  }) async {
    return _client.post('/v1/messaging/email', data: {
      'to': to,
      'subject': subject,
      if (html != null) 'html': html,
    });
  }

  Future<Map<String, dynamic>> sendSMS({
    required List<String> to,
    required String body,
  }) async {
    return _client.post('/v1/messaging/sms', data: {
      'to': to,
      'body': body,
    });
  }

  Future<Map<String, dynamic>> sendPush({
    required List<String> to,
    required String title,
    required String body,
    Map<String, dynamic>? data,
  }) async {
    return _client.post('/v1/messaging/push', data: {
      'to': to,
      'title': title,
      'body': body,
      if (data != null) 'data': data,
    });
  }

  // --- Templates ---

  Future<Map<String, dynamic>> createTemplate({
    required String name,
    required String type,
    required String subject,
    required String body,
    List<String>? variables,
    String templateId = 'unique()',
  }) {
    return _client.post('/v1/messaging/templates', data: {
      'templateId': templateId,
      'name': name,
      'type': type,
      'subject': subject,
      'body': body,
      'variables': variables ?? [],
    });
  }

  Future<Map<String, dynamic>> listTemplates() {
    return _client.get('/v1/messaging/templates');
  }

  Future<Map<String, dynamic>> getTemplate(String templateId) {
    return _client.get('/v1/messaging/templates/$templateId');
  }

  Future<Map<String, dynamic>> deleteTemplate(String templateId) {
    return _client.delete('/v1/messaging/templates/$templateId');
  }

  Future<Map<String, dynamic>> sendTemplate({
    required String templateId,
    required List<String> to,
    Map<String, String>? variables,
  }) {
    return _client.post('/v1/messaging/templates/$templateId/send', data: {
      'to': to,
      'variables': variables ?? {},
    });
  }
}
