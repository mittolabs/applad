import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:lucide_icons/lucide_icons.dart';
import '../../core/theme/console_colors.dart';
import '../../core/utils/url_utils.dart';
import '../../core/widgets/app_data_table.dart';
import '../../core/widgets/app_error_state.dart';
import 'observe_providers.dart';
import 'observe_shared.dart';

class ObReplaysTab extends ConsumerStatefulWidget {
  final String projectId;
  const ObReplaysTab({super.key, required this.projectId});
  @override
  ConsumerState<ObReplaysTab> createState() => _ObReplaysTabState();
}

class _ObReplaysTabState extends ConsumerState<ObReplaysTab> {
  String? _selected;
  final _search = TextEditingController();

  @override
  void dispose() {
    _search.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    if (_selected != null) {
      return _ReplayDetail(
        replayId: _selected!,
        onBack: () => setState(() => _selected = null),
      );
    }

    final async = ref.watch(replaysProvider);

    return Padding(
      padding: EdgeInsets.symmetric(horizontal: pageHPad(context)),
      child: async.when(
        loading: () =>
            const Center(child: CircularProgressIndicator(strokeWidth: 2)),
        error: (e, _) => AppErrorState(
            error: e, onRetry: () => ref.invalidate(replaysProvider)),
        data: (data) {
          var replays =
              List<Map<String, dynamic>>.from(data['replays'] ?? []);
          final q = _search.text.trim().toLowerCase();
          if (q.isNotEmpty) {
            replays = replays
                .where((r) =>
                    (r['user'] as String? ?? '').toLowerCase().contains(q) ||
                    (r['url'] as String? ?? '').toLowerCase().contains(q))
                .toList();
          }
          return AppDataTable(
            persistKey: 'ob_replays',
            columns: const [
              AppTableColumn(key: 'user', label: 'User', flex: 3),
              AppTableColumn(key: 'url', label: 'Page', flex: 5),
              AppTableColumn(key: 'duration', label: 'Duration', flex: 2),
              AppTableColumn(key: 'errors', label: 'Errors', flex: 2),
              AppTableColumn(
                  key: 'flags', label: 'Flags', flex: 2, sortable: false),
              AppTableColumn(key: 'browser', label: 'Browser', flex: 2),
              AppTableColumn(key: 'started', label: 'Started', flex: 2),
            ],
            rows: replays,
            getCellValue: (row, key) => switch (key) {
              'user'     => row['user'] as String? ?? 'Anonymous',
              'url'      => row['url'] as String? ?? '',
              'duration' => _fmtDur(row['durationSecs'] ?? 0),
              'errors'   => '${row['errorCount'] ?? 0}',
              'flags'    => _flagsStr(row),
              'browser'  => row['browser'] as String? ?? '',
              'started'  => obTimeAgo(row['startedAt']),
              _          => '',
            },
            cellBuilder: (row, key) {
              if (key == 'errors') {
                final n = row['errorCount'] ?? 0;
                if ((n as int) == 0) return null;
                return ObMetaBadge('$n', obRed);
              }
              if (key == 'flags') {
                final rage = row['hasRageClick'] == true;
                final dead = row['hasDeadClick'] == true;
                if (!rage && !dead) return const SizedBox.shrink();
                return Row(mainAxisSize: MainAxisSize.min, children: [
                  if (rage) ...[
                    const ObMetaBadge('Rage', obOrange),
                    if (dead) const SizedBox(width: 4),
                  ],
                  if (dead)
                    const ObMetaBadge('Dead', Color(0xFF64748B)),
                ]);
              }
              return null;
            },
            getRowIcon: (_) => LucideIcons.play,
            getRowIconColor: (_) => obAccent,
            onRowTap: (row) {
              final id = row[r'$id'] as String? ?? '';
              if (id.isNotEmpty) setState(() => _selected = id);
            },
            createLabel: 'Export',
            filters: const [
              AppTableFilter(
                  key: 'flags',
                  label: 'Flags',
                  options: ['has_errors', 'rage_click']),
            ],
            total: replays.length,
            perPage: replays.isEmpty ? 12 : replays.length,
            currentPage: 1,
            onPrev: () {},
            onNext: () {},
            onPerPageChanged: (_) {},
            itemLabel: 'replays',
            searchController: _search,
            onSearch: () => setState(() {}),
            searchHint: 'Search by user, URL…',
            emptyIcon: LucideIcons.video,
            emptyTitle: 'No session replays',
            emptySubtitle: 'Integrate the SDK to capture user sessions',
          );
        },
      ),
    );
  }

  static String _fmtDur(int secs) {
    if (secs < 60) return '${secs}s';
    return '${secs ~/ 60}m ${secs % 60}s';
  }

  static String _flagsStr(Map<String, dynamic> row) {
    final parts = <String>[];
    if ((row['errorCount'] ?? 0) > 0) parts.add('has_errors');
    if (row['hasRageClick'] == true) parts.add('rage_click');
    return parts.join(',');
  }
}

// ── Replay detail ─────────────────────────────────────────────────────────────

class _ReplayDetail extends ConsumerWidget {
  final String replayId;
  final VoidCallback onBack;
  const _ReplayDetail({required this.replayId, required this.onBack});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final colors = consoleColors(context);
    final async  = ref.watch(replaysProvider);

    return async.when(
      loading: () =>
          const Center(child: CircularProgressIndicator(color: obAccent)),
      error: (e, _) => AppErrorState(
          error: e, onRetry: () => ref.invalidate(replaysProvider)),
      data: (data) {
        final replays =
            List<Map<String, dynamic>>.from(data['replays'] ?? []);
        final r = replays.firstWhere(
          (rep) => rep[r'$id'] == replayId,
          orElse: () => {},
        );
        if (r.isEmpty) {
          return Center(
              child: Text('Replay not found',
                  style: TextStyle(color: colors.textSecondary)));
        }

        final user     = r['user'] as String? ?? 'Anonymous';
        final events   = List<Map<String, dynamic>>.from(r['events'] ?? []);
        final network  = List<Map<String, dynamic>>.from(r['network'] ?? []);
        final console  = List<Map<String, dynamic>>.from(r['console'] ?? []);
        final duration = r['durationSecs'] ?? 0;

        return Column(children: [
          // Header
          Container(
            padding:
                const EdgeInsets.symmetric(horizontal: 24, vertical: 14),
            decoration: BoxDecoration(
                color: colors.background,
                border: Border(
                    bottom: BorderSide(color: colors.border))),
            child: Row(children: [
              InkWell(
                onTap: onBack,
                borderRadius: BorderRadius.circular(6),
                child: Row(mainAxisSize: MainAxisSize.min, children: [
                  Icon(LucideIcons.arrowLeft,
                      size: 14, color: colors.textSecondary),
                  const SizedBox(width: 6),
                  Text('All replays',
                      style: TextStyle(
                          color: colors.textSecondary, fontSize: 13)),
                ]),
              ),
              const SizedBox(width: 20),
              Text('Session — $user',
                  style: TextStyle(
                      color: colors.textPrimary,
                      fontSize: 15,
                      fontWeight: FontWeight.w600)),
              const Spacer(),
              Text(_fmtDur(duration as int),
                  style: TextStyle(
                      color: colors.textSecondary, fontSize: 12)),
            ]),
          ),

          Expanded(
            child: SingleChildScrollView(
              padding: EdgeInsets.symmetric(
                  horizontal: pageHPad(context), vertical: 24),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  // Placeholder player
                  Container(
                    height: 200,
                    decoration: BoxDecoration(
                        color: const Color(0xFF0D0D10),
                        borderRadius: BorderRadius.circular(12),
                        border: Border.all(
                            color: Colors.white.withValues(alpha: 0.06))),
                    child: Center(
                      child: Column(
                          mainAxisSize: MainAxisSize.min,
                          children: [
                            Icon(LucideIcons.play,
                                size: 32,
                                color: Colors.white.withValues(alpha: 0.3)),
                            const SizedBox(height: 8),
                            Text('Replay player',
                                style: TextStyle(
                                    color:
                                        Colors.white.withValues(alpha: 0.3),
                                    fontSize: 13)),
                          ]),
                    ),
                  ),
                  const SizedBox(height: 24),

                  Row(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      // Events timeline
                      Expanded(
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            obSectionTitle(
                                'Events (${events.length})', colors),
                            const SizedBox(height: 12),
                            ...events
                                .take(20)
                                .map((e) => _EventRow(e: e, colors: colors)),
                          ],
                        ),
                      ),
                      const SizedBox(width: 24),

                      // Network requests
                      Expanded(
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            obSectionTitle(
                                'Network (${network.length})', colors),
                            const SizedBox(height: 12),
                            ...network
                                .take(20)
                                .map((n) =>
                                    _NetworkRow(req: n, colors: colors)),
                          ],
                        ),
                      ),
                    ],
                  ),

                  if (console.isNotEmpty) ...[
                    const SizedBox(height: 24),
                    obSectionTitle('Console (${console.length})', colors),
                    const SizedBox(height: 12),
                    Container(
                      padding: const EdgeInsets.all(12),
                      decoration: BoxDecoration(
                          color: const Color(0xFF0D0D10),
                          borderRadius: BorderRadius.circular(8),
                          border: Border.all(
                              color:
                                  Colors.white.withValues(alpha: 0.06))),
                      child: Column(
                        children: console
                            .map((c) => _ConsoleLine(
                                entry: c, colors: colors))
                            .toList(),
                      ),
                    ),
                  ],
                ],
              ),
            ),
          ),
        ]);
      },
    );
  }

  static String _fmtDur(int secs) {
    if (secs < 60) return '${secs}s';
    return '${secs ~/ 60}m ${secs % 60}s';
  }
}

class _EventRow extends StatelessWidget {
  final Map<String, dynamic> e;
  final ConsoleColors colors;
  const _EventRow({required this.e, required this.colors});

  @override
  Widget build(BuildContext context) {
    final type   = e['type'] as String? ?? 'click';
    final target = e['target'] as String? ?? '';
    final ts     = e['offsetMs'] ?? 0;
    final icon   = switch (type) {
      'click'  => LucideIcons.mousePointer2,
      'scroll' => LucideIcons.arrowUpDown,
      'input'  => LucideIcons.keyboard,
      'nav'    => LucideIcons.navigation,
      'error'  => LucideIcons.alertCircle,
      _        => LucideIcons.circle,
    };
    final color = type == 'error' ? obRed : colors.textSubtle;

    return Padding(
      padding: const EdgeInsets.only(bottom: 6),
      child: Row(children: [
        Text('${ts}ms',
            style: const TextStyle(
                color: Color(0xFF64748B),
                fontSize: 10,
                fontFamily: 'monospace')),
        const SizedBox(width: 8),
        Icon(icon, size: 12, color: color),
        const SizedBox(width: 6),
        Expanded(
          child: Text(target.isEmpty ? type : '$type: $target',
              style: TextStyle(
                  color: colors.textSecondary, fontSize: 11),
              overflow: TextOverflow.ellipsis),
        ),
      ]),
    );
  }
}

class _NetworkRow extends StatelessWidget {
  final Map<String, dynamic> req;
  final ConsoleColors colors;
  const _NetworkRow({required this.req, required this.colors});

  @override
  Widget build(BuildContext context) {
    final method = req['method'] as String? ?? 'GET';
    final url    = req['url'] as String? ?? '';
    final status = req['status'] ?? 200;
    final dur    = req['durationMs'] ?? 0;
    final sc     = (status as int) >= 400 ? obRed : obGreen;

    return Padding(
      padding: const EdgeInsets.only(bottom: 6),
      child: Row(children: [
        Text(method,
            style: const TextStyle(
                color: obAccent,
                fontSize: 10,
                fontWeight: FontWeight.w700,
                fontFamily: 'monospace')),
        const SizedBox(width: 6),
        Text('$status',
            style: TextStyle(
                color: sc, fontSize: 10, fontFamily: 'monospace')),
        const SizedBox(width: 8),
        Expanded(
          child: Text(url,
              style: TextStyle(
                  color: colors.textSecondary, fontSize: 11),
              overflow: TextOverflow.ellipsis),
        ),
        Text('${dur}ms',
            style: TextStyle(color: colors.textSubtle, fontSize: 10)),
      ]),
    );
  }
}

class _ConsoleLine extends StatelessWidget {
  final Map<String, dynamic> entry;
  final ConsoleColors colors;
  const _ConsoleLine({required this.entry, required this.colors});

  @override
  Widget build(BuildContext context) {
    final level = entry['level'] as String? ?? 'log';
    final msg   = entry['message'] as String? ?? '';
    final lc    = switch (level) {
      'error' || 'fatal' => obRed,
      'warn'             => obOrange,
      _                  => const Color(0xFF94A3B8),
    };
    return Padding(
      padding: const EdgeInsets.only(bottom: 4),
      child:
          Row(crossAxisAlignment: CrossAxisAlignment.start, children: [
        SizedBox(
          width: 40,
          child: Text(level.toUpperCase(),
              style: TextStyle(
                  color: lc,
                  fontSize: 10,
                  fontFamily: 'monospace',
                  fontWeight: FontWeight.w700)),
        ),
        Expanded(
          child: Text(msg,
              style: const TextStyle(
                  color: Color(0xFFE2E8F0),
                  fontSize: 11,
                  fontFamily: 'monospace')),
        ),
      ]),
    );
  }
}
