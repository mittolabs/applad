import 'package:dio/dio.dart';

class Regions {
  final Dio _dio;

  Regions(this._dio);

  Future<Map<String, dynamic>> list() async {
    final res = await _dio.get('/v1/regions');
    return res.data;
  }

  Future<Map<String, dynamic>> get(String regionId) async {
    final res = await _dio.get('/v1/regions/$regionId');
    return res.data;
  }

  Future<Map<String, dynamic>> getActive() async {
    final res = await _dio.get('/v1/regions/active');
    return res.data;
  }

  Future<Map<String, dynamic>> setActive(String regionId) async {
    final res = await _dio.put('/v1/regions/active', data: {
      'regionId': regionId,
    });
    return res.data;
  }

  Future<Map<String, dynamic>> getHealth(String regionId) async {
    final res = await _dio.get('/v1/regions/$regionId/health');
    return res.data;
  }
}
