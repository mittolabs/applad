import 'package:flutter/material.dart';
import 'package:lucide_icons_flutter/lucide_icons.dart';
import '../theme/console_colors.dart';

/// Standard dialog container matching the feedback modal style.
///
/// Usage:
/// ```dart
/// showAppDialog(
///   context: context,
///   title: 'Create project',
///   subtitle: 'Start building something new',
///   content: Column(...),
///   actions: [
///     AppDialogCancel(),
///     AppDialogAction(label: 'Create', onTap: () {}),
///   ],
/// );
/// ```
Future<T?> showAppDialog<T>({
  required BuildContext context,
  required String title,
  String? subtitle,
  required Widget content,
  List<Widget> actions = const [],
  double width = 440,
}) {
  return showDialog<T>(
    context: context,
    barrierColor: Colors.black.withValues(alpha: 0.6),
    builder: (ctx) => Center(
      child: Material(
        color: Colors.transparent,
        child: _AppDialogShell(
          title: title,
          subtitle: subtitle,
          width: width,
          actions: actions,
          child: content,
        ),
      ),
    ),
  );
}

const _accent = Color(0xFF3472A4);

class _AppDialogShell extends StatelessWidget {
  final String title;
  final String? subtitle;
  final double width;
  final Widget child;
  final List<Widget> actions;

  const _AppDialogShell({
    required this.title,
    this.subtitle,
    required this.width,
    required this.child,
    required this.actions,
  });

  @override
  Widget build(BuildContext context) {
    final colors = consoleColors(context);
    return Container(
      width: width,
      constraints: const BoxConstraints(maxHeight: 600),
      decoration: BoxDecoration(
        color: colors.surface,
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: colors.border),
        boxShadow: [
          BoxShadow(
            color: colors.shadow,
            blurRadius: 32,
            offset: const Offset(0, 8),
          ),
        ],
      ),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          // Header
          Padding(
            padding: const EdgeInsets.fromLTRB(20, 20, 20, 0),
            child: Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(title,
                          style: TextStyle(
                            color: colors.textPrimary,
                            fontSize: 16,
                            fontWeight: FontWeight.w600,
                          )),
                      if (subtitle != null) ...[
                        const SizedBox(height: 4),
                        Text(subtitle!,
                            style: TextStyle(
                            color: colors.textMuted,
                                fontSize: 13)),
                      ],
                    ],
                  ),
                ),
                const SizedBox(width: 12),
                GestureDetector(
                  onTap: () => Navigator.of(context).pop(),
                  child: Icon(LucideIcons.x,
                      size: 16, color: colors.textMuted),
                ),
              ],
            ),
          ),
          const SizedBox(height: 16),
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 20),
            child: Container(
              height: 1, color: colors.border),
          ),
          const SizedBox(height: 16),

          // Content
          Flexible(
            child: SingleChildScrollView(
              padding: const EdgeInsets.symmetric(horizontal: 20),
              child: child,
            ),
          ),

          // Actions
          if (actions.isNotEmpty) ...[
            const SizedBox(height: 16),
            Padding(
              padding: const EdgeInsets.fromLTRB(20, 0, 20, 20),
              child: Row(
                mainAxisAlignment: MainAxisAlignment.end,
                children: actions,
              ),
            ),
          ],
        ],
      ),
    );
  }
}

/// Public widget that can be returned directly from a [showDialog] builder.
/// Equivalent to [showAppDialog] but usable as a widget.
class AppDialog extends StatelessWidget {
  final String title;
  final String? subtitle;
  final Widget content;
  final List<Widget> actions;
  final double width;

  const AppDialog({
    super.key,
    required this.title,
    this.subtitle,
    required this.content,
    this.actions = const [],
    this.width = 440,
  });

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Material(
        color: Colors.transparent,
        child: _AppDialogShell(
          title: title,
          subtitle: subtitle,
          width: width,
          actions: actions,
          child: content,
        ),
      ),
    );
  }
}

// ── Shared dialog input field ────────────────────────────────────────────────

class AppDialogField extends StatelessWidget {
  final TextEditingController controller;
  final String? label;
  final String hint;
  final int maxLines;
  final TextInputType? keyboardType;
  final bool autofocus;

  const AppDialogField({
    super.key,
    required this.controller,
    this.label,
    this.hint = '',
    this.maxLines = 1,
    this.keyboardType,
    this.autofocus = false,
  });

  bool get _isMultiLine => maxLines > 1;

  @override
  Widget build(BuildContext context) {
    final colors = consoleColors(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        if (label != null) ...[
          Text(label!,
              style: TextStyle(
                color: colors.textSecondary,
                  fontSize: 12,
                  fontWeight: FontWeight.w500)),
          const SizedBox(height: 6),
        ],
        TextField(
          controller: controller,
          maxLines: _isMultiLine ? null : 1,
          minLines: _isMultiLine ? maxLines : 1,
          keyboardType: keyboardType,
          autofocus: autofocus,
          style: TextStyle(color: colors.textPrimary, fontSize: 13),
          decoration: InputDecoration(
            hintText: hint,
            hintStyle: TextStyle(
                color: colors.textSubtle, fontSize: 13),
            filled: true,
            fillColor: colors.fieldFill,
            isDense: true,
            contentPadding: _isMultiLine
                ? const EdgeInsets.all(12)
                : const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
            border: OutlineInputBorder(
              borderRadius: BorderRadius.circular(8),
              borderSide: BorderSide(color: colors.fieldBorder),
            ),
            enabledBorder: OutlineInputBorder(
              borderRadius: BorderRadius.circular(8),
              borderSide: BorderSide(color: colors.fieldBorder),
            ),
            focusedBorder: OutlineInputBorder(
              borderRadius: BorderRadius.circular(8),
              borderSide: const BorderSide(color: _accent),
            ),
          ),
        ),
      ],
    );
  }
}

// ── Shared dialog select (dropdown) ─────────────────────────────────────────

class AppSelectField<T> extends StatelessWidget {
  final T? value;
  final String? label;
  final String? hint;
  final List<DropdownMenuItem<T>> items;
  final ValueChanged<T?>? onChanged;

  const AppSelectField({
    super.key,
    required this.value,
    this.label,
    this.hint,
    required this.items,
    this.onChanged,
  });

  @override
  Widget build(BuildContext context) {
    final colors = consoleColors(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        if (label != null) ...[
          Text(label!,
              style: TextStyle(
                  color: colors.textSecondary,
                  fontSize: 12,
                  fontWeight: FontWeight.w500)),
          const SizedBox(height: 6),
        ],
        DropdownButtonFormField<T>(
          initialValue: value,
          dropdownColor: colors.surface,
          style: TextStyle(color: colors.textPrimary, fontSize: 13),
          hint: hint != null
              ? Text(hint!, style: TextStyle(color: colors.textSubtle, fontSize: 13))
              : null,
          icon: Icon(Icons.keyboard_arrow_down_rounded,
              size: 18, color: colors.textSubtle),
          decoration: InputDecoration(
            isDense: true,
            contentPadding:
                const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
            filled: true,
            fillColor: colors.fieldFill,
            border: OutlineInputBorder(
              borderRadius: BorderRadius.circular(8),
              borderSide: BorderSide(color: colors.fieldBorder),
            ),
            enabledBorder: OutlineInputBorder(
              borderRadius: BorderRadius.circular(8),
              borderSide: BorderSide(color: colors.fieldBorder),
            ),
            focusedBorder: OutlineInputBorder(
              borderRadius: BorderRadius.circular(8),
              borderSide: const BorderSide(color: _accent),
            ),
          ),
          items: items,
          onChanged: onChanged,
        ),
      ],
    );
  }
}

// ── Shared dialog buttons ────────────────────────────────────────────────────

class AppDialogCancel extends StatelessWidget {
  final String label;
  const AppDialogCancel({super.key, this.label = 'Cancel'});

  @override
  Widget build(BuildContext context) {
    return TextButton(
      onPressed: () => Navigator.of(context).pop(),
      style: TextButton.styleFrom(
        foregroundColor: consoleColors(context).textMuted,
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
      ),
      child: Text(label, style: const TextStyle(fontSize: 13)),
    );
  }
}

class AppDialogAction extends StatelessWidget {
  final String label;
  final VoidCallback? onTap;
  final bool loading;
  final bool destructive;

  const AppDialogAction({
    super.key,
    required this.label,
    this.onTap,
    this.loading = false,
    this.destructive = false,
  });

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.only(left: 8),
      child: FilledButton(
        style: FilledButton.styleFrom(
          backgroundColor:
              destructive ? const Color(0xFFEF4444) : _accent,
          padding:
              const EdgeInsets.symmetric(horizontal: 20, vertical: 10),
          shape: RoundedRectangleBorder(
              borderRadius: BorderRadius.circular(8)),
        ),
        onPressed: loading ? null : onTap,
        child: loading
            ? const SizedBox(
                width: 14,
                height: 14,
                child: CircularProgressIndicator(
                    strokeWidth: 2, color: Colors.white),
              )
            : Text(label, style: const TextStyle(fontSize: 13)),
      ),
    );
  }
}
