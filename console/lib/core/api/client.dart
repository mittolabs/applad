import 'package:dio/dio.dart';

class ApiClient {
  final Dio _dio;

  ApiClient({required String baseUrl})
      : _dio = Dio(BaseOptions(
          baseUrl: baseUrl,
          headers: {'Content-Type': 'application/json'},
        ));

  Future<Response<T>> get<T>(String path, {Map<String, dynamic>? params}) =>
      _dio.get(path, queryParameters: params);

  Future<Response<T>> post<T>(String path, {Object? data}) =>
      _dio.post(path, data: data);

  Future<Response<T>> patch<T>(String path, {Object? data}) =>
      _dio.patch(path, data: data);

  Future<Response<T>> delete<T>(String path) => _dio.delete(path);

  void setAuthToken(String token) {
    _dio.options.headers['Authorization'] = 'Bearer $token';
  }
}
