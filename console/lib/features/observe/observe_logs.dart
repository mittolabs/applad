import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:lucide_icons_flutter/lucide_icons.dart';
import '../../core/theme/console_colors.dart';
import '../../core/utils/url_utils.dart';
import '../../core/widgets/app_empty_state.dart';
import '../../core/widgets/app_error_state.dart';
import 'observe_providers.dart';
import 'observe_shared.dart';

// ── Column definitions ────────────────────────────────────────────────────────

const _kAllCols = [
  ('timestamp', 'Time'),
  ('level',     'Level'),
  ('source',    'Source'),
  ('message',   'Message'),
];

// ── Main widget ───────────────────────────────────────────────────────────────

class ObLogsTab extends ConsumerStatefulWidget {
  final String projectId;
  const ObLogsTab({super.key, required this.projectId});
  @override
  ConsumerState<ObLogsTab> createState() => _ObLogsTabState();
}

class _ObLogsTabState extends ConsumerState<ObLogsTab> {
  final _search = TextEditingController();
  bool _live = true;

  // Active filters: level and source (null = any)
  String? _filterLevel;
  String? _filterSource;

  // Column visibility (message always visible)
  final Set<String> _visibleCols = {'timestamp', 'level', 'source', 'message'};

  @override
  void dispose() {
    _search.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final colors  = consoleColors(context);
    final async   = ref.watch(logsProvider);
    final query   = _search.text.toLowerCase();

    final activeFilterCount = [_filterLevel, _filterSource]
        .where((v) => v != null && v.isNotEmpty)
        .length;

    return Column(children: [
      // ── Toolbar ─────────────────────────────────────────────────────────────
      Container(
        padding: EdgeInsets.symmetric(
            horizontal: pageHPad(context), vertical: 10),
        decoration: BoxDecoration(color: colors.background),
        child: Row(children: [
          // Search
          SizedBox(
            width: 280,
            child: ObSearchField(
              controller: _search,
              hint: 'Search logs…',
              onChanged: (_) => setState(() {}),
            ),
          ),
          const Spacer(),

          // Filter popover
          _LogFilterPopover(
            filterLevel:  _filterLevel,
            filterSource: _filterSource,
            activeCount:  activeFilterCount,
            allSources: async.valueOrNull == null
                ? const []
                : List<Map<String, dynamic>>
                      .from(async.valueOrNull!['logs'] ?? [])
                      .map((l) => l['source'] as String? ?? '')
                      .where((s) => s.isNotEmpty)
                      .toSet()
                      .toList()
                  ..sort(),
            onApply: (lvl, src) =>
                setState(() { _filterLevel = lvl; _filterSource = src; }),
            onClear: () =>
                setState(() { _filterLevel = null; _filterSource = null; }),
          ),
          const SizedBox(width: 6),

          // Column picker
          _LogColumnPickerPopover(
            visibleCols: _visibleCols,
            onToggle: (key) {
              if (key == 'message') return; // always visible
              setState(() {
                if (_visibleCols.contains(key)) {
                  if (_visibleCols.length > 1) _visibleCols.remove(key);
                } else {
                  _visibleCols.add(key);
                }
              });
            },
          ),
          const SizedBox(width: 8),

          // Live toggle
          _LiveToggle(
            live: _live,
            onToggle: () => setState(() => _live = !_live),
          ),
          const SizedBox(width: 4),

          // Refresh
          IconButton(
            onPressed: () => ref.invalidate(logsProvider),
            icon: Icon(LucideIcons.refreshCw,
                size: 14, color: colors.textSecondary),
            tooltip: 'Refresh',
          ),
        ]),
      ),

      // ── Log stream ───────────────────────────────────────────────────────────
      Expanded(
        child: async.when(
          loading: () => const Center(
              child: CircularProgressIndicator(color: obAccent)),
          error: (e, _) => AppErrorState(
              error: e,
              onRetry: () => ref.invalidate(logsProvider)),
          data: (data) {
            var logs =
                List<Map<String, dynamic>>.from(data['logs'] ?? []);
            if (_filterLevel != null && _filterLevel!.isNotEmpty) {
              logs = logs.where((l) => l['level'] == _filterLevel).toList();
            }
            if (_filterSource != null && _filterSource!.isNotEmpty) {
              logs = logs.where((l) => l['source'] == _filterSource).toList();
            }
            if (query.isNotEmpty) {
              logs = logs
                  .where((l) =>
                      (l['message'] as String? ?? '')
                          .toLowerCase()
                          .contains(query))
                  .toList();
            }
            if (logs.isEmpty) {
              return const AppEmptyState(
                icon: LucideIcons.terminal,
                title: 'No logs match the current filters',
                subtitle: 'Try adjusting your search or filter settings.',
              );
            }
            return Container(
              color: colors.background,
              child: ListView.builder(
                itemCount: logs.length,
                itemBuilder: (_, i) =>
                    _LogLine(log: logs[i], visibleCols: _visibleCols),
              ),
            );
          },
        ),
      ),
    ]);
  }
}

// ── Live toggle ───────────────────────────────────────────────────────────────

class _LiveToggle extends StatefulWidget {
  final bool live;
  final VoidCallback onToggle;
  const _LiveToggle({required this.live, required this.onToggle});
  @override
  State<_LiveToggle> createState() => _LiveToggleState();
}

class _LiveToggleState extends State<_LiveToggle> {
  bool _hovered = false;
  @override
  Widget build(BuildContext context) {
    final cs = consoleColors(context);
    return MouseRegion(
      cursor: SystemMouseCursors.click,
      onEnter: (_) => setState(() => _hovered = true),
      onExit:  (_) => setState(() => _hovered = false),
      child: GestureDetector(
        onTap: widget.onToggle,
        child: Container(
          height: 32,
          padding: const EdgeInsets.symmetric(horizontal: 10),
          decoration: BoxDecoration(
              color: widget.live
                  ? obGreen.withValues(alpha: 0.1)
                  : _hovered ? cs.fillHover : cs.fieldFill,
              borderRadius: BorderRadius.circular(8),
              border: Border.all(
                  color: widget.live
                      ? obGreen.withValues(alpha: 0.4)
                      : cs.fieldBorder)),
          child: Row(mainAxisSize: MainAxisSize.min, children: [
            Container(
                width: 6,
                height: 6,
                decoration: BoxDecoration(
                    color: widget.live ? obGreen : cs.textSubtle,
                    shape: BoxShape.circle)),
            const SizedBox(width: 6),
            Text(widget.live ? 'Live' : 'Paused',
                style: TextStyle(
                    color: widget.live ? obGreen : cs.textSecondary,
                    fontSize: 12)),
          ]),
        ),
      ),
    );
  }
}

// ── Filter popover ────────────────────────────────────────────────────────────

class _LogFilterPopover extends StatefulWidget {
  final String? filterLevel;
  final String? filterSource;
  final int activeCount;
  final List<String> allSources;
  final void Function(String? level, String? source) onApply;
  final VoidCallback onClear;

  const _LogFilterPopover({
    required this.filterLevel,
    required this.filterSource,
    required this.activeCount,
    required this.allSources,
    required this.onApply,
    required this.onClear,
  });

  @override
  State<_LogFilterPopover> createState() => _LogFilterPopoverState();
}

class _LogFilterPopoverState extends State<_LogFilterPopover> {
  final _link   = LayerLink();
  final _portal = OverlayPortalController();

  bool get _active => widget.activeCount > 0;

  @override
  Widget build(BuildContext context) {
    final cs = consoleColors(context);
    return CompositedTransformTarget(
      link: _link,
      child: OverlayPortal(
        controller: _portal,
        overlayChildBuilder: (_) => Stack(children: [
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
              child: _LogFilterPanel(
                filterLevel:  widget.filterLevel,
                filterSource: widget.filterSource,
                allSources:   widget.allSources,
                onApply: (lvl, src) {
                  widget.onApply(lvl, src);
                  _portal.hide();
                },
                onClear: () {
                  widget.onClear();
                  _portal.hide();
                },
              ),
            ),
          ),
        ]),
        child: _LogToolbarChip(
          active: _active,
          onTap: () => _portal.isShowing ? _portal.hide() : _portal.show(),
          child: Row(mainAxisSize: MainAxisSize.min, children: [
            Icon(LucideIcons.filter,
                size: 13,
                color: _active ? obAccent : cs.textMuted),
            const SizedBox(width: 5),
            Text(
              _active ? 'Filter (${widget.activeCount})' : 'Filter',
              style: TextStyle(
                  color: _active ? obAccent : cs.textMuted,
                  fontSize: 12,
                  fontWeight: FontWeight.w500),
            ),
          ]),
        ),
      ),
    );
  }
}

class _LogFilterPanel extends StatefulWidget {
  final String? filterLevel;
  final String? filterSource;
  final List<String> allSources;
  final void Function(String? level, String? source) onApply;
  final VoidCallback onClear;

  const _LogFilterPanel({
    required this.filterLevel,
    required this.filterSource,
    required this.allSources,
    required this.onApply,
    required this.onClear,
  });

  @override
  State<_LogFilterPanel> createState() => _LogFilterPanelState();
}

class _LogFilterPanelState extends State<_LogFilterPanel> {
  String? _level;
  String? _source;

  @override
  void initState() {
    super.initState();
    _level  = widget.filterLevel;
    _source = widget.filterSource;
  }

  bool get _hasActive =>
      (_level != null && _level!.isNotEmpty) ||
      (_source != null && _source!.isNotEmpty);

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
            Padding(
              padding: const EdgeInsets.fromLTRB(12, 12, 12, 8),
              child: Row(children: [
                Text('Filter',
                    style: TextStyle(
                        color: cs.textMuted,
                        fontSize: 11,
                        fontWeight: FontWeight.w600,
                        letterSpacing: 0.6)),
                const Spacer(),
                if (_hasActive)
                  GestureDetector(
                    onTap: () => setState(
                        () { _level = null; _source = null; }),
                    child: const MouseRegion(
                      cursor: SystemMouseCursors.click,
                      child: Text('Clear all',
                          style: TextStyle(
                              color: obAccent, fontSize: 11)),
                    ),
                  ),
              ]),
            ),

            // Level
            _FilterField(
              label: 'Level',
              value: _level,
              options: const ['debug', 'info', 'warn', 'error', 'fatal'],
              cs: cs,
              onChanged: (v) => setState(() => _level = v),
            ),

            // Source
            if (widget.allSources.isNotEmpty)
              _FilterField(
                label: 'Source',
                value: _source,
                options: widget.allSources,
                cs: cs,
                onChanged: (v) => setState(() => _source = v),
              ),

            Padding(
              padding: const EdgeInsets.all(12),
              child: SizedBox(
                width: double.infinity,
                child: FilledButton(
                  style: FilledButton.styleFrom(
                    backgroundColor: obAccent,
                    padding: const EdgeInsets.symmetric(vertical: 9),
                    shape: RoundedRectangleBorder(
                        borderRadius: BorderRadius.circular(8)),
                  ),
                  onPressed: () => widget.onApply(_level, _source),
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

class _FilterField extends StatelessWidget {
  final String label;
  final String? value;
  final List<String> options;
  final ConsoleColors cs;
  final ValueChanged<String?> onChanged;

  const _FilterField({
    required this.label,
    required this.value,
    required this.options,
    required this.cs,
    required this.onChanged,
  });

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(12, 4, 12, 4),
      child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
        Text(label,
            style: TextStyle(
                color: cs.textSecondary,
                fontSize: 12,
                fontWeight: FontWeight.w500)),
        const SizedBox(height: 4),
        DropdownButtonFormField<String?>(
          initialValue: value,
          dropdownColor: cs.popupSurface,
          style: TextStyle(color: cs.textPrimary, fontSize: 13),
          decoration: InputDecoration(
            filled: true,
            fillColor: cs.fieldFill,
            isDense: true,
            contentPadding:
                const EdgeInsets.symmetric(horizontal: 10, vertical: 8),
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
              borderSide: const BorderSide(color: obAccent),
            ),
          ),
          items: [
            DropdownMenuItem<String?>(
              value: null,
              child: Text('Any', style: TextStyle(color: cs.textSubtle)),
            ),
            ...options.map((o) => DropdownMenuItem(value: o, child: Text(o))),
          ],
          onChanged: onChanged,
        ),
      ]),
    );
  }
}

// ── Column picker popover ─────────────────────────────────────────────────────

class _LogColumnPickerPopover extends StatefulWidget {
  final Set<String> visibleCols;
  final void Function(String) onToggle;

  const _LogColumnPickerPopover({
    required this.visibleCols,
    required this.onToggle,
  });

  @override
  State<_LogColumnPickerPopover> createState() =>
      _LogColumnPickerPopoverState();
}

class _LogColumnPickerPopoverState extends State<_LogColumnPickerPopover> {
  final _link   = LayerLink();
  final _portal = OverlayPortalController();

  @override
  Widget build(BuildContext context) {
    final cs    = consoleColors(context);
    final count = widget.visibleCols.length;

    return CompositedTransformTarget(
      link: _link,
      child: OverlayPortal(
        controller: _portal,
        overlayChildBuilder: (_) => Stack(children: [
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
              child: _LogColumnPanel(
                visibleCols: widget.visibleCols,
                onToggle: widget.onToggle,
              ),
            ),
          ),
        ]),
        child: _LogToolbarChip(
          onTap: () => _portal.isShowing ? _portal.hide() : _portal.show(),
          child: Row(mainAxisSize: MainAxisSize.min, children: [
            _LogVerticalBars(color: cs.textMuted),
            const SizedBox(width: 5),
            Text('$count',
                style: TextStyle(
                    color: cs.textMuted,
                    fontSize: 12,
                    fontWeight: FontWeight.w500)),
          ]),
        ),
      ),
    );
  }
}

class _LogColumnPanel extends StatelessWidget {
  final Set<String> visibleCols;
  final void Function(String) onToggle;

  const _LogColumnPanel(
      {required this.visibleCols, required this.onToggle});

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
            ..._kAllCols.map((col) {
              final (key, label) = col;
              final visible  = visibleCols.contains(key);
              final isMsg    = key == 'message';
              final canToggle = !isMsg && (!visible || visibleCols.length > 1);
              return _LogCheckRow(
                label: label,
                checked: visible,
                enabled: canToggle || isMsg,
                onTap: () { if (!isMsg) onToggle(key); },
              );
            }),
            const SizedBox(height: 6),
          ],
        ),
      ),
    );
  }
}

// ── Shared popover primitives ─────────────────────────────────────────────────

class _LogToolbarChip extends StatefulWidget {
  final Widget child;
  final VoidCallback onTap;
  final bool active;

  const _LogToolbarChip(
      {required this.child, required this.onTap, this.active = false});

  @override
  State<_LogToolbarChip> createState() => _LogToolbarChipState();
}

class _LogToolbarChipState extends State<_LogToolbarChip> {
  bool _hovered = false;

  @override
  Widget build(BuildContext context) {
    final cs = consoleColors(context);
    return MouseRegion(
      cursor: SystemMouseCursors.click,
      onEnter: (_) => setState(() => _hovered = true),
      onExit:  (_) => setState(() => _hovered = false),
      child: GestureDetector(
        onTap: widget.onTap,
        child: Container(
          height: 32,
          padding: const EdgeInsets.symmetric(horizontal: 10),
          decoration: BoxDecoration(
            color: widget.active
                ? obAccent.withValues(alpha: 0.10)
                : _hovered ? cs.fillHover : cs.fieldFill,
            borderRadius: BorderRadius.circular(8),
            border: Border.all(
                color: widget.active
                    ? obAccent.withValues(alpha: 0.35)
                    : cs.fieldBorder),
          ),
          child: Center(child: widget.child),
        ),
      ),
    );
  }
}

class _LogCheckRow extends StatefulWidget {
  final String label;
  final bool checked;
  final bool enabled;
  final VoidCallback onTap;

  const _LogCheckRow({
    required this.label,
    required this.checked,
    required this.enabled,
    required this.onTap,
  });

  @override
  State<_LogCheckRow> createState() => _LogCheckRowState();
}

class _LogCheckRowState extends State<_LogCheckRow> {
  bool _hovered = false;

  @override
  Widget build(BuildContext context) {
    final cs = consoleColors(context);
    return MouseRegion(
      cursor: widget.enabled
          ? SystemMouseCursors.click
          : SystemMouseCursors.basic,
      onEnter: (_) => setState(() => _hovered = true),
      onExit:  (_) => setState(() => _hovered = false),
      child: GestureDetector(
        onTap: widget.enabled ? widget.onTap : null,
        child: Container(
          padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
          color: _hovered && widget.enabled
              ? cs.fillHover
              : Colors.transparent,
          child: Row(children: [
            AnimatedContainer(
              duration: const Duration(milliseconds: 100),
              width: 16,
              height: 16,
              decoration: BoxDecoration(
                color: widget.checked ? obAccent : Colors.transparent,
                borderRadius: BorderRadius.circular(4),
                border: Border.all(
                    color: widget.checked ? obAccent : cs.fieldBorder),
              ),
              child: widget.checked
                  ? const Icon(LucideIcons.check, size: 10, color: Colors.white)
                  : null,
            ),
            const SizedBox(width: 8),
            Text(widget.label,
                style: TextStyle(
                    color: widget.enabled ? cs.textSecondary : cs.textSubtle,
                    fontSize: 13)),
          ]),
        ),
      ),
    );
  }
}

class _LogVerticalBars extends StatelessWidget {
  final Color color;
  const _LogVerticalBars({required this.color});

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
                color: color, borderRadius: BorderRadius.circular(1)),
          ),
        );
      }),
    );
  }
}

// ── Log line ──────────────────────────────────────────────────────────────────

class _LogLine extends StatefulWidget {
  final Map<String, dynamic> log;
  final Set<String> visibleCols;
  const _LogLine({required this.log, required this.visibleCols});
  @override
  State<_LogLine> createState() => _LogLineState();
}

class _LogLineState extends State<_LogLine> {
  bool _expanded = false;

  @override
  Widget build(BuildContext context) {
    final log   = widget.log;
    final level = log['level'] as String? ?? 'info';
    final ts    = log['timestamp'] as String? ?? '';
    final msg   = log['message'] as String? ?? '';
    final src   = log['source'] as String? ?? '';
    final meta  = log['meta'] as Map<String, dynamic>?;
    final vis   = widget.visibleCols;

    final lc = switch (level) {
      'fatal' || 'error' => obRed,
      'warn'             => obOrange,
      'debug'            => const Color(0xFF64748B),
      _                  => const Color(0xFF94A3B8),
    };
    final tsShort = ts.contains('T')
        ? ts.split('T').last.split('.').first
        : ts;

    return InkWell(
      onTap: meta != null
          ? () => setState(() => _expanded = !_expanded)
          : null,
      child: Container(
        decoration: BoxDecoration(
            border: Border(
                bottom: BorderSide(
                    color: Colors.white.withValues(alpha: 0.04)))),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Padding(
              padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 5),
              child: Row(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    if (vis.contains('timestamp'))
                      SizedBox(
                        width: 76,
                        child: Text(tsShort,
                            style: const TextStyle(
                                color: Color(0xFF64748B),
                                fontSize: 11,
                                fontFamily: 'monospace')),
                      ),
                    if (vis.contains('level'))
                      SizedBox(
                        width: 44,
                        child: Text(level.toUpperCase(),
                            style: TextStyle(
                                color: lc,
                                fontSize: 11,
                                fontWeight: FontWeight.w700,
                                fontFamily: 'monospace')),
                      ),
                    if (vis.contains('source') && src.isNotEmpty)
                      Padding(
                        padding: const EdgeInsets.only(right: 10),
                        child: Text(src,
                            style: const TextStyle(
                                color: obAccent,
                                fontSize: 11,
                                fontFamily: 'monospace')),
                      ),
                    Expanded(
                      child: Text(msg,
                          style: const TextStyle(
                              color: Color(0xFFE2E8F0),
                              fontSize: 12,
                              fontFamily: 'monospace')),
                    ),
                    if (meta != null)
                      Icon(
                          _expanded
                              ? LucideIcons.chevronUp
                              : LucideIcons.chevronDown,
                          size: 12,
                          color: const Color(0xFF64748B)),
                  ]),
            ),
            if (_expanded && meta != null)
              Container(
                margin: const EdgeInsets.fromLTRB(136, 0, 16, 6),
                padding: const EdgeInsets.all(10),
                decoration: BoxDecoration(
                    color: Colors.white.withValues(alpha: 0.04),
                    borderRadius: BorderRadius.circular(4)),
                child: SelectableText(
                  meta.entries
                      .map((e) => '${e.key}: ${e.value}')
                      .join('\n'),
                  style: const TextStyle(
                      color: Color(0xFF94A3B8),
                      fontSize: 11,
                      fontFamily: 'monospace',
                      height: 1.5),
                ),
              ),
          ],
        ),
      ),
    );
  }
}
