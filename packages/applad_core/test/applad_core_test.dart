import 'dart:io';
import 'package:test/test.dart';
import 'package:applad_core/applad_core.dart';
import 'package:path/path.dart' as p;

void main() {
  group('SecretRef', () {
    test('parses valid secret reference', () {
      final ref = SecretRef.parse('\${MY_SECRET}');
      expect(ref.name, equals('MY_SECRET'));
    });

    test('identifies secret reference strings', () {
      expect(SecretRef.isSecretRef('\${MY_SECRET}'), isTrue);
      expect(SecretRef.isSecretRef('plain-value'), isFalse);
    });

    test('toString returns correct format', () {
      const ref = SecretRef('MY_SECRET');
      expect(ref.toString(), equals('\${MY_SECRET}'));
    });
  });

  group('Environment', () {
    test('parses environment strings', () {
      expect(Environment.fromString('development'), equals(Environment.development));
      expect(Environment.fromString('dev'), equals(Environment.development));
      expect(Environment.fromString('production'), equals(Environment.production));
      expect(Environment.fromString('prod'), equals(Environment.production));
      expect(Environment.fromString('staging'), equals(Environment.staging));
    });

    test('throws on unknown environment', () {
      expect(() => Environment.fromString('unknown'), throwsArgumentError);
    });
  });

  group('ApplAdError', () {
    test('ConfigError includes file path in message', () {
      const error = ConfigError('File not found', filePath: '/path/to/file.yaml', lineNumber: 42);
      expect(error.toString(), contains('/path/to/file.yaml'));
      expect(error.toString(), contains('42'));
    });

    test('ValidationError lists violations', () {
      const error = ValidationError('Validation failed', violations: [
        ValidationViolation(path: 'org.yaml > id', message: 'Required'),
      ]);
      expect(error.toString(), contains('org.yaml > id'));
    });
  });

  group('ConfigLoader', () {
    late Directory tempDir;

    setUp(() {
      tempDir = Directory.systemTemp.createTempSync('applad_test_');
    });

    tearDown(() {
      tempDir.deleteSync(recursive: true);
    });

    test('loads a valid YAML file', () {
      final file = File(p.join(tempDir.path, 'test.yaml'));
      file.writeAsStringSync('key: value\nnumber: 42');

      const loader = ConfigLoader();
      final result = loader.loadFile(file.path);
      expect(result['key'], equals('value'));
      expect(result['number'], equals(42));
    });

    test('throws ConfigError on missing file', () {
      const loader = ConfigLoader();
      expect(
        () => loader.loadFile('/nonexistent/file.yaml'),
        throwsA(isA<ConfigError>()),
      );
    });

    test('returns empty map for empty file', () {
      final file = File(p.join(tempDir.path, 'empty.yaml'));
      file.writeAsStringSync('');

      const loader = ConfigLoader();
      final result = loader.loadFile(file.path);
      expect(result, isEmpty);
    });
  });
}
