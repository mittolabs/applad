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
      background: const Color(0xFFF8F9FA),
      surface: Colors.white,
      surfaceAlt: const Color(0xFFF3F5F8),
      border: Colors.black.withOpacity(0.08),
      shadow: Colors.black.withOpacity(0.08),
      textPrimary: const Color(0xFF1A1A2E),
      textSecondary: const Color(0xFF1A1A2E).withOpacity(0.72),
      textMuted: Colors.black.withOpacity(0.45),
      textSubtle: Colors.black.withOpacity(0.25),
      fill: Colors.black.withOpacity(0.03),
      fillHover: Colors.black.withOpacity(0.04),
      fillActive: const Color(0xFF3472A4).withOpacity(0.1),
      fieldFill: Colors.black.withOpacity(0.03),
      fieldBorder: Colors.black.withOpacity(0.10),
      popupSurface: Colors.white,
      badgeFill: Colors.black.withOpacity(0.04),
    );
  }

  return ConsoleColors(
    background: const Color(0xFF0B0B0F),
    surface: const Color(0xFF16171B),
    surfaceAlt: const Color(0xFF101014),
    border: Colors.white.withOpacity(0.08),
    shadow: Colors.black.withOpacity(0.5),
    textPrimary: Colors.white,
    textSecondary: Colors.white.withOpacity(0.72),
    textMuted: Colors.white.withOpacity(0.45),
    textSubtle: Colors.white.withOpacity(0.25),
    fill: Colors.white.withOpacity(0.03),
    fillHover: Colors.white.withOpacity(0.04),
    fillActive: Colors.white.withOpacity(0.08),
    fieldFill: Colors.white.withOpacity(0.04),
    fieldBorder: Colors.white.withOpacity(0.10),
    popupSurface: const Color(0xFF1E1F24),
    badgeFill: Colors.white.withOpacity(0.06),
  );
}
