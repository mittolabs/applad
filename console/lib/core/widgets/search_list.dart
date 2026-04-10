import 'package:flutter/material.dart';
import '../theme/console_colors.dart';

/// Header row: search field + total count + optional trailing widget.
class SearchListHeader extends StatelessWidget {
  final String searchHint;
  final TextEditingController searchController;
  final int total;
  final int perPage;
  final int currentPage;
  final ValueChanged<int> onPerPageChanged;
  final VoidCallback onPrev;
  final VoidCallback onNext;
  final VoidCallback onSearch;
  final Widget? trailing;

  int get totalPages => (total / perPage).ceil().clamp(1, 999999);

  const SearchListHeader({
    super.key,
    this.searchHint = 'Search by name or ID',
    required this.searchController,
    required this.total,
    required this.perPage,
    required this.currentPage,
    required this.onPerPageChanged,
    required this.onPrev,
    required this.onNext,
    required this.onSearch,
    this.trailing,
  });

  @override
  Widget build(BuildContext context) {
    final colors = consoleColors(context);
    return Row(
      children: [
        // Search field
        SizedBox(
          width: 280,
          child: TextField(
            controller: searchController,
            onSubmitted: (_) => onSearch(),
            style: TextStyle(fontSize: 13, color: colors.textPrimary),
            decoration: InputDecoration(
              hintText: searchHint,
              hintStyle: TextStyle(color: colors.textSubtle, fontSize: 13),
              prefixIcon: Padding(
                padding: EdgeInsets.only(left: 10, right: 6),
                child: Icon(Icons.search, size: 16, color: colors.textSubtle),
              ),
              prefixIconConstraints:
                  const BoxConstraints(minWidth: 32, minHeight: 0),
              filled: true,
              fillColor: colors.fieldFill,
              isDense: true,
              contentPadding:
                  const EdgeInsets.symmetric(vertical: 10, horizontal: 12),
              border: OutlineInputBorder(
                borderRadius: BorderRadius.circular(8),
                borderSide: BorderSide(color: colors.border),
              ),
              enabledBorder: OutlineInputBorder(
                borderRadius: BorderRadius.circular(8),
                borderSide: BorderSide(color: colors.border),
              ),
              focusedBorder: OutlineInputBorder(
                borderRadius: BorderRadius.circular(8),
                borderSide: const BorderSide(color: Color(0xFF3472A4)),
              ),
            ),
          ),
        ),
        const Spacer(),
        if (trailing != null) trailing!,
      ],
    );
  }
}

/// Footer row: per-page selector + Appwrite-style Prev/[N]/Next pagination.
class SearchListFooter extends StatelessWidget {
  final int total;
  final int perPage;
  final int currentPage;
  final VoidCallback onPrev;
  final VoidCallback onNext;
  final ValueChanged<int> onPerPageChanged;
  final String itemLabel;

  const SearchListFooter({
    super.key,
    required this.total,
    required this.perPage,
    required this.currentPage,
    required this.onPrev,
    required this.onNext,
    required this.onPerPageChanged,
    this.itemLabel = 'items',
  });

  int get _totalPages => (total / perPage).ceil().clamp(1, 999999);

  @override
  Widget build(BuildContext context) {
    final canPrev = currentPage > 1;
    final canNext = currentPage < _totalPages;
    final colors = consoleColors(context);

    return Padding(
      padding: const EdgeInsets.only(top: 12),
      child: Row(
        children: [
          // Per-page dropdown
          _PerPageDropdown(value: perPage, onChanged: onPerPageChanged),
          const SizedBox(width: 12),
          Text(
            '$itemLabel per page.  Total: $total',
            style: TextStyle(fontSize: 12, color: colors.textSubtle),
          ),

          const Spacer(),

          // Prev
          _PaginationTextButton(
            label: 'Prev',
            leading: true,
            enabled: canPrev,
            onTap: canPrev ? onPrev : null,
          ),
          const SizedBox(width: 6),

          // Current page badge
          Container(
            width: 28,
            height: 28,
            decoration: BoxDecoration(
              color: colors.fillActive,
              borderRadius: BorderRadius.circular(6),
              border: Border.all(color: colors.border),
            ),
            child: Center(
              child: Text(
                '$currentPage',
                style: TextStyle(
                  color: colors.textPrimary,
                  fontSize: 12,
                  fontWeight: FontWeight.w500,
                ),
              ),
            ),
          ),
          const SizedBox(width: 6),

          // Next
          _PaginationTextButton(
            label: 'Next',
            leading: false,
            enabled: canNext,
            onTap: canNext ? onNext : null,
          ),
        ],
      ),
    );
  }
}

class _PerPageDropdown extends StatelessWidget {
  final int value;
  final ValueChanged<int> onChanged;

  const _PerPageDropdown({required this.value, required this.onChanged});

  @override
  Widget build(BuildContext context) {
    final colors = consoleColors(context);
    return Container(
      height: 28,
      padding: const EdgeInsets.symmetric(horizontal: 8),
      decoration: BoxDecoration(
        color: colors.fieldFill,
        borderRadius: BorderRadius.circular(6),
        border: Border.all(color: colors.border),
      ),
      child: DropdownButtonHideUnderline(
        child: DropdownButton<int>(
          value: value,
          dropdownColor: colors.popupSurface,
          style: TextStyle(fontSize: 12, color: colors.textPrimary),
          icon: Icon(Icons.keyboard_arrow_down,
              size: 16, color: colors.textSubtle),
          items: const [
            DropdownMenuItem(value: 6, child: Text('6')),
            DropdownMenuItem(value: 12, child: Text('12')),
            DropdownMenuItem(value: 25, child: Text('25')),
          ],
          onChanged: (v) {
            if (v != null) onChanged(v);
          },
        ),
      ),
    );
  }
}

class _PaginationTextButton extends StatefulWidget {
  final String label;
  final bool leading;
  final bool enabled;
  final VoidCallback? onTap;

  const _PaginationTextButton({
    required this.label,
    required this.leading,
    required this.enabled,
    this.onTap,
  });

  @override
  State<_PaginationTextButton> createState() =>
      _PaginationTextButtonState();
}

class _PaginationTextButtonState
    extends State<_PaginationTextButton> {
  bool _hovered = false;

  @override
  Widget build(BuildContext context) {
    final colors = consoleColors(context);
    final color = widget.enabled
        ? (_hovered
        ? colors.textSecondary
        : colors.textMuted)
      : colors.textSubtle;

    return MouseRegion(
      cursor: widget.enabled
          ? SystemMouseCursors.click
          : SystemMouseCursors.basic,
      onEnter: (_) => setState(() => _hovered = true),
      onExit: (_) => setState(() => _hovered = false),
      child: GestureDetector(
        onTap: widget.onTap,
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            if (widget.leading)
              Icon(Icons.chevron_left, size: 16, color: color),
            Text(
              widget.label,
              style: TextStyle(fontSize: 12, color: color),
            ),
            if (!widget.leading)
              Icon(Icons.chevron_right, size: 16, color: color),
          ],
        ),
      ),
    );
  }
}
