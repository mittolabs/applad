import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../api/client.dart';
import 'org_provider.dart';

/// Holds the currently selected project ID.
final currentProjectProvider = StateProvider<String?>((ref) => null);

/// Provider for the list of projects — filtered by the current org.
final projectsProvider =
    AsyncNotifierProvider<ProjectsNotifier, List<Map<String, dynamic>>>(
        ProjectsNotifier.new);

class ProjectsNotifier extends AsyncNotifier<List<Map<String, dynamic>>> {
  @override
  Future<List<Map<String, dynamic>>> build() async {
    final api = ref.read(apiClientProvider);
    final orgId = ref.watch(currentOrgProvider);
    final query = orgId != null ? '?orgId=$orgId' : '';
    final res = await api.get('/projects$query');
    final data = res.data as Map<String, dynamic>;
    return List<Map<String, dynamic>>.from(data['projects'] ?? []);
  }

  Future<void> create(String name, String description) async {
    final api = ref.read(apiClientProvider);
    final orgId = ref.read(currentOrgProvider);
    await api.post(
      orgId != null ? '/organizations/$orgId/projects' : '/projects',
      data: {
        'name': name,
        'description': description,
      },
    );
    ref.invalidateSelf();
  }

  Future<void> deleteProject(String id) async {
    final api = ref.read(apiClientProvider);
    await api.delete('/projects/$id');
    ref.invalidateSelf();
  }
}

/// Provider for API keys of the current project.
final apiKeysProvider =
    FutureProvider.family<List<Map<String, dynamic>>, String>(
        (ref, projectId) async {
  final api = ref.read(apiClientProvider);
  final res = await api.get('/projects/$projectId/keys');
  final data = res.data as Map<String, dynamic>;
  return List<Map<String, dynamic>>.from(data['keys'] ?? []);
});
