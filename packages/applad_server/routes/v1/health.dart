import 'package:applad_core/applad_core.dart';
import 'package:dart_frog/dart_frog.dart';
import 'package:applad_server/src/server_config.dart';

/// GET /v1/health — detailed health check
Response onRequest(RequestContext context) {
  ApplAdConfig? config;
  String configStatus;

  try {
    config = ServerConfig.instance;
    configStatus = 'loaded';
  } catch (_) {
    configStatus = 'not_loaded';
  }

  return Response.json(body: {
    'status': 'ok',
    'version': '0.1.0',
    'config': configStatus,
    if (config != null) ...{
      'org': config.org.id,
      'project': config.project.id,
    },
    'timestamp': DateTime.now().toIso8601String(),
  });
}
