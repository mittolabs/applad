// ignore: avoid_web_libraries_in_flutter
import 'dart:html' as html;
import 'package:flutter_riverpod/flutter_riverpod.dart';

final themeModeProvider = StateNotifierProvider<ThemeModeNotifier, bool>((ref) {
  return ThemeModeNotifier();
});

class ThemeModeNotifier extends StateNotifier<bool> {
  ThemeModeNotifier() : super(_loadFromStorage());

  static bool _loadFromStorage() {
    final stored = html.window.localStorage['applad_theme'];
    return stored == 'light';
  }

  /// true = light mode, false = dark mode
  bool get isLight => state;

  void toggle() {
    state = !state;
    html.window.localStorage['applad_theme'] = state ? 'light' : 'dark';
  }

  void setLight() {
    state = true;
    html.window.localStorage['applad_theme'] = 'light';
  }

  void setDark() {
    state = false;
    html.window.localStorage['applad_theme'] = 'dark';
  }
}
