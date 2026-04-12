import 'package:flutter/material.dart';
import '../theme/console_colors.dart';

const _kAccent = Color(0xFF3472A4);

/// A compact, consistently-styled dropdown that matches [AppDialogField] height.
///
/// Use wherever a select/dropdown is needed — dialogs, panels, filters.
class AppDropdown<T> extends StatelessWidget {
  final T value;
  final List<T> items;

  /// Optional human-readable label rendered above the dropdown.
  final String? label;

  /// Convert an item to its display string. Defaults to [toString].
  final String Function(T)? display;

  final void Function(T) onChanged;

  const AppDropdown({
    super.key,
    required this.value,
    required this.items,
    this.label,
    this.display,
    required this.onChanged,
  });

  @override
  Widget build(BuildContext context) {
    final colors = consoleColors(context);
    final disp   = display ?? (T v) => v.toString();

    Widget field = DropdownButtonFormField<T>(
      initialValue: value,
      isDense: true,
      dropdownColor: colors.popupSurface,
      icon: Icon(Icons.keyboard_arrow_down_rounded,
          size: 16, color: colors.textMuted),
      style: TextStyle(color: colors.textPrimary, fontSize: 13),
      decoration: InputDecoration(
        filled: true,
        fillColor: colors.fieldFill,
        isDense: true,
        contentPadding:
            const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
        border: OutlineInputBorder(
            borderRadius: BorderRadius.circular(8),
            borderSide: BorderSide(color: colors.fieldBorder)),
        enabledBorder: OutlineInputBorder(
            borderRadius: BorderRadius.circular(8),
            borderSide: BorderSide(color: colors.fieldBorder)),
        focusedBorder: OutlineInputBorder(
            borderRadius: BorderRadius.circular(8),
            borderSide: const BorderSide(color: _kAccent)),
      ),
      items: items
          .map((v) => DropdownMenuItem<T>(
                value: v,
                child: Text(disp(v),
                    style: TextStyle(
                        color: colors.textPrimary, fontSize: 13)),
              ))
          .toList(),
      onChanged: (v) { if (v != null) onChanged(v); },
    );

    if (label == null) return field;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      mainAxisSize: MainAxisSize.min,
      children: [
        Text(label!,
            style: TextStyle(
                color: colors.textMuted,
                fontSize: 12,
                fontWeight: FontWeight.w500)),
        const SizedBox(height: 6),
        field,
      ],
    );
  }
}
