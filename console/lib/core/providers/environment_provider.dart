// ignore: avoid_web_libraries_in_flutter
import 'dart:html' as html;
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../api/client.dart';
import 'project_provider.dart';

/// Currently selected environment ID. Persisted to localStorage.
final currentEnvironmentProvider =
    StateNotifierProvider<CurrentEnvironmentNotifier, String?>((ref) {
  return CurrentEnvironmentNotifier();
});

class CurrentEnvironmentNotifier extends StateNotifier<String?> {
  CurrentEnvironmentNotifier() : super(_load());

  static String? _load() {
    final stored = html.window.localStorage['applad_current_env'];
    return (stored != null && stored.isNotEmpty) ? stored : null;
  }

  @override
  set state(String? value) {
    super.state = value;
    if (value != null) {
      html.window.localStorage['applad_current_env'] = value;
    } else {
      html.window.localStorage.remove('applad_current_env');
    }
  }
}

/// List of environments for the current project.
final environmentsProvider =
    FutureProvider<List<Map<String, dynamic>>>((ref) async {
  final projectId = ref.watch(currentProjectProvider);
  if (projectId == null) return [];
  final api = ref.read(apiClientProvider);
  try {
    final res = await api.get('/deploy/environments');
    final envs =
        List<Map<String, dynamic>>.from(res.data['environments'] ?? []);
    // Auto-select default if none selected
    final current = ref.read(currentEnvironmentProvider);
    if (current == null && envs.isNotEmpty) {
      final defaultEnv = envs.firstWhere(
        (e) => e['isDefault'] == true,
        orElse: () => envs.first,
      );
      ref.read(currentEnvironmentProvider.notifier).state =
          defaultEnv['\$id'] as String;
    }
    return envs;
  } catch (_) {
    return [];
  }
});
