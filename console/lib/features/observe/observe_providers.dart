import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../core/api/client.dart';
import '../../core/providers/project_provider.dart';

// ─── Data providers ───────────────────────────────────────────────────────────

final overviewProvider = FutureProvider<Map<String, dynamic>>((ref) async {
  if (ref.watch(currentProjectProvider) == null) return {};
  final res = await ref.read(apiClientProvider).get('/observe/overview');
  return res.data as Map<String, dynamic>;
});

final errorsProvider = FutureProvider<Map<String, dynamic>>((ref) async {
  if (ref.watch(currentProjectProvider) == null) return {'errors': [], 'total': 0};
  final res = await ref.read(apiClientProvider).get('/observe/errors', params: {'limit': 50});
  return res.data as Map<String, dynamic>;
});

final logsProvider = FutureProvider<Map<String, dynamic>>((ref) async {
  if (ref.watch(currentProjectProvider) == null) return {'logs': [], 'total': 0};
  final res = await ref.read(apiClientProvider).get('/observe/logs', params: {'limit': 100});
  return res.data as Map<String, dynamic>;
});

final performanceProvider = FutureProvider<Map<String, dynamic>>((ref) async {
  if (ref.watch(currentProjectProvider) == null) return {};
  final res = await ref.read(apiClientProvider).get('/observe/performance');
  return res.data as Map<String, dynamic>;
});

final releasesProvider = FutureProvider<Map<String, dynamic>>((ref) async {
  if (ref.watch(currentProjectProvider) == null) return {'releases': []};
  final res = await ref.read(apiClientProvider).get('/observe/releases');
  return res.data as Map<String, dynamic>;
});

final replaysProvider = FutureProvider<Map<String, dynamic>>((ref) async {
  if (ref.watch(currentProjectProvider) == null) return {'replays': []};
  final res = await ref.read(apiClientProvider).get('/observe/replays');
  return res.data as Map<String, dynamic>;
});

final uptimeProvider = FutureProvider<Map<String, dynamic>>((ref) async {
  if (ref.watch(currentProjectProvider) == null) return {'monitors': []};
  final res = await ref.read(apiClientProvider).get('/observe/uptime');
  return res.data as Map<String, dynamic>;
});

final cronsProvider = FutureProvider<Map<String, dynamic>>((ref) async {
  if (ref.watch(currentProjectProvider) == null) return {'monitors': []};
  final res = await ref.read(apiClientProvider).get('/observe/crons');
  return res.data as Map<String, dynamic>;
});

final alertsProvider = FutureProvider<Map<String, dynamic>>((ref) async {
  if (ref.watch(currentProjectProvider) == null) return {'rules': [], 'incidents': []};
  final res = await ref.read(apiClientProvider).get('/observe/alerts');
  return res.data as Map<String, dynamic>;
});

// ─── UI state ─────────────────────────────────────────────────────────────────

final errStatusFilterProvider  = StateProvider<String>((ref) => 'unresolved');
final errLevelFilterProvider   = StateProvider<String>((ref) => '');
final selectedErrorIdProvider  = StateProvider<String?>((ref) => null);

final logLevelFilterProvider   = StateProvider<String>((ref) => '');
final logSourceFilterProvider  = StateProvider<String>((ref) => '');

final selectedReleaseIdProvider = StateProvider<String?>((ref) => null);
