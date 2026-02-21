import 'package:dart_frog/dart_frog.dart';

/// GET / — root health check
Response onRequest(RequestContext context) {
  return Response.json(body: {
    'service': 'applad',
    'status': 'ok',
    'version': '0.1.0',
    'docs': 'https://applad.dev',
  });
}
