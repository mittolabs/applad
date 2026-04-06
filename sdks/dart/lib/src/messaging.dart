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
}
