// ignore: avoid_web_libraries_in_flutter
import 'dart:html' as html;
import 'dart:convert';
import 'package:flutter_riverpod/flutter_riverpod.dart';

/// Experiments that can be enabled/disabled.
/// Stored in localStorage so they persist across sessions.
class Experiments {
  final bool specify;
  final bool design;
  final bool test;
  final bool observe;
  final bool aiChat;
  final bool search;
  final bool analytics;
  final bool cache;
  final bool billing;
  final bool cms;
  final bool edgeFunctions;
  final bool scheduledJobs;
  final bool vectors;
  final bool regions;

  const Experiments({
    this.specify = false,
    this.design = false,
    this.test = false,
    this.observe = false,
    this.aiChat = false,
    this.search = false,
    this.analytics = false,
    this.cache = false,
    this.billing = false,
    this.cms = false,
    this.edgeFunctions = false,
    this.scheduledJobs = false,
    this.vectors = false,
    this.regions = false,
  });

  Experiments copyWith({
    bool? specify,
    bool? design,
    bool? test,
    bool? observe,
    bool? aiChat,
    bool? search,
    bool? analytics,
    bool? cache,
    bool? billing,
    bool? cms,
    bool? edgeFunctions,
    bool? scheduledJobs,
    bool? vectors,
    bool? regions,
  }) {
    return Experiments(
      specify: specify ?? this.specify,
      design: design ?? this.design,
      test: test ?? this.test,
      observe: observe ?? this.observe,
      aiChat: aiChat ?? this.aiChat,
      search: search ?? this.search,
      analytics: analytics ?? this.analytics,
      cache: cache ?? this.cache,
      billing: billing ?? this.billing,
      cms: cms ?? this.cms,
      edgeFunctions: edgeFunctions ?? this.edgeFunctions,
      scheduledJobs: scheduledJobs ?? this.scheduledJobs,
      vectors: vectors ?? this.vectors,
      regions: regions ?? this.regions,
    );
  }

  Map<String, bool> toMap() => {
        'specify': specify,
        'design': design,
        'test': test,
        'observe': observe,
        'aiChat': aiChat,
        'search': search,
        'analytics': analytics,
        'cache': cache,
        'billing': billing,
        'cms': cms,
        'edgeFunctions': edgeFunctions,
        'scheduledJobs': scheduledJobs,
        'vectors': vectors,
        'regions': regions,
      };

  factory Experiments.fromMap(Map<String, dynamic> m) => Experiments(
        specify: m['specify'] == true,
        design: m['design'] == true,
        test: m['test'] == true,
        observe: m['observe'] == true,
        aiChat: m['aiChat'] == true,
        search: m['search'] == true,
        analytics: m['analytics'] == true,
        cache: m['cache'] == true,
        billing: m['billing'] == true,
        cms: m['cms'] == true,
        edgeFunctions: m['edgeFunctions'] == true,
        scheduledJobs: m['scheduledJobs'] == true,
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
    for (final k in m.keys) {
      m[k] = true;
    }
    state = Experiments.fromMap(m);
    _save();
  }

  void disableAll() {
    state = const Experiments();
    _save();
  }
}
