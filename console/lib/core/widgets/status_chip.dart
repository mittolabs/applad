import 'package:flutter/material.dart';
import '../theme/console_colors.dart';

enum StatusVariant { success, warning, danger, info, neutral }

/// A semantic status pill with a dot indicator.
///
/// Usage — auto-mapped from common status strings:
///   StatusChip(status: 'verified')
///   StatusChip(status: 'unverified')
///   StatusChip(status: 'disabled')
///
/// Or with explicit variant:
///   StatusChip(label: 'Custom', variant: StatusVariant.warning)
class StatusChip extends StatelessWidget {
  final String label;
  final StatusVariant variant;

  const StatusChip({super.key, required this.label, required this.variant});

  /// Resolves a raw status string to a [StatusChip] automatically.
  factory StatusChip.fromStatus(String status) {
    final s = status.toLowerCase().trim();
    StatusVariant v;
    String l = _capitalize(status);

    switch (s) {
      // Success — green
      case 'verified':
      case 'active':
      case 'completed':
      case 'success':
      case 'deployed':
      case 'published':
      case 'enabled':
        v = StatusVariant.success;
        break;

      // Warning — amber
      case 'unverified':
      case 'pending':
      case 'draft':
      case 'paused':
      case 'idle':
      case 'scheduled':
        v = StatusVariant.warning;
        break;

      // Danger — red
      case 'disabled':
      case 'failed':
      case 'error':
      case 'suspended':
      case 'banned':
      case 'inactive':
      case 'deleted':
        v = StatusVariant.danger;
        break;

      // Info — blue
      case 'running':
      case 'processing':
      case 'building':
      case 'deploying':
      case 'queued':
        v = StatusVariant.info;
        break;

      // Neutral — gray
      default:
        v = StatusVariant.neutral;
    }
    return StatusChip(label: l, variant: v);
  }

  static String _capitalize(String s) =>
      s.isEmpty ? s : s[0].toUpperCase() + s.substring(1);

  @override
  Widget build(BuildContext context) {
    final isLight = consoleIsLight(context);
    final colors = _variantColors(variant, isLight);

    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
      decoration: BoxDecoration(
        color: colors.bg,
        borderRadius: BorderRadius.circular(4),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Container(
            width: 5,
            height: 5,
            decoration: BoxDecoration(color: colors.dot, shape: BoxShape.circle),
          ),
          const SizedBox(width: 5),
          Text(
            label,
            style: TextStyle(
              color: colors.text,
              fontSize: 11,
              fontWeight: FontWeight.w500,
            ),
          ),
        ],
      ),
        ),
      ],
    );
  }
}

class _ChipColors {
  final Color bg;
  final Color dot;
  final Color text;
  const _ChipColors({required this.bg, required this.dot, required this.text});
}

_ChipColors _variantColors(StatusVariant v, bool isLight) {
  switch (v) {
    case StatusVariant.success:
      return isLight
          ? _ChipColors(
              bg: const Color(0xFF059669).withValues(alpha: 0.10),
              dot: const Color(0xFF059669),
              text: const Color(0xFF059669))
          : _ChipColors(
              bg: const Color(0xFF34D399).withValues(alpha: 0.12),
              dot: const Color(0xFF34D399),
              text: const Color(0xFF34D399));

    case StatusVariant.warning:
      return isLight
          ? _ChipColors(
              bg: const Color(0xFFD97706).withValues(alpha: 0.10),
              dot: const Color(0xFFD97706),
              text: const Color(0xFFD97706))
          : _ChipColors(
              bg: const Color(0xFFFBBF24).withValues(alpha: 0.12),
              dot: const Color(0xFFFBBF24),
              text: const Color(0xFFFBBF24));

    case StatusVariant.danger:
      return isLight
          ? _ChipColors(
              bg: const Color(0xFFDC2626).withValues(alpha: 0.10),
              dot: const Color(0xFFDC2626),
              text: const Color(0xFFDC2626))
          : _ChipColors(
              bg: const Color(0xFFF87171).withValues(alpha: 0.12),
              dot: const Color(0xFFF87171),
              text: const Color(0xFFF87171));

    case StatusVariant.info:
      return isLight
          ? _ChipColors(
              bg: const Color(0xFF2563EB).withValues(alpha: 0.10),
              dot: const Color(0xFF2563EB),
              text: const Color(0xFF2563EB))
          : _ChipColors(
              bg: const Color(0xFF60A5FA).withValues(alpha: 0.12),
              dot: const Color(0xFF60A5FA),
              text: const Color(0xFF60A5FA));

    case StatusVariant.neutral:
      return isLight
          ? _ChipColors(
              bg: Colors.black.withValues(alpha: 0.06),
              dot: const Color(0xFF6B7280),
              text: const Color(0xFF6B7280))
          : _ChipColors(
              bg: Colors.white.withValues(alpha: 0.06),
              dot: const Color(0xFF9CA3AF),
              text: const Color(0xFF9CA3AF));
  }
}
