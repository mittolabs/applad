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
}
