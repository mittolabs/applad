import 'dart:math' as math;
import 'package:flutter/material.dart';
import 'package:lucide_icons_flutter/lucide_icons.dart';
import '../../core/theme/console_colors.dart';
import '../../core/widgets/app_dropdown.dart';

// ─── Palette ──────────────────────────────────────────────────────────────────

const obAccent = Color(0xFF3472A4);
const obGreen  = Color(0xFF10B981);
const obRed    = Color(0xFFEF4444);
const obOrange = Color(0xFFF59E0B);
const obPurple = Color(0xFF8B5CF6);

// ─── Helpers ──────────────────────────────────────────────────────────────────

String obTimeAgo(dynamic v) {
  if (v == null) return '—';
  final dt = DateTime.tryParse(v.toString());
  if (dt == null) return v.toString();
  final d = DateTime.now().difference(dt);
  if (d.inSeconds < 60)  return 'just now';
  if (d.inMinutes < 60)  return '${d.inMinutes}m ago';
  if (d.inHours   < 24)  return '${d.inHours}h ago';
  return '${d.inDays}d ago';
}

String obFmtNum(dynamic v) {
  final n = (v is num) ? v.toInt() : int.tryParse('$v') ?? 0;
  if (n >= 1000000) return '${(n / 1000000).toStringAsFixed(1)}M';
  if (n >= 1000)    return '${(n / 1000).toStringAsFixed(1)}k';
  return '$n';
}

// ─── Filter chip ─────────────────────────────────────────────────────────────

class ObFilterChip extends StatelessWidget {
  final String label;
  final bool active;
  final VoidCallback onTap;
  final ConsoleColors colors;
  const ObFilterChip({
    super.key,
    required this.label,
    required this.active,
    required this.onTap,
    required this.colors,
  });

  @override
  Widget build(BuildContext context) => InkWell(
        onTap: onTap,
        borderRadius: BorderRadius.circular(5),
        child: Container(
          padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
          decoration: BoxDecoration(
              color: active ? obAccent.withValues(alpha: 0.1) : colors.surface,
              borderRadius: BorderRadius.circular(5),
              border: Border.all(
                  color: active ? obAccent.withValues(alpha: 0.4) : colors.border)),
          child: Text(label,
              style: TextStyle(
                  color: active ? obAccent : colors.textSecondary,
                  fontSize: 12,
                  fontWeight: active ? FontWeight.w600 : FontWeight.w400)),
        ),
      );
}

// ─── Level chip ───────────────────────────────────────────────────────────────

class ObLevelChip extends StatelessWidget {
  final String level;
  final bool active;
  final VoidCallback onTap;
  final ConsoleColors colors;
  const ObLevelChip({
    super.key,
    required this.level,
    required this.active,
    required this.onTap,
    required this.colors,
  });

  static Color colorFor(String level) => switch (level) {
        'fatal' || 'error'  => obRed,
        'warn'  || 'warning'=> obOrange,
        'debug'             => const Color(0xFF64748B),
        _                   => obAccent,
      };

  @override
  Widget build(BuildContext context) {
    final c = colorFor(level);
    return InkWell(
      onTap: onTap,
      borderRadius: BorderRadius.circular(5),
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
        decoration: BoxDecoration(
            color: active ? c.withValues(alpha: 0.12) : colors.surface,
            borderRadius: BorderRadius.circular(5),
            border: Border.all(
                color: active ? c.withValues(alpha: 0.5) : colors.border)),
        child: Text(level.toUpperCase(),
            style: TextStyle(
                color: active ? c : colors.textSecondary,
                fontSize: 10,
                fontWeight: FontWeight.w700,
                fontFamily: 'monospace')),
      ),
    );
  }
}

// ─── Meta badge ───────────────────────────────────────────────────────────────

class ObMetaBadge extends StatelessWidget {
  final String label;
  final Color color;
  const ObMetaBadge(this.label, this.color, {super.key});

  @override
  Widget build(BuildContext context) => Container(
        padding: const EdgeInsets.symmetric(horizontal: 7, vertical: 2),
        decoration: BoxDecoration(
            color: color.withValues(alpha: 0.08),
            borderRadius: BorderRadius.circular(4)),
        child: Text(label,
            style: TextStyle(
                color: color, fontSize: 11, fontWeight: FontWeight.w500)),
      );
}

// ─── Small action button ──────────────────────────────────────────────────────

class ObActionBtn extends StatelessWidget {
  final String label;
  final Color color;
  final VoidCallback? onTap;
  const ObActionBtn(this.label, this.color, this.onTap, {super.key});

  @override
  Widget build(BuildContext context) => InkWell(
        onTap: onTap,
        borderRadius: BorderRadius.circular(4),
        child: Container(
          padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
          decoration: BoxDecoration(
              border: Border.all(color: color.withValues(alpha: 0.4)),
              borderRadius: BorderRadius.circular(4)),
          child: Text(label,
              style: TextStyle(
                  color: color, fontSize: 11, fontWeight: FontWeight.w500)),
        ),
      );
}

// ─── Dialog dropdown ──────────────────────────────────────────────────────────

/// Thin wrapper that keeps the same call-site API as before.
/// Internally delegates to [AppDropdown] for consistent compact styling.
class ObDialogDropdown extends StatelessWidget {
  final String label, value;
  final List<String> items;
  final String Function(String)? display;
  final void Function(String) onChanged;
  const ObDialogDropdown({
    super.key,
    required this.label,
    required this.value,
    required this.items,
    this.display,
    required this.onChanged,
  });

  @override
  Widget build(BuildContext context) => AppDropdown<String>(
        label: label,
        value: value,
        items: items,
        display: display,
        onChanged: onChanged,
      );
}

// ─── Mini section title ────────────────────────────────────────────────────────

Widget obSectionTitle(String title, ConsoleColors colors, {Widget? trailing}) =>
    Row(children: [
      Text(title,
          style: TextStyle(
              color: colors.textPrimary,
              fontSize: 14,
              fontWeight: FontWeight.w600)),
      if (trailing != null) ...[const SizedBox(width: 8), trailing],
      const Spacer(),
    ]);

// ─── Empty state card ─────────────────────────────────────────────────────────

Widget obEmptyCard(String msg, ConsoleColors colors) => Container(
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
          color: colors.surface,
          borderRadius: BorderRadius.circular(8),
          border: Border.all(color: colors.border)),
      child: Text(msg, style: TextStyle(color: colors.textSecondary, fontSize: 13)),
    );

// ─── Line chart ───────────────────────────────────────────────────────────────

class ObLineChart extends StatelessWidget {
  final List<double> points;
  final Color color;
  const ObLineChart({super.key, required this.points, required this.color});

  @override
  Widget build(BuildContext context) => CustomPaint(
        painter: _LineChartPainter(points: points, color: color),
        child: const SizedBox.expand(),
      );
}

class _LineChartPainter extends CustomPainter {
  final List<double> points;
  final Color color;
  const _LineChartPainter({required this.points, required this.color});

  @override
  void paint(Canvas canvas, Size size) {
    if (points.length < 2) return;
    final max   = points.reduce(math.max);
    final min   = points.reduce(math.min);
    final range = (max - min).clamp(1.0, double.infinity);
    final step  = size.width / (points.length - 1);

    final linePaint = Paint()
      ..color = color
      ..strokeWidth = 1.5
      ..style = PaintingStyle.stroke;
    final fillPaint = Paint()
      ..color = color.withValues(alpha: 0.08)
      ..style = PaintingStyle.fill;

    final path = Path();
    final fill = Path();

    for (var i = 0; i < points.length; i++) {
      final x = i * step;
      final y = size.height - ((points[i] - min) / range) * size.height;
      if (i == 0) {
        path.moveTo(x, y);
        fill.moveTo(x, size.height);
        fill.lineTo(x, y);
      } else {
        path.lineTo(x, y);
        fill.lineTo(x, y);
      }
    }
    fill.lineTo(size.width, size.height);
    fill.close();

    canvas.drawPath(fill, fillPaint);
    canvas.drawPath(path, linePaint);
  }

  @override
  bool shouldRepaint(_LineChartPainter old) => old.points != points;
}

// ─── Toolbar search field ─────────────────────────────────────────────────────

class ObSearchField extends StatelessWidget {
  final TextEditingController controller;
  final String hint;
  final ValueChanged<String> onChanged;
  const ObSearchField({
    super.key,
    required this.controller,
    required this.hint,
    required this.onChanged,
  });

  @override
  Widget build(BuildContext context) {
    final colors = consoleColors(context);
    return SizedBox(
      width: 240,
      height: 34,
      child: TextField(
        controller: controller,
        onChanged: onChanged,
        style: TextStyle(color: colors.textPrimary, fontSize: 13),
        decoration: InputDecoration(
          hintText: hint,
          hintStyle: TextStyle(color: colors.textSubtle, fontSize: 13),
          prefixIcon:
              Icon(LucideIcons.search, size: 14, color: colors.textSubtle),
          contentPadding:
              const EdgeInsets.symmetric(horizontal: 10, vertical: 7),
          filled: true,
          fillColor: colors.surface,
          border: OutlineInputBorder(
              borderRadius: BorderRadius.circular(6),
              borderSide: BorderSide(color: colors.border)),
          enabledBorder: OutlineInputBorder(
              borderRadius: BorderRadius.circular(6),
              borderSide: BorderSide(color: colors.border)),
          focusedBorder: OutlineInputBorder(
              borderRadius: BorderRadius.circular(6),
              borderSide: const BorderSide(color: obAccent)),
        ),
      ),
    );
  }
}

// ─── Context panel ────────────────────────────────────────────────────────────

class ObContextPanel extends StatelessWidget {
  final String title;
  final Map<String, dynamic> data;
  final ConsoleColors colors;
  const ObContextPanel({
    super.key,
    required this.title,
    required this.data,
    required this.colors,
  });

  @override
  Widget build(BuildContext context) {
    if (data.isEmpty) return const SizedBox();
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(title,
            style: TextStyle(
                color: colors.textMuted,
                fontSize: 11,
                fontWeight: FontWeight.w600,
                letterSpacing: 0.6)),
        const SizedBox(height: 8),
        Container(
          width: double.infinity,
          padding: const EdgeInsets.all(12),
          decoration: BoxDecoration(
              color: colors.surface,
              borderRadius: BorderRadius.circular(8),
              border: Border.all(color: colors.border)),
          child: Column(
            children: data.entries.map((e) => Padding(
              padding: const EdgeInsets.only(bottom: 6),
              child: Row(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  SizedBox(
                    width: 120,
                    child: Text(e.key,
                        style: TextStyle(
                            color: colors.textSubtle, fontSize: 12)),
                  ),
                  Expanded(
                    child: SelectableText(
                      '${e.value}',
                      style: TextStyle(
                          color: colors.textPrimary,
                          fontSize: 12,
                          fontFamily: 'monospace'),
                    ),
                  ),
                ],
              ),
            )).toList(),
          ),
        ),
      ],
    );
  }
}

// ─── Add monitor / create rule shared button style ────────────────────────────

ButtonStyle obFilledBtn() => FilledButton.styleFrom(
      backgroundColor: obAccent,
      foregroundColor: Colors.white,
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(8)),
      padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 10),
    );
