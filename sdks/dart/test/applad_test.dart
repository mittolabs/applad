import 'package:test/test.dart';
import 'package:applad/applad.dart';

void main() {
  group('Applad client', () {
    test('creates with endpoint and projectId', () {
      final client = Applad(
        endpoint: 'http://localhost:8080',
        projectId: 'test-project',
      );
      expect(client.endpoint, 'http://localhost:8080');
      expect(client.projectId, 'test-project');
    });

    test('exposes service instances', () {
      final client = Applad(
        endpoint: 'http://localhost:8080',
        projectId: 'test-project',
      );
      expect(client.auth, isA<Auth>());
      expect(client.users, isA<Users>());
      expect(client.databases, isA<Databases>());
      expect(client.storage, isA<Storage>());
    });

    test('sets JWT header', () {
      final client = Applad(
        endpoint: 'http://localhost:8080',
        projectId: 'test',
      );
      client.setJWT('my-token');
      expect(
        client.dio.options.headers['Authorization'],
        'Bearer my-token',
      );
    });

    test('sets API key header', () {
      final client = Applad(
        endpoint: 'http://localhost:8080',
        projectId: 'test',
      );
      client.setKey('applad_key_abc123');
      expect(
        client.dio.options.headers['x-applad-key'],
        'applad_key_abc123',
      );
    });

    test('sets project header on init', () {
      final client = Applad(
        endpoint: 'http://localhost:8080',
        projectId: 'my-project',
      );
      expect(
        client.dio.options.headers['x-applad-project'],
        'my-project',
      );
    });
  });
}
