import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_web_plugins/flutter_web_plugins.dart';
import 'core/router/router.dart';
import 'core/providers/theme_provider.dart';

const _accent = Color(0xFF3472A4);
const _bg = Color(0xFF0B0B0F);
const _surface = Color(0xFF16171B);
const _lightBg = Color(0xFFF8F9FA);
const _lightSurface = Color(0xFFFFFFFF);
const _radius = 8.0;
final _shape = RoundedRectangleBorder(borderRadius: BorderRadius.circular(_radius));

class ApplAdApp extends ConsumerWidget {
  const ApplAdApp({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final router = ref.watch(routerProvider);
    final isLight = ref.watch(themeModeProvider);
    return MaterialApp.router(
      title: 'Applad',
      debugShowCheckedModeBanner: false,
      theme: _lightTheme,
      darkTheme: _darkTheme,
      themeMode: isLight ? ThemeMode.light : ThemeMode.dark,
      routerConfig: router,
      scrollBehavior: const MaterialScrollBehavior().copyWith(
        physics: const ClampingScrollPhysics(),
      ),
    );
  }

  static final _darkTheme = ThemeData(
    useMaterial3: true,
    brightness: Brightness.dark,
    scaffoldBackgroundColor: _bg,
    colorScheme: ColorScheme.fromSeed(
      seedColor: _accent,
      brightness: Brightness.dark,
      surface: _surface,
    ),
    cardColor: _surface,

    // AppBar
    appBarTheme: const AppBarTheme(
      backgroundColor: _bg,
      foregroundColor: Colors.white,
      elevation: 0,
      scrolledUnderElevation: 0,
    ),

    // Dialogs — sharp corners, dark background, X close button style
    dialogTheme: DialogThemeData(
      backgroundColor: _surface,
      shape: _shape,
      titleTextStyle: const TextStyle(
        color: Colors.white,
        fontSize: 18,
        fontWeight: FontWeight.w600,
      ),
    ),

    // FilledButton — accent purple, sharp corners
    filledButtonTheme: FilledButtonThemeData(
      style: FilledButton.styleFrom(
        backgroundColor: _accent,
        foregroundColor: Colors.white,
        shape: _shape,
        padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 12),
        textStyle: const TextStyle(fontSize: 14, fontWeight: FontWeight.w600),
      ),
    ),

    // ElevatedButton — same sharp corners
    elevatedButtonTheme: ElevatedButtonThemeData(
      style: ElevatedButton.styleFrom(
        backgroundColor: _surface,
        foregroundColor: Colors.white,
        shape: _shape,
        padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 12),
        textStyle: const TextStyle(fontSize: 14, fontWeight: FontWeight.w600),
      ),
    ),

    // OutlinedButton — sharp corners, subtle border
    outlinedButtonTheme: OutlinedButtonThemeData(
      style: OutlinedButton.styleFrom(
        foregroundColor: Colors.white,
        shape: _shape,
        side: BorderSide(color: Colors.white.withOpacity(0.12)),
        padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 12),
        textStyle: const TextStyle(fontSize: 14, fontWeight: FontWeight.w600),
      ),
    ),

    // TextButton — no rounding
    textButtonTheme: TextButtonThemeData(
      style: TextButton.styleFrom(
        foregroundColor: Colors.white70,
        shape: _shape,
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
        textStyle: const TextStyle(fontSize: 14),
      ),
    ),

    // Input fields — sharp corners, dark fill
    inputDecorationTheme: InputDecorationTheme(
      filled: true,
      fillColor: Colors.white.withOpacity(0.04),
      contentPadding: const EdgeInsets.symmetric(horizontal: 14, vertical: 14),
      border: OutlineInputBorder(
        borderRadius: BorderRadius.circular(_radius),
        borderSide: BorderSide(color: Colors.white.withOpacity(0.1)),
      ),
      enabledBorder: OutlineInputBorder(
        borderRadius: BorderRadius.circular(_radius),
        borderSide: BorderSide(color: Colors.white.withOpacity(0.1)),
      ),
      focusedBorder: OutlineInputBorder(
        borderRadius: BorderRadius.circular(_radius),
        borderSide: const BorderSide(color: _accent),
      ),
      hintStyle: TextStyle(color: Colors.white.withOpacity(0.25)),
    ),

    // Chips
    chipTheme: ChipThemeData(
      backgroundColor: Colors.white.withOpacity(0.06),
      shape: _shape,
    ),

    // Cards
    cardTheme: CardThemeData(
      color: _surface,
      shape: _shape,
      elevation: 0,
    ),

    // PopupMenu
    popupMenuTheme: PopupMenuThemeData(
      color: _surface,
      shape: _shape,
    ),

    // DropdownMenu
    dropdownMenuTheme: DropdownMenuThemeData(
      menuStyle: MenuStyle(
        backgroundColor: WidgetStatePropertyAll(_surface),
        shape: WidgetStatePropertyAll(_shape),
      ),
    ),

    // Divider
    dividerColor: Colors.white.withOpacity(0.06),

    // Snackbar
    snackBarTheme: SnackBarThemeData(
      backgroundColor: _surface,
      contentTextStyle: const TextStyle(color: Colors.white),
      shape: _shape,
      behavior: SnackBarBehavior.floating,
    ),
  );

  static final _lightTheme = ThemeData(
    useMaterial3: true,
    brightness: Brightness.light,
    scaffoldBackgroundColor: _lightBg,
    colorScheme: ColorScheme.fromSeed(
      seedColor: _accent,
      brightness: Brightness.light,
      surface: _lightSurface,
    ),
    cardColor: _lightSurface,
    appBarTheme: const AppBarTheme(
      backgroundColor: _lightSurface,
      foregroundColor: Color(0xFF1A1A2E),
      elevation: 0,
      scrolledUnderElevation: 0,
    ),
    dialogTheme: DialogThemeData(
      backgroundColor: _lightSurface,
      shape: _shape,
      titleTextStyle: const TextStyle(
        color: Color(0xFF1A1A2E),
        fontSize: 18,
        fontWeight: FontWeight.w600,
      ),
    ),
    filledButtonTheme: FilledButtonThemeData(
      style: FilledButton.styleFrom(
        backgroundColor: _accent,
        foregroundColor: Colors.white,
        shape: _shape,
        padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 12),
        textStyle: const TextStyle(fontSize: 14, fontWeight: FontWeight.w600),
      ),
    ),
    outlinedButtonTheme: OutlinedButtonThemeData(
      style: OutlinedButton.styleFrom(
        foregroundColor: const Color(0xFF1A1A2E),
        shape: _shape,
        side: BorderSide(color: Colors.black.withOpacity(0.12)),
        padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 12),
      ),
    ),
    textButtonTheme: TextButtonThemeData(
      style: TextButton.styleFrom(
        foregroundColor: const Color(0xFF555),
        shape: _shape,
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
      ),
    ),
    inputDecorationTheme: InputDecorationTheme(
      filled: true,
      fillColor: Colors.black.withOpacity(0.03),
      contentPadding: const EdgeInsets.symmetric(horizontal: 14, vertical: 14),
      border: OutlineInputBorder(
        borderRadius: BorderRadius.circular(_radius),
        borderSide: BorderSide(color: Colors.black.withOpacity(0.12)),
      ),
      enabledBorder: OutlineInputBorder(
        borderRadius: BorderRadius.circular(_radius),
        borderSide: BorderSide(color: Colors.black.withOpacity(0.12)),
      ),
      focusedBorder: OutlineInputBorder(
        borderRadius: BorderRadius.circular(_radius),
        borderSide: const BorderSide(color: _accent),
      ),
      hintStyle: TextStyle(color: Colors.black.withOpacity(0.35)),
    ),
    chipTheme: ChipThemeData(
      backgroundColor: Colors.black.withOpacity(0.04),
      shape: _shape,
    ),
    cardTheme: CardThemeData(
      color: _lightSurface,
      shape: _shape,
      elevation: 1,
    ),
    popupMenuTheme: PopupMenuThemeData(
      color: _lightSurface,
      shape: _shape,
    ),
    dividerColor: Colors.black.withOpacity(0.08),
    snackBarTheme: SnackBarThemeData(
      backgroundColor: const Color(0xFF1A1A2E),
      contentTextStyle: const TextStyle(color: Colors.white),
      shape: _shape,
      behavior: SnackBarBehavior.floating,
    ),
  );
}

/// Call this in main() before runApp to use path-based URLs.
void configureUrlStrategy() {
  usePathUrlStrategy();
}
