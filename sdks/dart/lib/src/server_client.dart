import 'dart:convert';

import 'package:http/http.dart' as http;

import 'server/analytics.dart';
import 'server/users.dart';
import 'server/databases.dart';
import 'server/storage.dart';
import 'server/functions.dart';
import 'server/teams.dart';
import 'server/workflows.dart';
import 'server/messaging.dart';
import 'server/deploy.dart';
import 'server/edge.dart';
import 'server/flags.dart';
import 'server/regions.dart';
import 'server/search.dart';
import 'server/vectors.dart';

/// Exception thrown when the Applad API returns a non-2xx status code.
class AppladException implements Exception {
  final int statusCode;
  final String message;
  final String? type;

  AppladException({
    required this.statusCode,
    required this.message,
    this.type,
  });

  @override
  String toString() =>
      'AppladException($statusCode): $message${type != null ? ' [$type]' : ''}';
}

/// Server-side Applad client that uses API key authentication.
///
/// Uses `package:http` — no Flutter dependency.
///
/// ```dart
/// final server = ApplAdServer(
///   endpoint: 'https://applad.example.com',
///   projectId: 'my-project',
///   apiKey: 'applad_key_abc123',
/// );
///
/// final users = await server.users.listUsers();
/// ```
class ApplAdServer {
  final String endpoint;
  final String projectId;
  final String apiKey;
  final http.Client _httpClient;

  late final Analytics analytics;
  late final Users users;
  late final Databases databases;
  late final Storage storage;
  late final Functions functions;
  late final Teams teams;
  late final Workflows workflows;
  late final Messaging messaging;
  late final Deploy deploy;
  late final Edge edge;
  late final Flags flags;
  late final Regions regions;
  late final Search search;
  late final Vectors vectors;

  ApplAdServer({
    required this.endpoint,
    required this.projectId,
    required this.apiKey,
    http.Client? httpClient,
  }) : _httpClient = httpClient ?? http.Client() {
    analytics = Analytics(this);
    users = Users(this);
    databases = Databases(this);
    storage = Storage(this);
    functions = Functions(this);
    teams = Teams(this);
    workflows = Workflows(this);
    messaging = Messaging(this);
    deploy = Deploy(this);
    edge = Edge(this);
    flags = Flags(this);
    regions = Regions(this);
    search = Search(this);
    vectors = Vectors(this);
  }

  Map<String, String> get _headers => {
        'X-Applad-Project': projectId,
        'X-Applad-Key': apiKey,
        'Content-Type': 'application/json',
      };

  Uri _uri(String path) => Uri.parse('$endpoint$path');

  Map<String, dynamic> _parseResponse(http.Response response) {
    final body = response.body.isNotEmpty
        ? jsonDecode(response.body) as Map<String, dynamic>
        : <String, dynamic>{};

    if (response.statusCode < 200 || response.statusCode >= 300) {
      throw AppladException(
        statusCode: response.statusCode,
        message: body['message']?.toString() ??
            response.reasonPhrase ??
            'Unknown error',
        type: body['type']?.toString(),
      );
    }

    return body;
  }

  /// Send a GET request.
  Future<Map<String, dynamic>> get(String path) async {
    final response = await _httpClient.get(_uri(path), headers: _headers);
    return _parseResponse(response);
  }

  /// Send a POST request.
  Future<Map<String, dynamic>> post(String path,
      {Map<String, dynamic>? data}) async {
    final response = await _httpClient.post(
      _uri(path),
      headers: _headers,
      body: data != null ? jsonEncode(data) : null,
    );
    return _parseResponse(response);
  }

  /// Send a PUT request.
  Future<Map<String, dynamic>> put(String path,
      {Map<String, dynamic>? data}) async {
    final response = await _httpClient.put(
      _uri(path),
      headers: _headers,
      body: data != null ? jsonEncode(data) : null,
    );
    return _parseResponse(response);
  }

  /// Send a PATCH request.
  Future<Map<String, dynamic>> patch(String path,
      {Map<String, dynamic>? data}) async {
    final response = await _httpClient.patch(
      _uri(path),
      headers: _headers,
      body: data != null ? jsonEncode(data) : null,
    );
    return _parseResponse(response);
  }

  /// Send a DELETE request.
  Future<Map<String, dynamic>> delete(String path) async {
    final response = await _httpClient.delete(_uri(path), headers: _headers);
    return _parseResponse(response);
  }

  /// Close the underlying HTTP client.
  void close() {
    _httpClient.close();
  }
}
