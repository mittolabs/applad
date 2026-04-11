// ignore: avoid_web_libraries_in_flutter
import 'dart:html' as html;
import 'package:flutter_riverpod/flutter_riverpod.dart';

/// localStorage key for a given project's get-started completion.
String _key(String projectId) => 'gs_done_$projectId';

/// Whether the get-started checklist has been permanently completed for
/// [projectId].  Reads from localStorage on first access; can be updated
/// reactively via [markGetStartedDone].
final getStartedDoneProvider =
    StateProvider.family<bool, String>((ref, projectId) {
  try {
    return html.window.localStorage[_key(projectId)] == '1';
  } catch (_) {
    return false;
  }
});

/// Permanently marks get-started as done for [projectId].
/// Writes to localStorage and updates the in-memory provider state.
/// Accepts either [Ref] or [WidgetRef] via the common [AutoDisposeRef] base.
void markGetStartedDone(String projectId, dynamic ref) {
  try {
    html.window.localStorage[_key(projectId)] = '1';
  } catch (_) {}
  // ignore: avoid_dynamic_calls
  ref.read(getStartedDoneProvider(projectId).notifier).state = true;
}
