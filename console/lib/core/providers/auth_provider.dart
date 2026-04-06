import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../api/client.dart';

/// Holds the console auth token (null = not logged in).
final consoleTokenProvider = StateProvider<String?>((ref) => null);

/// Holds the console auth state (loading, logged in, logged out).
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
      return ConsoleUser.fromJson(res.data as Map<String, dynamic>);
    } catch (_) {
      // Token invalid/expired
      ref.read(consoleTokenProvider.notifier).state = null;
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
    ref.read(consoleTokenProvider.notifier).state = token;
    api.setAuthToken(token);
    state = AsyncData(ConsoleUser.fromJson(data['user'] as Map<String, dynamic>));
  }

  Future<void> login(String email, String password) async {
    final api = ref.read(apiClientProvider);
    final res = await api.post('/console/login', data: {
      'email': email,
      'password': password,
    });
    final data = res.data as Map<String, dynamic>;
    final token = data['token'] as String;
    ref.read(consoleTokenProvider.notifier).state = token;
    api.setAuthToken(token);
    state = AsyncData(ConsoleUser.fromJson(data['user'] as Map<String, dynamic>));
  }

  void logout() {
    ref.read(consoleTokenProvider.notifier).state = null;
    state = const AsyncData(null);
  }
}

/// Whether console signup is enabled (fetched from backend).
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
