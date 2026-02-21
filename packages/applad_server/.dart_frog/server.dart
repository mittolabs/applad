// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, implicit_dynamic_list_literal

import 'dart:io';

import 'package:dart_frog/dart_frog.dart';

import '../main.dart' as entrypoint;
import '../routes/index.dart' as index;
import '../routes/v1/health.dart' as v1_health;
import '../routes/v1/projects/tables.dart' as v1_projects_tables;
import '../routes/v1/projects/index.dart' as v1_projects_index;
import '../routes/v1/projects/database.dart' as v1_projects_database;
import '../routes/v1/projects/auth.dart' as v1_projects_auth;
import '../routes/v1/orgs/index.dart' as v1_orgs_index;
import '../routes/v1/config/index.dart' as v1_config_index;

import '../routes/v1/_middleware.dart' as v1_middleware;

void main() async {
  final address = InternetAddress.tryParse('') ?? InternetAddress.anyIPv6;
  final port = int.tryParse(Platform.environment['PORT'] ?? '8080') ?? 8080;
  hotReload(() => createServer(address, port));
}

Future<HttpServer> createServer(InternetAddress address, int port) {
  final handler = Cascade().add(buildRootHandler()).handler;
  return entrypoint.run(handler, address, port);
}

Handler buildRootHandler() {
  final pipeline = const Pipeline();
  final router = Router()
    ..mount('/v1/config', (context) => buildV1ConfigHandler()(context))
    ..mount('/v1/orgs', (context) => buildV1OrgsHandler()(context))
    ..mount('/v1/projects', (context) => buildV1ProjectsHandler()(context))
    ..mount('/v1', (context) => buildV1Handler()(context))
    ..mount('/', (context) => buildHandler()(context));
  return pipeline.addHandler(router);
}

Handler buildV1ConfigHandler() {
  final pipeline = const Pipeline().addMiddleware(v1_middleware.middleware);
  final router = Router()
    ..all('/', (context) => v1_config_index.onRequest(context,));
  return pipeline.addHandler(router);
}

Handler buildV1OrgsHandler() {
  final pipeline = const Pipeline().addMiddleware(v1_middleware.middleware);
  final router = Router()
    ..all('/', (context) => v1_orgs_index.onRequest(context,));
  return pipeline.addHandler(router);
}

Handler buildV1ProjectsHandler() {
  final pipeline = const Pipeline().addMiddleware(v1_middleware.middleware);
  final router = Router()
    ..all('/auth', (context) => v1_projects_auth.onRequest(context,))..all('/database', (context) => v1_projects_database.onRequest(context,))..all('/tables', (context) => v1_projects_tables.onRequest(context,))..all('/', (context) => v1_projects_index.onRequest(context,));
  return pipeline.addHandler(router);
}

Handler buildV1Handler() {
  final pipeline = const Pipeline().addMiddleware(v1_middleware.middleware);
  final router = Router()
    ..all('/health', (context) => v1_health.onRequest(context,));
  return pipeline.addHandler(router);
}

Handler buildHandler() {
  final pipeline = const Pipeline();
  final router = Router()
    ..all('/', (context) => index.onRequest(context,));
  return pipeline.addHandler(router);
}

