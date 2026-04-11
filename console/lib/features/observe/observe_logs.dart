import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:lucide_icons/lucide_icons.dart';
import '../../core/theme/console_colors.dart';
import '../../core/widgets/app_error_state.dart';
import 'observe_providers.dart';
import 'observe_shared.dart';

class ObLogsTab extends ConsumerStatefulWidget {
  final String projectId;
  const ObLogsTab({super.key, required this.projectId});
  @override
  ConsumerState<ObLogsTab> createState() => _ObLogsTabState();
}

class _ObLogsTabState extends ConsumerState<ObLogsTab> {
  final _search = TextEditingController();
  bool _live = true;

  @override
  void dispose() {
    _search.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final colors       = consoleColors(context);
    final async        = ref.watch(logsProvider);
    final levelFilter  = ref.watch(logLevelFilterProvider);
    final sourceFilter = ref.watch(logSourceFilterProvider);
    final query        = _search.text.toLowerCase();

    return Column(children: [
      // Toolbar
      Container(
        padding:
            const EdgeInsets.symmetric(horizontal: 24, vertical: 10),
        decoration: BoxDecoration(
            color: colors.background,
            border: Border(bottom: BorderSide(color: colors.border))),
        child: Row(children: [
          ObSearchField(
            controller: _search,
            hint: 'Search logs…',
            onChanged: (_) => setState(() {}),
          ),
          const SizedBox(width: 10),
          for (final l in ['debug', 'info', 'warn', 'error', 'fatal'])
            Padding(
              padding: const EdgeInsets.only(right: 5),
              child: ObLevelChip(
                level: l,
                active: levelFilter == l,
                onTap: () =>
                    ref.read(logLevelFilterProvider.notifier).state =
                        levelFilter == l ? '' : l,
                colors: colors,
              ),
            ),
          const Spacer(),
          // Live toggle
          InkWell(
            onTap: () => setState(() => _live = !_live),
            borderRadius: BorderRadius.circular(6),
            child: Container(
              padding: const EdgeInsets.symmetric(
                  horizontal: 10, vertical: 5),
              decoration: BoxDecoration(
                  color:
                      _live ? obGreen.withValues(alpha: 0.1) : colors.surface,
                  borderRadius: BorderRadius.circular(6),
                  border: Border.all(
                      color: _live
                          ? obGreen.withValues(alpha: 0.4)
                          : colors.border)),
              child: Row(mainAxisSize: MainAxisSize.min, children: [
                Container(
                    width: 6,
                    height: 6,
                    decoration: BoxDecoration(
                        color: _live ? obGreen : colors.textSubtle,
                        shape: BoxShape.circle)),
                const SizedBox(width: 6),
                Text(_live ? 'Live' : 'Paused',
                    style: TextStyle(
                        color:
                            _live ? obGreen : colors.textSecondary,
                        fontSize: 12)),
              ]),
            ),
          ),
          const SizedBox(width: 8),
          IconButton(
            onPressed: () => ref.invalidate(logsProvider),
            icon: Icon(LucideIcons.refreshCw,
                size: 14, color: colors.textSecondary),
            tooltip: 'Refresh',
          ),
        ]),
      ),

      // Log stream
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
            if (levelFilter.isNotEmpty) {
              logs = logs
                  .where((l) => l['level'] == levelFilter)
                  .toList();
            }
            if (sourceFilter.isNotEmpty) {
              logs = logs
                  .where((l) => l['source'] == sourceFilter)
                  .toList();
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
              return Center(
                child: Text('No logs match the current filters',
                    style: TextStyle(
                        color: colors.textSecondary,
                        fontSize: 14)),
              );
            }
            return Container(
              color: const Color(0xFF0D0D10),
              child: ListView.builder(
                itemCount: logs.length,
                itemBuilder: (_, i) => _LogLine(log: logs[i]),
              ),
            );
          },
        ),
      ),
    ]);
  }
}

// ── Log line ──────────────────────────────────────────────────────────────────

class _LogLine extends StatefulWidget {
  final Map<String, dynamic> log;
  const _LogLine({required this.log});
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
    final lc    = switch (level) {
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
                bottom:
                    BorderSide(color: Colors.white.withValues(alpha: 0.04)))),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Padding(
              padding: const EdgeInsets.symmetric(
                  horizontal: 16, vertical: 5),
              child: Row(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    SizedBox(
                      width: 76,
                      child: Text(tsShort,
                          style: const TextStyle(
                              color: Color(0xFF64748B),
                              fontSize: 11,
                              fontFamily: 'monospace')),
                    ),
                    SizedBox(
                      width: 44,
                      child: Text(level.toUpperCase(),
                          style: TextStyle(
                              color: lc,
                              fontSize: 11,
                              fontWeight: FontWeight.w700,
                              fontFamily: 'monospace')),
                    ),
                    if (src.isNotEmpty)
                      Padding(
                        padding: const EdgeInsets.only(right: 10),
                        child: Text(src,
                            style: const TextStyle(
                                color: Color(0xFF6C47FF),
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
