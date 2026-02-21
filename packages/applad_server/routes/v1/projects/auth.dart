import 'package:applad_core/applad_core.dart';
import 'package:dart_frog/dart_frog.dart';

/// GET /v1/projects/auth
Response onRequest(RequestContext context) {
  final config = context.read<ApplAdConfig>();
  final auth = config.auth;

  if (auth == null) {
    return Response.json(
      statusCode: 404,
      body: {
        'status': 'error',
        'message': 'Auth configuration not found',
      },
    );
  }

  return Response.json(body: {
    'status': 'ok',
    'auth': auth.toJson(),
  });
}
