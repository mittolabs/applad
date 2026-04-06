import 'package:dio/dio.dart';

import 'auth.dart';
import 'databases.dart';
import 'deploy.dart';
import 'functions.dart';
import 'messaging.dart';
import 'realtime.dart';
import 'storage.dart';
import 'workflows.dart';

class Applad {
  final String endpoint;
  final String projectId;
  final Dio _dio;

  late final Auth auth;
  late final Users users;
  late final Databases databases;
  late final Storage storage;
  late final Deploy deploy;
  late final Messaging messaging;
  late final Realtime realtime;
  late final Functions functions;
  late final Workflows workflows;

  Applad({required this.endpoint, required this.projectId})
      : _dio = Dio(BaseOptions(
          baseUrl: endpoint,
          headers: {
            'x-applad-project': projectId,
            'Content-Type': 'application/json',
          },
        )) {
    auth = Auth(_dio);
    users = Users(_dio);
    databases = Databases(_dio);
    storage = Storage(_dio);
    deploy = Deploy(_dio);
    messaging = Messaging(_dio);
    realtime = Realtime(endpoint: endpoint, projectId: projectId);
    functions = Functions(_dio);
    workflows = Workflows(_dio);
  }

  Dio get dio => _dio;

  void setSession(String sessionId) {
    _dio.options.headers['x-applad-session'] = sessionId;
  }

  void setJWT(String jwt) {
    _dio.options.headers['Authorization'] = 'Bearer $jwt';
  }

  void setKey(String key) {
    _dio.options.headers['x-applad-key'] = key;
  }
}
