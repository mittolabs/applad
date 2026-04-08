import 'package:test/test.dart';
import 'package:applad/applad.dart';
import 'package:applad/applad_server.dart' as server;

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
      expect(client.analytics, isA<Analytics>());
      expect(client.auth, isA<Auth>());
      expect(client.billing, isA<Billing>());
      expect(client.users, isA<Users>());
      expect(client.databases, isA<Databases>());
      expect(client.edge, isA<Edge>());
      expect(client.flags, isA<Flags>());
      expect(client.functions, isA<Functions>());
      expect(client.messaging, isA<Messaging>());
      expect(client.regions, isA<Regions>());
      expect(client.search, isA<Search>());
      expect(client.storage, isA<Storage>());
      expect(client.vectors, isA<Vectors>());
      expect(client.workflows, isA<Workflows>());
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

  group('ApplAdServer', () {
    test('creates with endpoint, projectId, and apiKey', () {
      final srv = server.ApplAdServer(
        endpoint: 'http://localhost:8080',
        projectId: 'test-project',
        apiKey: 'applad_key_abc123',
      );
      expect(srv.endpoint, 'http://localhost:8080');
      expect(srv.projectId, 'test-project');
      expect(srv.apiKey, 'applad_key_abc123');
    });

    test('exposes server service instances', () {
      final srv = server.ApplAdServer(
        endpoint: 'http://localhost:8080',
        projectId: 'test-project',
        apiKey: 'applad_key_abc123',
      );
      expect(srv.analytics, isNotNull);
      expect(srv.billing, isNotNull);
      expect(srv.users, isNotNull);
      expect(srv.databases, isNotNull);
      expect(srv.storage, isNotNull);
      expect(srv.edge, isNotNull);
      expect(srv.functions, isNotNull);
      expect(srv.teams, isNotNull);
      expect(srv.workflows, isNotNull);
      expect(srv.messaging, isNotNull);
      expect(srv.deploy, isNotNull);
      expect(srv.flags, isNotNull);
      expect(srv.regions, isNotNull);
      expect(srv.search, isNotNull);
      expect(srv.vectors, isNotNull);
    });

    test('can be closed without error', () {
      final srv = server.ApplAdServer(
        endpoint: 'http://localhost:8080',
        projectId: 'test',
        apiKey: 'applad_key_test',
      );
      expect(() => srv.close(), returnsNormally);
    });
  });
}
