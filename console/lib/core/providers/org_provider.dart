// ignore: avoid_web_libraries_in_flutter, deprecated_member_use
import 'dart:html' as html;
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../api/client.dart';
import 'auth_provider.dart';

/// Holds the currently selected organization ID.
/// Persisted to localStorage so it survives page refresh.
final currentOrgProvider =
    StateNotifierProvider<CurrentOrgNotifier, String?>((ref) {
  return CurrentOrgNotifier();
});

class CurrentOrgNotifier extends StateNotifier<String?> {
  CurrentOrgNotifier() : super(_load());

  static String? _load() {
    final stored = html.window.localStorage['applad_current_org'];
    return (stored != null && stored.isNotEmpty) ? stored : null;
  }

  @override
  set state(String? value) {
    super.state = value;
    if (value != null) {
      html.window.localStorage['applad_current_org'] = value;
    } else {
      html.window.localStorage.remove('applad_current_org');
    }
  }
}

/// Provider for the list of organizations.
/// Waits for auth to be ready before fetching (ensures X-Console-User-ID is set).
/// Auto-selects the first org if none is selected.
final orgsProvider = FutureProvider<List<Map<String, dynamic>>>((ref) async {
  // Wait for auth — this ensures setConsoleUser() has been called
  final user = ref.watch(consoleAuthProvider).valueOrNull;
  if (user == null) return [];

  final api = ref.read(apiClientProvider);
  try {
    final res = await api.get('/organizations');
    final orgs =
        List<Map<String, dynamic>>.from(res.data['organizations'] ?? []);

    // Auto-select first org if none selected
    final currentOrg = ref.read(currentOrgProvider);
    if (currentOrg == null && orgs.isNotEmpty) {
      final firstId = orgs.first['\$id'] as String?;
      if (firstId != null) {
        ref.read(currentOrgProvider.notifier).state = firstId;
      }
    }

    return orgs;
  } catch (_) {
    return [];
  }
});
