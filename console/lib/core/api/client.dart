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
          // Ensure trailing slash so Dio appends paths correctly.
          // Without it, Uri.resolve('/path') treats the path as absolute
          // and drops the '/v1' prefix entirely.
          baseUrl: baseUrl.endsWith('/') ? baseUrl : '$baseUrl/',
          headers: {'Content-Type': 'application/json'},
        ));

  Dio get dio => _dio;

  // Strip a leading '/' so paths like '/console/me' are resolved relative
  // to the baseUrl instead of being treated as absolute host-root paths by
  // Dart's Uri.resolve() (which would drop the '/v1' prefix entirely).
  String _p(String path) => path.startsWith('/') ? path.substring(1) : path;

  Future<Response<T>> get<T>(String path,
          {Map<String, dynamic>? params}) =>
      _dio.get(_p(path), queryParameters: params);

  Future<Response<T>> post<T>(String path, {Object? data}) =>
      _dio.post(_p(path), data: data);

  Future<Response<T>> put<T>(String path, {Object? data}) =>
      _dio.put(_p(path), data: data);

  Future<Response<T>> patch<T>(String path, {Object? data}) =>
      _dio.patch(_p(path), data: data);

  Future<Response<T>> delete<T>(String path) => _dio.delete(_p(path));

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
