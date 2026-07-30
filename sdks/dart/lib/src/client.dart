import 'package:dio/dio.dart';

import 'analytics.dart';
import 'auth.dart';
import 'billing.dart';
import 'databases.dart';
import 'deploy.dart';
import 'edge.dart';
import 'flags.dart';
import 'functions.dart';
import 'messaging.dart';
import 'realtime.dart';
import 'regions.dart';
import 'search.dart';
import 'storage.dart';
import 'teams.dart';
import 'vectors.dart';
import 'workflows.dart';
import 'observe.dart';

class Applad {
  final String endpoint;
  final String projectId;
  final Dio _dio;

  late final Analytics analytics;
  late final Auth auth;
  late final Billing billing;
  late final Users users;
  late final Databases databases;
  late final Storage storage;
  late final Teams teams;
  late final Deploy deploy;
  late final Edge edge;
  late final Flags flags;
  late final Messaging messaging;
  late final Realtime realtime;
  late final Functions functions;
  late final Regions regions;
  late final Search search;
  late final Vectors vectors;
  late final Workflows workflows;
  late final Observe observe;

  Applad({required this.endpoint, required this.projectId})
      : _dio = Dio(BaseOptions(
          baseUrl: endpoint,
          headers: {
            'x-applad-project': projectId,
            'Content-Type': 'application/json',
          },
        )) {
    analytics = Analytics(_dio);
    auth = Auth(_dio);
    billing = Billing(_dio);
    users = Users(_dio);
    databases = Databases(_dio);
    storage = Storage(_dio);
    teams = Teams(_dio);
    deploy = Deploy(_dio);
    edge = Edge(_dio);
    flags = Flags(_dio);
    messaging = Messaging(_dio);
    realtime = Realtime(endpoint: endpoint, projectId: projectId);
    functions = Functions(_dio);
    regions = Regions(_dio);
    search = Search(_dio);
    vectors = Vectors(_dio);
    workflows = Workflows(_dio);
    observe = Observe(_dio);
  }

  Dio get dio => _dio;

  /// Authenticate as a signed-in user with their session secret / JWT.
  ///
  /// The backend authenticates users via `Authorization: Bearer <secret>`, so
  /// this is an alias for [setJWT]. The token is also forwarded to the realtime
  /// client so subscriptions to data channels are authorized.
  void setSession(String sessionId) {
    setJWT(sessionId);
  }

  void setJWT(String jwt) {
    _dio.options.headers['Authorization'] = 'Bearer $jwt';
    realtime.setToken(jwt);
  }

  void setKey(String key) {
    _dio.options.headers['x-applad-key'] = key;
  }
}
