import 'dart:io';
import 'package:applad_core/applad_core.dart';
import 'package:dart_frog/dart_frog.dart';

/// GET /v1/config
///
/// Returns the entire parsed configuration tree (all merged YAML files)
/// discovered from the workspace root.
Response onRequest(RequestContext context) {
  try {
    final config = context.read<ApplAdConfig>();
    return Response.json(body: {
      'status': 'ok',
      'config': config.toJson(),
    });
  } catch (e, st) {
    return Response.json(
      statusCode: 500,
      body: {
        'status': 'error',
        'error': e.toString(),
        if (Platform.environment['APPLAD_ENV'] == 'development')
          'trace': st.toString(),
      },
    );
  }
}
