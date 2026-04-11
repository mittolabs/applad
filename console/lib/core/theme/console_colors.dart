import 'package:flutter/material.dart';

bool consoleIsLight(BuildContext context) =>
    Theme.of(context).brightness == Brightness.light;

class ConsoleColors {
  final Color background;
  final Color surface;
  final Color surfaceAlt;
  final Color border;
  final Color shadow;
  final Color textPrimary;
  final Color textSecondary;
  final Color textMuted;
  final Color textSubtle;
  final Color fill;
  final Color fillHover;
  final Color fillActive;
  final Color fieldFill;
  final Color fieldBorder;
  final Color popupSurface;
  final Color badgeFill;

  const ConsoleColors({
    required this.background,
    required this.surface,
    required this.surfaceAlt,
    required this.border,
    required this.shadow,
    required this.textPrimary,
    required this.textSecondary,
    required this.textMuted,
    required this.textSubtle,
    required this.fill,
    required this.fillHover,
    required this.fillActive,
    required this.fieldFill,
    required this.fieldBorder,
    required this.popupSurface,
    required this.badgeFill,
  });
}

ConsoleColors consoleColors(BuildContext context) {
  if (consoleIsLight(context)) {
    return ConsoleColors(
      background: Colors.white,
      surface: Colors.white,
      surfaceAlt: const Color(0xFFF7F8FA),
      border: Colors.black.withValues(alpha: 0.09),
      shadow: Colors.black.withValues(alpha: 0.06),
      textPrimary: const Color(0xFF0F1117),
      textSecondary: const Color(0xFF0F1117).withValues(alpha: 0.72),
      textMuted: Colors.black.withValues(alpha: 0.48),
      textSubtle: Colors.black.withValues(alpha: 0.30),
      fill: Colors.black.withValues(alpha: 0.04),
      fillHover: Colors.black.withValues(alpha: 0.06),
      fillActive: const Color(0xFF3472A4).withValues(alpha: 0.10),
      fieldFill: const Color(0xFFF7F8FA),
      fieldBorder: Colors.black.withValues(alpha: 0.12),
      popupSurface: Colors.white,
      badgeFill: Colors.black.withValues(alpha: 0.05),
    );
  }

  return ConsoleColors(
    background: const Color(0xFF0B0B0F),
    surface: const Color(0xFF16171B),
    surfaceAlt: const Color(0xFF101014),
    border: Colors.white.withValues(alpha: 0.08),
    shadow: Colors.black.withValues(alpha: 0.5),
    textPrimary: Colors.white,
    textSecondary: Colors.white.withValues(alpha: 0.72),
    textMuted: Colors.white.withValues(alpha: 0.45),
    textSubtle: Colors.white.withValues(alpha: 0.25),
    fill: Colors.white.withValues(alpha: 0.03),
    fillHover: Colors.white.withValues(alpha: 0.04),
    fillActive: Colors.white.withValues(alpha: 0.08),
    fieldFill: Colors.white.withValues(alpha: 0.04),
    fieldBorder: Colors.white.withValues(alpha: 0.10),
    popupSurface: const Color(0xFF1E1F24),
    badgeFill: Colors.white.withValues(alpha: 0.06),
  );
}
