import 'package:dio/dio.dart';

class Applad {
  final String endpoint;
  final String projectId;
  final Dio _dio;

  Applad({required this.endpoint, required this.projectId})
      : _dio = Dio(BaseOptions(
          baseUrl: endpoint,
          headers: {
            'x-applad-project': projectId,
            'Content-Type': 'application/json',
          },
        ));

  Dio get dio => _dio;

  void setSession(String sessionId) {
    _dio.options.headers['x-applad-session'] = sessionId;
  }

  void setJWT(String jwt) {
    _dio.options.headers['Authorization'] = 'Bearer $jwt';
  }
}
