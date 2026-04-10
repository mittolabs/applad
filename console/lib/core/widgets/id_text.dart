import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:lucide_icons/lucide_icons.dart';
import '../theme/console_colors.dart';

/// Displays an ID string truncated with ellipsis.
///
/// - Tap the text to toggle between truncated and full display.
/// - Tap the copy icon to copy the full ID to the clipboard.
///
/// Usage:
/// ```dart
/// IdText(id: row['\$id'])
/// IdText(id: projectId, previewLength: 8)
/// ```
class IdText extends StatefulWidget {
  final String id;

  /// Number of characters shown before "…" in collapsed state. Defaults to 12.
  final int previewLength;

  /// Font size. Defaults to 13.
  final double fontSize;

  const IdText({
    super.key,
    required this.id,
    this.previewLength = 12,
    this.fontSize = 13,
  });

  @override
  State<IdText> createState() => _IdTextState();
}

class _IdTextState extends State<IdText> {
  bool _expanded = false;
  bool _copied = false;

  Future<void> _copy() async {
    await Clipboard.setData(ClipboardData(text: widget.id));
    if (!mounted) return;
    setState(() => _copied = true);
    await Future.delayed(const Duration(seconds: 2));
    if (mounted) setState(() => _copied = false);
  }

  @override
  Widget build(BuildContext context) {
    final cs = consoleColors(context);
    final truncated = widget.id.length > widget.previewLength;
    final display = _expanded || !truncated
        ? widget.id
        : '${widget.id.substring(0, widget.previewLength)}\u2026';

    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        // Expandable text
        if (truncated)
          GestureDetector(
            onTap: () => setState(() => _expanded = !_expanded),
            child: MouseRegion(
              cursor: SystemMouseCursors.click,
              child: Text(
                display,
                style: TextStyle(
                  color: cs.textSubtle,
                  fontSize: widget.fontSize,
                  fontFamily: 'monospace',
                ),
              ),
            ),
          )
        else
          Text(
            display,
            style: TextStyle(
              color: cs.textSubtle,
              fontSize: widget.fontSize,
              fontFamily: 'monospace',
            ),
          ),
        const SizedBox(width: 5),
        // Copy button
        GestureDetector(
          onTap: _copy,
          child: MouseRegion(
            cursor: SystemMouseCursors.click,
            child: AnimatedSwitcher(
              duration: const Duration(milliseconds: 150),
              child: Icon(
                _copied ? LucideIcons.check : LucideIcons.copy,
                key: ValueKey(_copied),
                size: 11,
                color: _copied
                    ? const Color(0xFF4CAF50)
                    : cs.textSubtle,
              ),
            ),
          ),
        ),
      ],
    );
  }
}
