import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../api/client.dart';

/// Holds the currently selected organization ID.
final currentOrgProvider = StateProvider<String?>((ref) => null);

/// Provider for the list of organizations.
final orgsProvider = FutureProvider<List<Map<String, dynamic>>>((ref) async {
  final api = ref.read(apiClientProvider);
  try {
    final res = await api.get('/organizations');
    return List<Map<String, dynamic>>.from(res.data['organizations'] ?? []);
  } catch (_) {
    return [];
  }
});
