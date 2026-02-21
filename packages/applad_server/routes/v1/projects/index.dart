import 'package:applad_core/applad_core.dart';
import 'package:dart_frog/dart_frog.dart';

/// GET /v1/projects
Response onRequest(RequestContext context) {
  final config = context.read<ApplAdConfig>();
  return Response.json(body: {
    'status': 'ok',
    'projects': [config.project.toJson()],
  });
}
