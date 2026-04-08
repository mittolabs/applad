import '../server_client.dart';

class Regions {
  final ApplAdServer _client;

  Regions(this._client);

  Future<Map<String, dynamic>> list() async {
    return _client.get('/v1/regions');
  }

  Future<Map<String, dynamic>> get(String regionId) async {
    return _client.get('/v1/regions/$regionId');
  }

  Future<Map<String, dynamic>> getActive() async {
    return _client.get('/v1/regions/active');
  }

  Future<Map<String, dynamic>> setActive(String regionId) async {
    return _client.put('/v1/regions/active', data: {
      'regionId': regionId,
    });
  }

  Future<Map<String, dynamic>> getHealth(String regionId) async {
    return _client.get('/v1/regions/$regionId/health');
  }
}
