import 'package:flutter/material.dart';
import 'package:lucide_icons/lucide_icons.dart';
import '../theme/console_colors.dart';
import 'id_text.dart';
import 'search_list.dart';

// =============================================================================
// Public data types
// =============================================================================

/// Defines a single column in the table.
class AppTableColumn {
  final String key;
  final String label;

  /// Flex weight. Larger = wider column.
  final int flex;

  /// Whether the header is clickable for sorting.
  final bool sortable;

  /// Whether the column is shown by default.
  final bool defaultVisible;

  const AppTableColumn({
    required this.key,
    required this.label,
    this.flex = 1,
    this.sortable = true,
    this.defaultVisible = true,
  });
}

/// Defines a single filter field (shown in the filter popover).
/// [options] are the selectable values. An implicit "Any" choice is prepended.
class AppTableFilter {
  final String key;
  final String label;
  final List<String> options;

  const AppTableFilter({
    required this.key,
    required this.label,
    required this.options,
  });
}

// =============================================================================
// AppDataTable — the main widget
// =============================================================================

const _kAccent = Color(0xFF3472A4);

/// A fully standardized, reusable data table with:
/// - Search
/// - Column visibility toggle
/// - List ↔ Grid view switch
/// - Filter popover
/// - Sortable headers
/// - On-hover row actions
/// - Pagination footer
/// - Empty state
class AppDataTable extends StatefulWidget {
  // ── Data ──────────────────────────────────────────────────────────────────

  final List<AppTableColumn> columns;
  final List<Map<String, dynamic>> rows;

  /// Returns the display string for [row] at [columnKey].
  final String Function(Map<String, dynamic> row, String columnKey)
      getCellValue;

  /// Optional per-row icon (shown left of the first column value).
  final IconData Function(Map<String, dynamic> row)? getRowIcon;

  /// Optional per-row icon colour (defaults to [_kAccent]).
  final Color Function(Map<String, dynamic> row)? getRowIconColor;

  /// Optional per-cell widget override. Return null to use the default cell.
  final Widget? Function(Map<String, dynamic> row, String columnKey)?
      cellBuilder;

  // ── Actions ───────────────────────────────────────────────────────────────

  final void Function(Map<String, dynamic> row)? onRowTap;
  final Future<void> Function(Map<String, dynamic> row)? onDeleteRow;

  // ── Create button ─────────────────────────────────────────────────────────

  final String createLabel;
  final VoidCallback? onCreateTap;

  /// Optional fully custom widget rendered in place of the default create
  /// button. When set, [onCreateTap] and [createLabel] are ignored for the
  /// toolbar button (but [createLabel] is still used in the empty-state).
  final Widget? createWidget;

  // ── Pagination ────────────────────────────────────────────────────────────

  final int total;
  final int perPage;
  final int currentPage;
  final VoidCallback onPrev;
  final VoidCallback onNext;
  final ValueChanged<int> onPerPageChanged;
  final String itemLabel;

  // ── Search ────────────────────────────────────────────────────────────────

  final TextEditingController searchController;
  final VoidCallback onSearch;
  final String searchHint;

  // ── Filters ───────────────────────────────────────────────────────────────

  final List<AppTableFilter> filters;
  final void Function(Map<String, String?>)? onFiltersChanged;

  // ── Grid view ─────────────────────────────────────────────────────────────

  /// If supplied, the grid-view toggle button is shown.
  final Widget Function(Map<String, dynamic> row)? gridCardBuilder;

  // ── Empty state ───────────────────────────────────────────────────────────

  final IconData emptyIcon;
  final String emptyTitle;
  final String emptySubtitle;

  const AppDataTable({
    super.key,
    required this.columns,
    required this.rows,
    required this.getCellValue,
    this.getRowIcon,
    this.getRowIconColor,
    this.cellBuilder,
    this.onRowTap,
    this.onDeleteRow,
    this.createLabel = 'Create',
    this.onCreateTap,
    this.createWidget,
    required this.total,
    required this.perPage,
    required this.currentPage,
    required this.onPrev,
    required this.onNext,
    required this.onPerPageChanged,
    this.itemLabel = 'items',
    required this.searchController,
    required this.onSearch,
    this.searchHint = 'Search by name or ID',
    this.filters = const [],
    this.onFiltersChanged,
    this.gridCardBuilder,
    this.emptyIcon = LucideIcons.layoutList,
    this.emptyTitle = 'No items',
    this.emptySubtitle = 'Create one to get started',
  });

  @override
  State<AppDataTable> createState() => _AppDataTableState();
}

class _AppDataTableState extends State<AppDataTable> {
  bool _isGridView = false;
  late Set<String> _visibleColumns;
  String? _sortKey;
  bool _sortAscending = true;
  final Map<String, String?> _activeFilters = {};

  @override
  void initState() {
    super.initState();
    _visibleColumns = {
      for (final c in widget.columns)
        if (c.defaultVisible) c.key,
    };
  }

  // ── Derived data ────────────────────────────────────────────────────────────

  List<AppTableColumn> get _visibleCols =>
      widget.columns.where((c) => _visibleColumns.contains(c.key)).toList();

  List<Map<String, dynamic>> get _processedRows {
    var rows = List<Map<String, dynamic>>.from(widget.rows);
    // Filter
    for (final f in widget.filters) {
      final val = _activeFilters[f.key];
      if (val != null && val.isNotEmpty) {
        rows = rows
            .where((r) => widget.getCellValue(r, f.key) == val)
            .toList();
      }
    }
    // Sort
    if (_sortKey != null) {
      rows.sort((a, b) {
        final va = widget.getCellValue(a, _sortKey!);
        final vb = widget.getCellValue(b, _sortKey!);
        return _sortAscending ? va.compareTo(vb) : vb.compareTo(va);
      });
    }
    return rows;
  }

  int get _activeFilterCount =>
      _activeFilters.values.where((v) => v != null && v!.isNotEmpty).length;

  // ── Handlers ────────────────────────────────────────────────────────────────

  void _toggleSort(String key) {
    setState(() {
      if (_sortKey == key) {
        if (_sortAscending) {
          _sortAscending = false;
        } else {
          _sortKey = null;
          _sortAscending = true;
        }
      } else {
        _sortKey = key;
        _sortAscending = true;
      }
    });
  }

  void _toggleColumn(String key) {
    setState(() {
      if (_visibleColumns.contains(key)) {
        if (_visibleColumns.length > 1) _visibleColumns.remove(key);
      } else {
        _visibleColumns.add(key);
      }
    });
  }

  void _applyFilters(Map<String, String?> updated) {
    setState(() {
      _activeFilters
        ..clear()
        ..addAll(updated);
    });
    widget.onFiltersChanged?.call(Map.from(_activeFilters));
  }

  // ── Build ───────────────────────────────────────────────────────────────────

  @override
  Widget build(BuildContext context) {
    final rows = _processedRows;

    return Column(
      children: [
        _Toolbar(
          searchController: widget.searchController,
          searchHint: widget.searchHint,
          onSearch: widget.onSearch,
          filters: widget.filters,
          activeFilters: _activeFilters,
          activeFilterCount: _activeFilterCount,
          onFiltersApply: _applyFilters,
          visibleColumns: _visibleColumns,
          columns: widget.columns,
          onToggleColumn: _toggleColumn,
          isGridView: _isGridView,
          hasGridView: widget.gridCardBuilder != null,
          onSetListView: () => setState(() => _isGridView = false),
          onSetGridView: () => setState(() => _isGridView = true),
          createLabel: widget.createLabel,
          onCreateTap: widget.onCreateTap,
          createWidget: widget.createWidget,
        ),
        const SizedBox(height: 16),
        Expanded(
          child: rows.isEmpty
              ? _EmptyState(
                  icon: widget.emptyIcon,
                  title: widget.emptyTitle,
                  subtitle: widget.emptySubtitle,
                  actionLabel: widget.createLabel,
                  onAction: widget.onCreateTap,
                )
              : _isGridView && widget.gridCardBuilder != null
                  ? _GridBody(rows: rows, builder: widget.gridCardBuilder!)
                  : _ListBody(
                      columns: _visibleCols,
                      rows: rows,
                      sortKey: _sortKey,
                      sortAscending: _sortAscending,
                      onSort: _toggleSort,
                      getCellValue: widget.getCellValue,
                      cellBuilder: widget.cellBuilder,
                      getRowIcon: widget.getRowIcon,
                      getRowIconColor: widget.getRowIconColor,
                      onRowTap: widget.onRowTap,
                      onDeleteRow: widget.onDeleteRow,
                    ),
        ),
        SearchListFooter(
          total: widget.total,
          perPage: widget.perPage,
          currentPage: widget.currentPage,
          onPrev: widget.onPrev,
          onNext: widget.onNext,
          onPerPageChanged: widget.onPerPageChanged,
          itemLabel: widget.itemLabel,
        ),
        const SizedBox(height: 8),
      ],
    );
  }
}

// =============================================================================
// Toolbar
// =============================================================================

class _Toolbar extends StatelessWidget {
  final TextEditingController searchController;
  final String searchHint;
  final VoidCallback onSearch;
  final List<AppTableFilter> filters;
  final Map<String, String?> activeFilters;
  final int activeFilterCount;
  final void Function(Map<String, String?>) onFiltersApply;
  final Set<String> visibleColumns;
  final List<AppTableColumn> columns;
  final void Function(String) onToggleColumn;
  final bool isGridView;
  final bool hasGridView;
  final VoidCallback onSetListView;
  final VoidCallback onSetGridView;
  final String createLabel;
  final VoidCallback? onCreateTap;
  final Widget? createWidget;

  const _Toolbar({
    required this.searchController,
    required this.searchHint,
    required this.onSearch,
    required this.filters,
    required this.activeFilters,
    required this.activeFilterCount,
    required this.onFiltersApply,
    required this.visibleColumns,
    required this.columns,
    required this.onToggleColumn,
    required this.isGridView,
    required this.hasGridView,
    required this.onSetListView,
    required this.onSetGridView,
    required this.createLabel,
    required this.onCreateTap,
    this.createWidget,
  });

  @override
  Widget build(BuildContext context) {
    final cs = consoleColors(context);
    return Row(
      children: [
        // ── Search ──────────────────────────────────────────────────────────
        SizedBox(
          width: 280,
          child: TextField(
            controller: searchController,
            onSubmitted: (_) => onSearch(),
            style: TextStyle(fontSize: 13, color: cs.textPrimary),
            decoration: InputDecoration(
              hintText: searchHint,
              hintStyle: TextStyle(color: cs.textSubtle, fontSize: 13),
              prefixIcon: Padding(
                padding: const EdgeInsets.only(left: 10, right: 6),
                child: Icon(LucideIcons.search,
                    size: 15, color: cs.textSubtle),
              ),
              prefixIconConstraints:
                  const BoxConstraints(minWidth: 32, minHeight: 0),
              filled: true,
              fillColor: cs.fieldFill,
              isDense: true,
              contentPadding: const EdgeInsets.symmetric(
                  vertical: 10, horizontal: 12),
              border: OutlineInputBorder(
                  borderRadius: BorderRadius.circular(8),
                  borderSide: BorderSide(color: cs.fieldBorder)),
              enabledBorder: OutlineInputBorder(
                  borderRadius: BorderRadius.circular(8),
                  borderSide: BorderSide(color: cs.fieldBorder)),
              focusedBorder: OutlineInputBorder(
                  borderRadius: BorderRadius.circular(8),
                  borderSide: const BorderSide(color: _kAccent)),
            ),
          ),
        ),

        const Spacer(),

        // ── Filter ──────────────────────────────────────────────────────────
        if (filters.isNotEmpty) ...[
          _FilterPopover(
            filters: filters,
            activeFilters: activeFilters,
            activeCount: activeFilterCount,
            onApply: onFiltersApply,
          ),
          const SizedBox(width: 6),
        ],

        // ── Column picker (list mode only) ───────────────────────────────────
        if (!isGridView) ...[
          _ColumnPickerPopover(
            columns: columns,
            visibleColumns: visibleColumns,
            onToggle: onToggleColumn,
          ),
          const SizedBox(width: 4),
        ],

        // ── View toggle ──────────────────────────────────────────────────────
        if (hasGridView) ...[
          _ViewToggle(
            isGrid: isGridView,
            onList: onSetListView,
            onGrid: onSetGridView,
          ),
          const SizedBox(width: 8),
        ],

        // ── Create ───────────────────────────────────────────────────────────
        if (createWidget != null)
          createWidget!
        else if (onCreateTap != null)
          FilledButton.icon(
            style: FilledButton.styleFrom(
              backgroundColor: _kAccent,
              padding: const EdgeInsets.symmetric(
                  horizontal: 14, vertical: 8),
              shape: RoundedRectangleBorder(
                  borderRadius: BorderRadius.circular(8)),
            ),
            icon: const Icon(LucideIcons.plus, size: 14),
            label: Text(createLabel,
                style: const TextStyle(fontSize: 12)),
            onPressed: onCreateTap,
          ),
      ],
    );
  }
}

// =============================================================================
// List body
// =============================================================================

class _ListBody extends StatelessWidget {
  final List<AppTableColumn> columns;
  final List<Map<String, dynamic>> rows;
  final String? sortKey;
  final bool sortAscending;
  final void Function(String) onSort;
  final String Function(Map<String, dynamic>, String) getCellValue;
  final Widget? Function(Map<String, dynamic>, String)? cellBuilder;
  final IconData Function(Map<String, dynamic>)? getRowIcon;
  final Color Function(Map<String, dynamic>)? getRowIconColor;
  final void Function(Map<String, dynamic>)? onRowTap;
  final Future<void> Function(Map<String, dynamic>)? onDeleteRow;

  const _ListBody({
    required this.columns,
    required this.rows,
    required this.sortKey,
    required this.sortAscending,
    required this.onSort,
    required this.getCellValue,
    this.cellBuilder,
    this.getRowIcon,
    this.getRowIconColor,
    this.onRowTap,
    this.onDeleteRow,
  });

  @override
  Widget build(BuildContext context) {
    final cs = consoleColors(context);
    return Column(
      children: [
        // Header row
        Container(
          padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
          decoration: BoxDecoration(
            border: Border(bottom: BorderSide(color: cs.border)),
          ),
          child: Row(
            children: [
              ...columns.map((col) => Expanded(
                    flex: col.flex,
                    child: col.sortable
                        ? _SortableHeader(
                            label: col.label,
                            isActive: sortKey == col.key,
                            ascending: sortAscending,
                            onTap: () => onSort(col.key),
                          )
                        : Text(col.label,
                            style: TextStyle(
                                color: cs.textMuted,
                                fontSize: 12,
                                fontWeight: FontWeight.w500)),
                  )),
              if (onDeleteRow != null) const SizedBox(width: 40),
            ],
          ),
        ),
        // Data rows
        Expanded(
          child: ListView.builder(
            itemCount: rows.length,
            itemBuilder: (context, i) {
              final row = rows[i];
              return _DataRow(
                row: row,
                columns: columns,
                getCellValue: getCellValue,
                cellBuilder: cellBuilder,
                getRowIcon: getRowIcon,
                getRowIconColor: getRowIconColor,
                onTap: onRowTap != null ? () => onRowTap!(row) : null,
                onDelete:
                    onDeleteRow != null ? () => onDeleteRow!(row) : null,
              );
            },
          ),
        ),
      ],
    );
  }
}

// =============================================================================
// Sortable header cell
// =============================================================================

class _SortableHeader extends StatefulWidget {
  final String label;
  final bool isActive;
  final bool ascending;
  final VoidCallback onTap;

  const _SortableHeader({
    required this.label,
    required this.isActive,
    required this.ascending,
    required this.onTap,
  });

  @override
  State<_SortableHeader> createState() => _SortableHeaderState();
}

class _SortableHeaderState extends State<_SortableHeader> {
  bool _hovered = false;

  @override
  Widget build(BuildContext context) {
    final cs = consoleColors(context);
    final color = (widget.isActive || _hovered)
        ? cs.textSecondary
        : cs.textMuted;

    return MouseRegion(
      cursor: SystemMouseCursors.click,
      onEnter: (_) => setState(() => _hovered = true),
      onExit: (_) => setState(() => _hovered = false),
      child: GestureDetector(
        onTap: widget.onTap,
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            Text(widget.label,
                style: TextStyle(
                    color: color,
                    fontSize: 12,
                    fontWeight: FontWeight.w500)),
            const SizedBox(width: 4),
            _SortChevrons(
              isActive: widget.isActive,
              ascending: widget.ascending,
              hovered: _hovered,
            ),
          ],
        ),
      ),
    );
  }
}

class _SortChevrons extends StatelessWidget {
  final bool isActive;
  final bool ascending;
  final bool hovered;

  const _SortChevrons({
    required this.isActive,
    required this.ascending,
    required this.hovered,
  });

  @override
  Widget build(BuildContext context) {
    final cs = consoleColors(context);
    if (isActive) {
      return Icon(
        ascending ? LucideIcons.chevronUp : LucideIcons.chevronDown,
        size: 12,
        color: _kAccent,
      );
    }
    final dim = hovered ? cs.textSecondary : cs.textSubtle;
    return Column(
      mainAxisSize: MainAxisSize.min,
      children: [
        Icon(LucideIcons.chevronUp, size: 9, color: dim),
        Icon(LucideIcons.chevronDown, size: 9, color: dim),
      ],
    );
  }
}

// =============================================================================
// Data row
// =============================================================================

class _DataRow extends StatefulWidget {
  final Map<String, dynamic> row;
  final List<AppTableColumn> columns;
  final String Function(Map<String, dynamic>, String) getCellValue;
  final Widget? Function(Map<String, dynamic>, String)? cellBuilder;
  final IconData Function(Map<String, dynamic>)? getRowIcon;
  final Color Function(Map<String, dynamic>)? getRowIconColor;
  final VoidCallback? onTap;
  final Future<void> Function()? onDelete;

  const _DataRow({
    required this.row,
    required this.columns,
    required this.getCellValue,
    this.cellBuilder,
    this.getRowIcon,
    this.getRowIconColor,
    this.onTap,
    this.onDelete,
  });

  @override
  State<_DataRow> createState() => _DataRowState();
}

class _DataRowState extends State<_DataRow> {
  bool _hovered = false;

  @override
  Widget build(BuildContext context) {
    final cs = consoleColors(context);
    return MouseRegion(
      onEnter: (_) => setState(() => _hovered = true),
      onExit: (_) => setState(() => _hovered = false),
      cursor: widget.onTap != null
          ? SystemMouseCursors.click
          : SystemMouseCursors.basic,
      child: GestureDetector(
        onTap: widget.onTap,
        child: Container(
          padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
          decoration: BoxDecoration(
            color: _hovered ? cs.fillHover : null,
            border: Border(bottom: BorderSide(color: cs.fill)),
          ),
          child: Row(
            children: [
              ...widget.columns.asMap().entries.map((entry) {
                final index = entry.key;
                final col = entry.value;
                final isFirst = index == 0;

                // Custom cell override (null return = use default)
                final custom = widget.cellBuilder?.call(widget.row, col.key);
                if (custom != null) {
                  return Expanded(flex: col.flex, child: custom);
                }

                final value = widget.getCellValue(widget.row, col.key);
                final isId = col.key == r'$id';

                // First column with icon
                if (isFirst && widget.getRowIcon != null) {
                  final icon = widget.getRowIcon!(widget.row);
                  final iconColor =
                      widget.getRowIconColor?.call(widget.row) ?? _kAccent;
                  return Expanded(
                    flex: col.flex,
                    child: Row(
                      children: [
                        Icon(icon, size: 14, color: iconColor),
                        const SizedBox(width: 8),
                        Expanded(
                          child: isId
                              ? IdText(id: value, fontSize: 12)
                              : Text(
                                  value,
                                  style: TextStyle(
                                      color: cs.textPrimary,
                                      fontSize: 13,
                                      fontFamily: 'monospace'),
                                  overflow: TextOverflow.ellipsis,
                                ),
                        ),
                      ],
                    ),
                  );
                }

                // $id column — always render with expand + copy
                if (isId) {
                  return Expanded(
                    flex: col.flex,
                    child: IdText(id: value, fontSize: 12),
                  );
                }

                // Default cell
                return Expanded(
                  flex: col.flex,
                  child: Text(
                    value,
                    style: TextStyle(
                        color: isFirst ? cs.textPrimary : cs.textMuted,
                        fontSize: 13),
                    overflow: TextOverflow.ellipsis,
                  ),
                );
              }),

              // Delete action (shown on hover)
              if (widget.onDelete != null)
                SizedBox(
                  width: 40,
                  child: _hovered
                      ? MouseRegion(
                          cursor: SystemMouseCursors.click,
                          child: GestureDetector(
                            onTap: widget.onDelete,
                            child: Icon(LucideIcons.trash2,
                                size: 14, color: cs.textSubtle),
                          ),
                        )
                      : const SizedBox(),
                ),
            ],
          ),
        ),
      ),
    );
  }
}

// =============================================================================
// Grid body
// =============================================================================

class _GridBody extends StatelessWidget {
  final List<Map<String, dynamic>> rows;
  final Widget Function(Map<String, dynamic>) builder;

  const _GridBody({required this.rows, required this.builder});

  @override
  Widget build(BuildContext context) {
    return GridView.builder(
      gridDelegate: const SliverGridDelegateWithMaxCrossAxisExtent(
        maxCrossAxisExtent: 300,
        mainAxisSpacing: 12,
        crossAxisSpacing: 12,
        childAspectRatio: 1.5,
      ),
      itemCount: rows.length,
      itemBuilder: (context, i) => builder(rows[i]),
    );
  }
}

// =============================================================================
// Column picker popover
// =============================================================================

class _ColumnPickerPopover extends StatefulWidget {
  final List<AppTableColumn> columns;
  final Set<String> visibleColumns;
  final void Function(String) onToggle;

  const _ColumnPickerPopover({
    required this.columns,
    required this.visibleColumns,
    required this.onToggle,
  });

  @override
  State<_ColumnPickerPopover> createState() => _ColumnPickerPopoverState();
}

class _ColumnPickerPopoverState extends State<_ColumnPickerPopover> {
  final _link = LayerLink();
  final _portal = OverlayPortalController();

  @override
  Widget build(BuildContext context) {
    final cs = consoleColors(context);
    final count = widget.visibleColumns.length;

    return CompositedTransformTarget(
      link: _link,
      child: OverlayPortal(
        controller: _portal,
        overlayChildBuilder: (_) => Stack(
          children: [
            // Dismiss on outside tap
            Positioned.fill(
              child: GestureDetector(
                behavior: HitTestBehavior.translucent,
                onTap: _portal.hide,
              ),
            ),
            Align(
              alignment: Alignment.topLeft,
              child: CompositedTransformFollower(
                link: _link,
                followerAnchor: Alignment.topRight,
                targetAnchor: Alignment.bottomRight,
                offset: const Offset(0, 4),
                child: _ColumnPickerPanel(
                  columns: widget.columns,
                  visibleColumns: widget.visibleColumns,
                  onToggle: (key) {
                    widget.onToggle(key);
                    // The parent rebuilds and passes updated visibleColumns
                  },
                ),
              ),
            ),
          ],
        ),
        child: _ToolbarChip(
          onTap: () =>
              _portal.isShowing ? _portal.hide() : _portal.show(),
          child: Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              // Three vertical bars (mimics the Appwrite columns indicator)
              _VerticalBars(color: cs.textMuted),
              const SizedBox(width: 5),
              Text(
                '$count',
                style: TextStyle(
                    color: cs.textMuted,
                    fontSize: 12,
                    fontWeight: FontWeight.w500),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _ColumnPickerPanel extends StatelessWidget {
  final List<AppTableColumn> columns;
  final Set<String> visibleColumns;
  final void Function(String) onToggle;

  const _ColumnPickerPanel({
    required this.columns,
    required this.visibleColumns,
    required this.onToggle,
  });

  @override
  Widget build(BuildContext context) {
    final cs = consoleColors(context);
    return Material(
      color: Colors.transparent,
      child: Container(
        width: 200,
        decoration: BoxDecoration(
          color: cs.popupSurface,
          borderRadius: BorderRadius.circular(10),
          border: Border.all(color: cs.border),
          boxShadow: [
            BoxShadow(
                color: cs.shadow, blurRadius: 16, offset: const Offset(0, 4)),
          ],
        ),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Padding(
              padding: const EdgeInsets.fromLTRB(12, 12, 12, 6),
              child: Text('Columns',
                  style: TextStyle(
                      color: cs.textMuted,
                      fontSize: 11,
                      fontWeight: FontWeight.w600,
                      letterSpacing: 0.6)),
            ),
            ...columns.map((col) {
              final visible = visibleColumns.contains(col.key);
              final canToggle = !visible || visibleColumns.length > 1;
              return _CheckRow(
                label: col.label,
                checked: visible,
                enabled: canToggle,
                onTap: () {
                  if (canToggle) onToggle(col.key);
                },
              );
            }),
            const SizedBox(height: 6),
          ],
        ),
      ),
    );
  }
}

// =============================================================================
// Filter popover
// =============================================================================

class _FilterPopover extends StatefulWidget {
  final List<AppTableFilter> filters;
  final Map<String, String?> activeFilters;
  final int activeCount;
  final void Function(Map<String, String?>) onApply;

  const _FilterPopover({
    required this.filters,
    required this.activeFilters,
    required this.activeCount,
    required this.onApply,
  });

  @override
  State<_FilterPopover> createState() => _FilterPopoverState();
}

class _FilterPopoverState extends State<_FilterPopover> {
  final _link = LayerLink();
  final _portal = OverlayPortalController();

  bool get _active => widget.activeCount > 0;

  @override
  Widget build(BuildContext context) {
    final cs = consoleColors(context);

    return CompositedTransformTarget(
      link: _link,
      child: OverlayPortal(
        controller: _portal,
        overlayChildBuilder: (_) => Stack(
          children: [
            Positioned.fill(
              child: GestureDetector(
                behavior: HitTestBehavior.translucent,
                onTap: _portal.hide,
              ),
            ),
            Align(
              alignment: Alignment.topLeft,
              child: CompositedTransformFollower(
                link: _link,
                followerAnchor: Alignment.topRight,
                targetAnchor: Alignment.bottomRight,
                offset: const Offset(0, 4),
                child: _FilterPanel(
                  filters: widget.filters,
                  initialFilters: Map.from(widget.activeFilters),
                  onApply: (updated) {
                    widget.onApply(updated);
                    _portal.hide();
                  },
                  onClear: () {
                    widget.onApply({});
                    _portal.hide();
                  },
                ),
              ),
            ),
          ],
        ),
        child: _ToolbarChip(
          active: _active,
          onTap: () =>
              _portal.isShowing ? _portal.hide() : _portal.show(),
          child: Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              Icon(LucideIcons.filter,
                  size: 13,
                  color: _active ? _kAccent : cs.textMuted),
              const SizedBox(width: 5),
              Text(
                _active
                    ? 'Filter (${widget.activeCount})'
                    : 'Filter',
                style: TextStyle(
                    color: _active ? _kAccent : cs.textMuted,
                    fontSize: 12,
                    fontWeight: FontWeight.w500),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _FilterPanel extends StatefulWidget {
  final List<AppTableFilter> filters;
  final Map<String, String?> initialFilters;
  final void Function(Map<String, String?>) onApply;
  final VoidCallback onClear;

  const _FilterPanel({
    required this.filters,
    required this.initialFilters,
    required this.onApply,
    required this.onClear,
  });

  @override
  State<_FilterPanel> createState() => _FilterPanelState();
}

class _FilterPanelState extends State<_FilterPanel> {
  late Map<String, String?> _local;

  @override
  void initState() {
    super.initState();
    _local = Map.from(widget.initialFilters);
  }

  bool get _hasActive =>
      _local.values.any((v) => v != null && v!.isNotEmpty);

  @override
  Widget build(BuildContext context) {
    final cs = consoleColors(context);
    return Material(
      color: Colors.transparent,
      child: Container(
        width: 240,
        decoration: BoxDecoration(
          color: cs.popupSurface,
          borderRadius: BorderRadius.circular(10),
          border: Border.all(color: cs.border),
          boxShadow: [
            BoxShadow(
                color: cs.shadow, blurRadius: 16, offset: const Offset(0, 4)),
          ],
        ),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            // Header
            Padding(
              padding: const EdgeInsets.fromLTRB(12, 12, 12, 8),
              child: Row(
                children: [
                  Text('Filter',
                      style: TextStyle(
                          color: cs.textMuted,
                          fontSize: 11,
                          fontWeight: FontWeight.w600,
                          letterSpacing: 0.6)),
                  const Spacer(),
                  if (_hasActive)
                    GestureDetector(
                      onTap: () {
                        setState(() =>
                            _local = {for (final f in widget.filters) f.key: null});
                      },
                      child: MouseRegion(
                        cursor: SystemMouseCursors.click,
                        child: Text('Clear all',
                            style: TextStyle(
                                color: _kAccent, fontSize: 11)),
                      ),
                    ),
                ],
              ),
            ),
            // Filter fields
            ...widget.filters.map((f) {
              return Padding(
                padding: const EdgeInsets.fromLTRB(12, 4, 12, 4),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(f.label,
                        style: TextStyle(
                            color: cs.textSecondary,
                            fontSize: 12,
                            fontWeight: FontWeight.w500)),
                    const SizedBox(height: 4),
                    DropdownButtonFormField<String?>(
                      value: _local[f.key],
                      dropdownColor: cs.popupSurface,
                      style:
                          TextStyle(color: cs.textPrimary, fontSize: 13),
                      decoration: InputDecoration(
                        filled: true,
                        fillColor: cs.fieldFill,
                        isDense: true,
                        contentPadding: const EdgeInsets.symmetric(
                            horizontal: 10, vertical: 8),
                        border: OutlineInputBorder(
                          borderRadius: BorderRadius.circular(7),
                          borderSide: BorderSide(color: cs.fieldBorder),
                        ),
                        enabledBorder: OutlineInputBorder(
                          borderRadius: BorderRadius.circular(7),
                          borderSide: BorderSide(color: cs.fieldBorder),
                        ),
                        focusedBorder: OutlineInputBorder(
                          borderRadius: BorderRadius.circular(7),
                          borderSide: const BorderSide(color: _kAccent),
                        ),
                      ),
                      items: [
                        DropdownMenuItem<String?>(
                          value: null,
                          child: Text('Any',
                              style:
                                  TextStyle(color: cs.textSubtle)),
                        ),
                        ...f.options.map((o) => DropdownMenuItem(
                              value: o,
                              child: Text(o),
                            )),
                      ],
                      onChanged: (v) =>
                          setState(() => _local[f.key] = v),
                    ),
                  ],
                ),
              );
            }),
            // Apply button
            Padding(
              padding: const EdgeInsets.all(12),
              child: SizedBox(
                width: double.infinity,
                child: FilledButton(
                  style: FilledButton.styleFrom(
                    backgroundColor: _kAccent,
                    padding: const EdgeInsets.symmetric(vertical: 9),
                    shape: RoundedRectangleBorder(
                        borderRadius: BorderRadius.circular(8)),
                  ),
                  onPressed: () => widget.onApply(_local),
                  child: const Text('Apply',
                      style: TextStyle(fontSize: 13)),
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

// =============================================================================
// View toggle (list / grid)
// =============================================================================

class _ViewToggle extends StatelessWidget {
  final bool isGrid;
  final VoidCallback onList;
  final VoidCallback onGrid;

  const _ViewToggle({
    required this.isGrid,
    required this.onList,
    required this.onGrid,
  });

  @override
  Widget build(BuildContext context) {
    final cs = consoleColors(context);
    return Container(
      height: 32,
      decoration: BoxDecoration(
        color: cs.fieldFill,
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: cs.fieldBorder),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          _ToggleBtn(
            icon: LucideIcons.list,
            active: !isGrid,
            isLeft: true,
            onTap: onList,
          ),
          Container(width: 1, color: cs.fieldBorder),
          _ToggleBtn(
            icon: LucideIcons.layoutGrid,
            active: isGrid,
            isLeft: false,
            onTap: onGrid,
          ),
        ],
      ),
    );
  }
}

class _ToggleBtn extends StatefulWidget {
  final IconData icon;
  final bool active;
  final bool isLeft;
  final VoidCallback onTap;

  const _ToggleBtn({
    required this.icon,
    required this.active,
    required this.isLeft,
    required this.onTap,
  });

  @override
  State<_ToggleBtn> createState() => _ToggleBtnState();
}

class _ToggleBtnState extends State<_ToggleBtn> {
  bool _hovered = false;

  @override
  Widget build(BuildContext context) {
    final cs = consoleColors(context);
    final color = widget.active
        ? cs.textPrimary
        : _hovered
            ? cs.textSecondary
            : cs.textSubtle;
    final radius = widget.isLeft
        ? const BorderRadius.horizontal(left: Radius.circular(7))
        : const BorderRadius.horizontal(right: Radius.circular(7));

    return MouseRegion(
      cursor: SystemMouseCursors.click,
      onEnter: (_) => setState(() => _hovered = true),
      onExit: (_) => setState(() => _hovered = false),
      child: GestureDetector(
        onTap: widget.onTap,
        child: Container(
          width: 32,
          height: 32,
          decoration: BoxDecoration(
            color: widget.active
                ? cs.fillActive
                : _hovered
                    ? cs.fillHover
                    : Colors.transparent,
            borderRadius: radius,
          ),
          child: Icon(widget.icon, size: 14, color: color),
        ),
      ),
    );
  }
}

// =============================================================================
// Shared small widgets
// =============================================================================

/// Styled pill button used in the toolbar (filter, column picker).
class _ToolbarChip extends StatefulWidget {
  final Widget child;
  final VoidCallback onTap;
  final bool active;

  const _ToolbarChip({
    required this.child,
    required this.onTap,
    this.active = false,
  });

  @override
  State<_ToolbarChip> createState() => _ToolbarChipState();
}

class _ToolbarChipState extends State<_ToolbarChip> {
  bool _hovered = false;

  @override
  Widget build(BuildContext context) {
    final cs = consoleColors(context);
    return MouseRegion(
      cursor: SystemMouseCursors.click,
      onEnter: (_) => setState(() => _hovered = true),
      onExit: (_) => setState(() => _hovered = false),
      child: GestureDetector(
        onTap: widget.onTap,
        child: Container(
          height: 32,
          padding: const EdgeInsets.symmetric(horizontal: 10),
          decoration: BoxDecoration(
            color: widget.active
                ? _kAccent.withOpacity(0.10)
                : _hovered
                    ? cs.fillHover
                    : cs.fieldFill,
            borderRadius: BorderRadius.circular(8),
            border: Border.all(
                color: widget.active
                    ? _kAccent.withOpacity(0.35)
                    : cs.fieldBorder),
          ),
          child: Center(child: widget.child),
        ),
      ),
    );
  }
}

/// Checkbox row used in the column picker panel.
class _CheckRow extends StatefulWidget {
  final String label;
  final bool checked;
  final bool enabled;
  final VoidCallback onTap;

  const _CheckRow({
    required this.label,
    required this.checked,
    this.enabled = true,
    required this.onTap,
  });

  @override
  State<_CheckRow> createState() => _CheckRowState();
}

class _CheckRowState extends State<_CheckRow> {
  bool _hovered = false;

  @override
  Widget build(BuildContext context) {
    final cs = consoleColors(context);
    return MouseRegion(
      cursor: widget.enabled
          ? SystemMouseCursors.click
          : SystemMouseCursors.basic,
      onEnter: (_) => setState(() => _hovered = true),
      onExit: (_) => setState(() => _hovered = false),
      child: GestureDetector(
        onTap: widget.enabled ? widget.onTap : null,
        child: Container(
          padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
          color: _hovered && widget.enabled ? cs.fillHover : Colors.transparent,
          child: Row(
            children: [
              AnimatedContainer(
                duration: const Duration(milliseconds: 100),
                width: 16,
                height: 16,
                decoration: BoxDecoration(
                  color: widget.checked ? _kAccent : Colors.transparent,
                  borderRadius: BorderRadius.circular(4),
                  border: Border.all(
                    color: widget.checked ? _kAccent : cs.fieldBorder,
                  ),
                ),
                child: widget.checked
                    ? const Icon(LucideIcons.check,
                        size: 10, color: Colors.white)
                    : null,
              ),
              const SizedBox(width: 8),
              Text(widget.label,
                  style: TextStyle(
                      color: widget.enabled
                          ? cs.textSecondary
                          : cs.textSubtle,
                      fontSize: 13)),
            ],
          ),
        ),
      ),
    );
  }
}

/// Three vertical bars icon (matches the Appwrite columns-count indicator).
class _VerticalBars extends StatelessWidget {
  final Color color;
  const _VerticalBars({required this.color});

  @override
  Widget build(BuildContext context) {
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: List.generate(3, (i) {
        final heights = [10.0, 8.0, 11.0];
        return Padding(
          padding: EdgeInsets.only(right: i < 2 ? 2 : 0),
          child: Container(
            width: 2,
            height: heights[i],
            decoration: BoxDecoration(
              color: color,
              borderRadius: BorderRadius.circular(1),
            ),
          ),
        );
      }),
    );
  }
}

// =============================================================================
// Empty state
// =============================================================================

class AppTableEmptyState extends StatelessWidget {
  final IconData icon;
  final String title;
  final String subtitle;
  final String actionLabel;
  final VoidCallback? onAction;

  const AppTableEmptyState({
    super.key,
    required this.icon,
    required this.title,
    required this.subtitle,
    required this.actionLabel,
    this.onAction,
  });

  @override
  Widget build(BuildContext context) => _EmptyState(
        icon: icon,
        title: title,
        subtitle: subtitle,
        actionLabel: actionLabel,
        onAction: onAction,
      );
}

class _EmptyState extends StatelessWidget {
  final IconData icon;
  final String title;
  final String subtitle;
  final String actionLabel;
  final VoidCallback? onAction;

  const _EmptyState({
    required this.icon,
    required this.title,
    required this.subtitle,
    required this.actionLabel,
    this.onAction,
  });

  @override
  Widget build(BuildContext context) {
    final cs = consoleColors(context);
    return Center(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Container(
            width: 48,
            height: 48,
            decoration: BoxDecoration(
              color: cs.fill,
              borderRadius: BorderRadius.circular(12),
            ),
            child: Icon(icon, size: 22, color: cs.textSubtle),
          ),
          const SizedBox(height: 16),
          Text(title,
              style: TextStyle(
                  color: cs.textPrimary,
                  fontSize: 15,
                  fontWeight: FontWeight.w500)),
          const SizedBox(height: 6),
          Text(subtitle,
              style: TextStyle(color: cs.textMuted, fontSize: 13)),
          if (onAction != null) ...[
            const SizedBox(height: 16),
            FilledButton(
              style: FilledButton.styleFrom(
                backgroundColor: _kAccent,
                padding: const EdgeInsets.symmetric(
                    horizontal: 20, vertical: 10),
                shape: RoundedRectangleBorder(
                    borderRadius: BorderRadius.circular(8)),
              ),
              onPressed: onAction,
              child:
                  Text(actionLabel, style: const TextStyle(fontSize: 13)),
            ),
          ],
        ],
      ),
    );
  }
}
