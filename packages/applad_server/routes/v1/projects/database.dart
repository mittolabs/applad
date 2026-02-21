import 'package:applad_core/applad_core.dart';
import 'package:dart_frog/dart_frog.dart';

/// GET /v1/projects/database
Response onRequest(RequestContext context) {
  final config = context.read<ApplAdConfig>();
  final database = config.database;

  if (database == null) {
    return Response.json(
      statusCode: 404,
      body: {
        'status': 'error',
        'message': 'Database configuration not found',
      },
    );
  }

  return Response.json(body: {
    'status': 'ok',
    'database': database.toJson(),
  });
}
