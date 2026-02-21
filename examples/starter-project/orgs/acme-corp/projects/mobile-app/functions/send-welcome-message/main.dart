/// Send Welcome Message — Applad serverless function.
///
/// Triggered when a new user signs up. Sends a welcome email
/// using the configured messaging provider.
library;

import 'dart:convert';

/// Function entry point.
/// [event] contains the trigger payload (e.g., the new user's data).
Future<Map<String, dynamic>> handler(Map<String, dynamic> event) async {
  final user = event['user'] as Map<String, dynamic>?;
  if (user == null) {
    return {'error': 'No user data in event'};
  }

  final email = user['email'] as String?;
  final name = user['display_name'] as String? ?? 'there';

  if (email == null) {
    return {'error': 'User has no email'};
  }

  // In Phase 4, this will use the Applad messaging SDK.
  // For now, this demonstrates the function structure.
  print('Sending welcome email to $email...');

  // Simulate sending email
  await Future.delayed(const Duration(milliseconds: 100));

  print(jsonEncode({
    'to': email,
    'subject': 'Welcome to Acme Corp, $name!',
    'template': 'welcome',
    'variables': {
      'name': name,
      'email': email,
    },
  }));

  return {
    'success': true,
    'message': 'Welcome email queued for $email',
  };
}
