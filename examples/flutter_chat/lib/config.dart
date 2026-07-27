/// Where this app points and the shape of its data.
///
/// Endpoint and project come from --dart-define so the same build runs against a
/// local stack or the cloud:
///
///   fvm flutter run \
///     --dart-define=APPLAD_ENDPOINT=https://api.applad.io \
///     --dart-define=APPLAD_PROJECT=your_project_id
///
/// The database and table ids are fixed by tool/bootstrap.dart, which creates
/// them once.
class Config {
  static const endpoint = String.fromEnvironment(
    'APPLAD_ENDPOINT',
    defaultValue: 'http://localhost:8080',
  );

  static const projectId = String.fromEnvironment('APPLAD_PROJECT');

  /// The chat schema, created by the bootstrap tool.
  static const databaseId = 'chat';
  static const messagesTable = 'messages';

  static bool get isConfigured => projectId.isNotEmpty;
}
