import 'package:flutter/widgets.dart';
import 'package:go_router/go_router.dart';

/// Returns the current URL with [updates] applied to query parameters.
/// A null value removes the key; a value of '1' for 'page' is also removed
/// (page 1 is the default, so no need to clutter the URL).
String withQuery(BuildContext context, Map<String, String?> updates) {
  final uri = GoRouterState.of(context).uri;
  final params = Map<String, String>.from(uri.queryParameters);
  for (final e in updates.entries) {
    if (e.value == null || (e.key == 'page' && e.value == '1')) {
      params.remove(e.key);
    } else {
      params[e.key] = e.value!;
    }
  }
  return Uri(
    path: uri.path,
    queryParameters: params.isEmpty ? null : params,
  ).toString();
}

/// Reads the current page number from `?page=N`, defaulting to 1.
int pageFromQuery(BuildContext context) =>
    int.tryParse(
      GoRouterState.of(context).uri.queryParameters['page'] ?? '1',
    ) ??
    1;

/// Reads a named tab from `?tab=name`, returning [defaultTab] if absent.
String tabFromQuery(BuildContext context, {String defaultTab = ''}) =>
    GoRouterState.of(context).uri.queryParameters['tab'] ?? defaultTab;

/// Responsive horizontal page padding.
/// Mobile (<650): 16px · Narrow (<1100): 32px · Normal (<1400): 48px · Wide: 64px
double pageHPad(BuildContext context) {
  final w = MediaQuery.sizeOf(context).width;
  if (w < 650) return 16.0;
  if (w > 1400) return 64.0;
  if (w > 1100) return 48.0;
  return 32.0;
}
