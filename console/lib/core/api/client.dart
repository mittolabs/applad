import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

final apiClientProvider = Provider<ApiClient>((ref) {
  // When accessed via the proxy (port 80), relative URL works.
  // When accessed directly (port 3000), we need the full API URL.
  const apiBase = String.fromEnvironment('API_URL', defaultValue: '/v1');
  return ApiClient(baseUrl: apiBase);
});

class ApiClient {
  final Dio _dio;

  ApiClient({required String baseUrl})
      : _dio = Dio(BaseOptions(
          baseUrl: baseUrl,
          headers: {'Content-Type': 'application/json'},
        ));

  Dio get dio => _dio;

  Future<Response<T>> get<T>(String path,
          {Map<String, dynamic>? params}) =>
      _dio.get(path, queryParameters: params);

  Future<Response<T>> post<T>(String path, {Object? data}) =>
      _dio.post(path, data: data);

  Future<Response<T>> put<T>(String path, {Object? data}) =>
      _dio.put(path, data: data);

  Future<Response<T>> patch<T>(String path, {Object? data}) =>
      _dio.patch(path, data: data);

  Future<Response<T>> delete<T>(String path) => _dio.delete(path);

  void setProject(String projectId) {
    _dio.options.headers['X-Applad-Project'] = projectId;
  }

  void setAuthToken(String token) {
    _dio.options.headers['Authorization'] = 'Bearer $token';
  }

  void setApiKey(String key) {
    _dio.options.headers['X-Applad-Key'] = key;
  }

  void setConsoleUser({required String id, required String email, required String name}) {
    _dio.options.headers['X-Console-User-ID'] = id;
    _dio.options.headers['X-Console-User-Email'] = email;
    _dio.options.headers['X-Console-User-Name'] = name;
  }
}
