import 'dart:html' as html;
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../api/client.dart';

const _tokenKey = 'applad_console_token';

/// Holds the console auth token. Persisted to localStorage.
final consoleTokenProvider = StateProvider<String?>((ref) {
  // Restore from localStorage on app start
  return html.window.localStorage[_tokenKey];
});

/// Holds the console auth state.
final consoleAuthProvider =
    AsyncNotifierProvider<ConsoleAuthNotifier, ConsoleUser?>(
        ConsoleAuthNotifier.new);

class ConsoleUser {
  final String id;
  final String email;
  final String name;

  ConsoleUser({required this.id, required this.email, required this.name});

  factory ConsoleUser.fromJson(Map<String, dynamic> json) {
    return ConsoleUser(
      id: json['\$id'] ?? json['id'] ?? '',
      email: json['email'] ?? '',
      name: json['name'] ?? '',
    );
  }
}

class ConsoleAuthNotifier extends AsyncNotifier<ConsoleUser?> {
  @override
  Future<ConsoleUser?> build() async {
    final token = ref.watch(consoleTokenProvider);
    if (token == null) return null;

    final api = ref.read(apiClientProvider);
    api.setAuthToken(token);
    try {
      final res = await api.get('/console/me');
      final user = ConsoleUser.fromJson(res.data as Map<String, dynamic>);
      api.setConsoleUser(id: user.id, email: user.email, name: user.name);
      return user;
    } catch (_) {
      // Token invalid/expired — clear it
      _clearToken();
      return null;
    }
  }

  Future<void> signup(String email, String password, String name) async {
    final api = ref.read(apiClientProvider);
    final res = await api.post('/console/signup', data: {
      'email': email,
      'password': password,
      'name': name,
    });
    final data = res.data as Map<String, dynamic>;
    final token = data['token'] as String;
    _setToken(token);
    api.setAuthToken(token);
    final user = ConsoleUser.fromJson(data['user'] as Map<String, dynamic>);
    api.setConsoleUser(id: user.id, email: user.email, name: user.name);
    state = AsyncData(user);
  }

  Future<void> login(String email, String password) async {
    final api = ref.read(apiClientProvider);
    final res = await api.post('/console/login', data: {
      'email': email,
      'password': password,
    });
    final data = res.data as Map<String, dynamic>;
    final token = data['token'] as String;
    _setToken(token);
    api.setAuthToken(token);
    final user = ConsoleUser.fromJson(data['user'] as Map<String, dynamic>);
    api.setConsoleUser(id: user.id, email: user.email, name: user.name);
    state = AsyncData(user);
  }

  /// Called after a browser-redirect OAuth flow returns a token via query param.
  Future<void> loginWithToken(String token) async {
    _setToken(token);
    final api = ref.read(apiClientProvider);
    api.setAuthToken(token);
    final res = await api.get('/console/me');
    final user = ConsoleUser.fromJson(res.data as Map<String, dynamic>);
    api.setConsoleUser(id: user.id, email: user.email, name: user.name);
    state = AsyncData(user);
  }

  void logout() {
    _clearToken();
    state = const AsyncData(null);
  }

  void _setToken(String token) {
    ref.read(consoleTokenProvider.notifier).state = token;
    html.window.localStorage[_tokenKey] = token;
  }

  void _clearToken() {
    ref.read(consoleTokenProvider.notifier).state = null;
    html.window.localStorage.remove(_tokenKey);
  }
}

/// Which OAuth providers are configured for console login (e.g. ["github", "google"]).
final consoleOAuthProvidersProvider = FutureProvider<List<String>>((ref) async {
  final api = ref.read(apiClientProvider);
  try {
    final res = await api.get('/console/auth-providers');
    final data = res.data as Map<String, dynamic>;
    return List<String>.from(data['providers'] as List? ?? []);
  } catch (_) {
    return [];
  }
});

/// Whether console signup is enabled.
final signupEnabledProvider = FutureProvider<bool>((ref) async {
  final api = ref.read(apiClientProvider);
  try {
    final res = await api.get('/console/signup-status');
    final data = res.data as Map<String, dynamic>;
    return data['signupEnabled'] == true;
  } catch (_) {
    return false;
  }
});
