import 'dart:io';
import 'package:applad_core/applad_core.dart';
import 'package:dart_frog/dart_frog.dart';

/// GET /v1/config
///
/// Returns the entire parsed configuration tree (all merged YAML files)
/// discovered from the workspace root.
Response onRequest(RequestContext context) {
  final loader = context.read<ConfigLoader>();
  final workspacePath =
      Platform.environment['APPLAD_WORKSPACE_ROOT'] ?? Directory.current.path;

  try {
    final configTree = loader.loadDirectoryRecursive(workspacePath);
    return Response.json(body: {
      'status': 'ok',
      'workspace': configTree,
    });
  } catch (e, st) {
    return Response.json(
      statusCode: 500,
      body: {
        'error': e.toString(),
        'trace': st.toString(),
      },
    );
  }
}
