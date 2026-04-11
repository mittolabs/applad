// ignore: avoid_web_libraries_in_flutter, deprecated_member_use
import 'dart:html' as html;
import 'dart:convert';
import 'package:flutter_riverpod/flutter_riverpod.dart';

/// Feature flags for unreleased backend services.
/// Stored in localStorage so they persist across sessions.
class Experiments {
  final bool aiChat;
  final bool search;
  final bool analytics;
  final bool cache;
  final bool billing;
  final bool edgeFunctions;
  final bool vectors;
  final bool regions;

  const Experiments({
    this.aiChat = false,
    this.search = false,
    this.analytics = false,
    this.cache = false,
    this.billing = false,
    this.edgeFunctions = false,
    this.vectors = false,
    this.regions = false,
  });

  Experiments copyWith({
    bool? aiChat,
    bool? search,
    bool? analytics,
    bool? cache,
    bool? billing,
    bool? edgeFunctions,
    bool? vectors,
    bool? regions,
  }) {
    return Experiments(
      aiChat: aiChat ?? this.aiChat,
      search: search ?? this.search,
      analytics: analytics ?? this.analytics,
      cache: cache ?? this.cache,
      billing: billing ?? this.billing,
      edgeFunctions: edgeFunctions ?? this.edgeFunctions,
      vectors: vectors ?? this.vectors,
      regions: regions ?? this.regions,
    );
  }

  Map<String, bool> toMap() => {
        'aiChat': aiChat,
        'search': search,
        'analytics': analytics,
        'cache': cache,
        'billing': billing,
        'edgeFunctions': edgeFunctions,
        'vectors': vectors,
        'regions': regions,
      };

  factory Experiments.fromMap(Map<String, dynamic> m) => Experiments(
        aiChat: m['aiChat'] == true,
        search: m['search'] == true,
        analytics: m['analytics'] == true,
        cache: m['cache'] == true,
        billing: m['billing'] == true,
        edgeFunctions: m['edgeFunctions'] == true,
        vectors: m['vectors'] == true,
        regions: m['regions'] == true,
      );
}

final experimentsProvider =
    StateNotifierProvider<ExperimentsNotifier, Experiments>((ref) {
  return ExperimentsNotifier();
});

class ExperimentsNotifier extends StateNotifier<Experiments> {
  ExperimentsNotifier() : super(_load());

  static Experiments _load() {
    final stored = html.window.localStorage['applad_experiments'];
    if (stored != null) {
      try {
        return Experiments.fromMap(
            json.decode(stored) as Map<String, dynamic>);
      } catch (_) {}
    }
    return const Experiments();
  }

  void _save() {
    html.window.localStorage['applad_experiments'] =
        json.encode(state.toMap());
  }

  void toggle(String key) {
    final m = state.toMap();
    m[key] = !(m[key] ?? false);
    state = Experiments.fromMap(m);
    _save();
  }

  void enableAll() {
    final m = state.toMap();
    for (final k in m.keys) { m[k] = true; }
    state = Experiments.fromMap(m);
    _save();
  }

  void disableAll() {
    state = const Experiments();
    _save();
  }
}
