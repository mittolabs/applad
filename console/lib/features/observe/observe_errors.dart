import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:lucide_icons/lucide_icons.dart';
import '../../core/api/client.dart';
import '../../core/theme/console_colors.dart';
import '../../core/utils/url_utils.dart';
import '../../core/widgets/app_data_table.dart';
import '../../core/widgets/app_error_state.dart';
import 'observe_providers.dart';
import 'observe_shared.dart';

class ObErrorsTab extends ConsumerStatefulWidget {
  final String projectId;
  const ObErrorsTab({super.key, required this.projectId});
  @override
  ConsumerState<ObErrorsTab> createState() => _ObErrorsTabState();
}

class _ObErrorsTabState extends ConsumerState<ObErrorsTab> {
  final _search = TextEditingController();

  @override
  void dispose() {
    _search.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final selectedId = ref.watch(selectedErrorIdProvider);
    if (selectedId != null) {
      return _ErrorDetailView(
        errorId: selectedId,
        onBack: () => ref.read(selectedErrorIdProvider.notifier).state = null,
      );
    }

    final async = ref.watch(errorsProvider);

    return Padding(
      padding: EdgeInsets.symmetric(horizontal: pageHPad(context)),
      child: async.when(
        loading: () =>
            const Center(child: CircularProgressIndicator(strokeWidth: 2)),
        error: (e, _) => AppErrorState(
            error: e, onRetry: () => ref.invalidate(errorsProvider)),
        data: (data) {
          var errors =
              List<Map<String, dynamic>>.from(data['errors'] ?? []);
          final q = _search.text.trim().toLowerCase();
          if (q.isNotEmpty) {
            errors = errors
                .where((e) =>
                    (e['title'] as String? ?? '').toLowerCase().contains(q))
                .toList();
          }
          return AppDataTable(
            persistKey: 'ob_errors',
            columns: const [
              AppTableColumn(
                  key: 'level', label: 'Level', flex: 2, sortable: false),
              AppTableColumn(key: 'title', label: 'Title', flex: 6),
              AppTableColumn(
                  key: 'status', label: 'Status', flex: 2, sortable: false),
              AppTableColumn(key: 'count', label: 'Events', flex: 2),
              AppTableColumn(key: 'users', label: 'Users', flex: 2),
              AppTableColumn(key: 'lastSeen', label: 'Last seen', flex: 2),
            ],
            rows: errors,
            getCellValue: (row, key) => switch (key) {
              'level'   => row['level'] as String? ?? 'error',
              'title'   => row['title'] as String? ?? '',
              'status'  => row['status'] as String? ?? 'unresolved',
              'count'   => '${row['count'] ?? 0}',
              'users'   => '${row['affectedUsers'] ?? 0}',
              'lastSeen' => obTimeAgo(row['lastSeen']),
              _         => '',
            },
            cellBuilder: (row, key) {
              if (key == 'level') {
                final l = row['level'] as String? ?? 'error';
                final c = ObLevelChip.colorFor(l);
                return Row(mainAxisSize: MainAxisSize.min, children: [
                  Container(
                      width: 7,
                      height: 7,
                      decoration:
                          BoxDecoration(color: c, shape: BoxShape.circle)),
                  const SizedBox(width: 6),
                  Text(l[0].toUpperCase() + l.substring(1),
                      style: TextStyle(
                          color: c,
                          fontSize: 12,
                          fontWeight: FontWeight.w500)),
                ]);
              }
              if (key == 'status') {
                final s = row['status'] as String? ?? 'unresolved';
                final colors = consoleColors(context);
                final c = switch (s) {
                  'resolved' => obGreen,
                  'ignored'  => colors.textSecondary,
                  _          => obOrange,
                };
                return Container(
                  padding:
                      const EdgeInsets.symmetric(horizontal: 7, vertical: 2),
                  decoration: BoxDecoration(
                      color: c.withValues(alpha: 0.1),
                      borderRadius: BorderRadius.circular(4)),
                  child: Text(s[0].toUpperCase() + s.substring(1),
                      style: TextStyle(
                          color: c,
                          fontSize: 11,
                          fontWeight: FontWeight.w500)),
                );
              }
              if (key == 'count') {
                final n = row['count'] ?? 0;
                return Text(obFmtNum(n),
                    style: TextStyle(
                        color: consoleColors(context).textSecondary,
                        fontSize: 12));
              }
              return null;
            },
            getRowIcon: (_) => LucideIcons.alertCircle,
            getRowIconColor: (row) => ObLevelChip.colorFor(
                row['level'] as String? ?? 'error'),
            onRowTap: (row) {
              final id = row[r'$id'] as String? ?? '';
              if (id.isNotEmpty) {
                ref.read(selectedErrorIdProvider.notifier).state = id;
              }
            },
            onDeleteRow: (row) async {
              await ref
                  .read(apiClientProvider)
                  .patch('/observe/errors/${row[r'$id']}/ignore');
              ref.invalidate(errorsProvider);
            },
            createLabel: 'Export',
            filters: const [
              AppTableFilter(
                  key: 'status',
                  label: 'Status',
                  options: ['unresolved', 'resolved', 'ignored']),
              AppTableFilter(
                  key: 'level',
                  label: 'Level',
                  options: ['fatal', 'error', 'warning', 'info']),
            ],
            total: errors.length,
            perPage: errors.isEmpty ? 12 : errors.length,
            currentPage: 1,
            onPrev: () {},
            onNext: () {},
            onPerPageChanged: (_) {},
            itemLabel: 'errors',
            searchController: _search,
            onSearch: () => setState(() {}),
            searchHint: 'Search errors…',
            emptyIcon: LucideIcons.checkCircle2,
            emptyTitle: 'No errors found',
            emptySubtitle: 'Your project is running clean',
          );
        },
      ),
    );
  }
}

// ── Error detail view ─────────────────────────────────────────────────────────

class _ErrorDetailView extends ConsumerWidget {
  final String errorId;
  final VoidCallback onBack;
  const _ErrorDetailView({required this.errorId, required this.onBack});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final colors = consoleColors(context);
    final async  = ref.watch(errorsProvider);

    return async.when(
      loading: () =>
          const Center(child: CircularProgressIndicator(color: obAccent)),
      error: (e, _) => AppErrorState(
          error: e, onRetry: () => ref.invalidate(errorsProvider)),
      data: (data) {
        final errors = List<Map<String, dynamic>>.from(data['errors'] ?? []);
        final err = errors.firstWhere(
          (e) => e[r'$id'] == errorId,
          orElse: () => {},
        );
        if (err.isEmpty) {
          return Center(
              child: Text('Error not found',
                  style: TextStyle(color: colors.textSecondary)));
        }
        return _ErrorDetailBody(
            err: err, onBack: onBack, ref: ref, colors: colors);
      },
    );
  }
}

class _ErrorDetailBody extends StatelessWidget {
  final Map<String, dynamic> err;
  final VoidCallback onBack;
  final WidgetRef ref;
  final ConsoleColors colors;
  const _ErrorDetailBody({
    required this.err,
    required this.onBack,
    required this.ref,
    required this.colors,
  });

  @override
  Widget build(BuildContext context) {
    final title       = err['title'] as String? ?? 'Unknown error';
    final level       = err['level'] as String? ?? 'error';
    final status      = err['status'] as String? ?? 'unresolved';
    final stack       = err['stackTrace'] as String? ?? '';
    final breadcrumbs =
        List<Map<String, dynamic>>.from(err['breadcrumbs'] ?? []);
    final userCtx    = Map<String, dynamic>.from(err['userContext'] ?? {});
    final reqCtx     = Map<String, dynamic>.from(err['requestContext'] ?? {});
    final runtimeCtx = Map<String, dynamic>.from(err['runtimeContext'] ?? {});
    final tags       = Map<String, dynamic>.from(err['tags'] ?? {});
    final activity   = List<Map<String, dynamic>>.from(err['activity'] ?? []);
    final lc         = ObLevelChip.colorFor(level);

    return Column(children: [
      // Header
      Container(
        padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 14),
        decoration: BoxDecoration(
            color: colors.background,
            border: Border(bottom: BorderSide(color: colors.border))),
        child: Row(children: [
          InkWell(
            onTap: onBack,
            borderRadius: BorderRadius.circular(6),
            child: Row(mainAxisSize: MainAxisSize.min, children: [
              Icon(LucideIcons.arrowLeft,
                  size: 14, color: colors.textSecondary),
              const SizedBox(width: 6),
              Text('All errors',
                  style: TextStyle(
                      color: colors.textSecondary, fontSize: 13)),
            ]),
          ),
          const SizedBox(width: 20),
          Container(
              width: 8,
              height: 8,
              decoration: BoxDecoration(color: lc, shape: BoxShape.circle)),
          const SizedBox(width: 10),
          Expanded(
            child: Text(title,
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
                style: TextStyle(
                    color: colors.textPrimary,
                    fontSize: 15,
                    fontWeight: FontWeight.w600)),
          ),
          const SizedBox(width: 16),
          if (status == 'unresolved') ...[
            ObActionBtn('Resolve', obGreen, () async {
              await ref
                  .read(apiClientProvider)
                  .patch('/observe/errors/${err[r'$id']}/resolve');
              ref.invalidate(errorsProvider);
              onBack();
            }),
            const SizedBox(width: 8),
            ObActionBtn('Ignore', colors.textSecondary, () async {
              await ref
                  .read(apiClientProvider)
                  .patch('/observe/errors/${err[r'$id']}/ignore');
              ref.invalidate(errorsProvider);
              onBack();
            }),
          ] else
            ObMetaBadge(
                status[0].toUpperCase() + status.substring(1),
                status == 'resolved' ? obGreen : colors.textSecondary),
        ]),
      ),

      // Body
      Expanded(
        child: SingleChildScrollView(
          padding: const EdgeInsets.all(24),
          child: Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              // Left column — stack + breadcrumbs
              Expanded(
                flex: 3,
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Row(children: [
                      _DetailStat('Events', obFmtNum(err['count'] ?? 0), colors),
                      const SizedBox(width: 24),
                      _DetailStat('Affected users',
                          '${err['affectedUsers'] ?? 0}', colors),
                      const SizedBox(width: 24),
                      _DetailStat('First seen',
                          obTimeAgo(err['firstSeen']), colors),
                      const SizedBox(width: 24),
                      _DetailStat('Last seen',
                          obTimeAgo(err['lastSeen']), colors),
                    ]),
                    const SizedBox(height: 24),

                    if (stack.isNotEmpty) ...[
                      Text('STACK TRACE',
                          style: TextStyle(
                              color: colors.textMuted,
                              fontSize: 11,
                              fontWeight: FontWeight.w600,
                              letterSpacing: 0.6)),
                      const SizedBox(height: 8),
                      Container(
                        width: double.infinity,
                        padding: const EdgeInsets.all(16),
                        decoration: BoxDecoration(
                            color: const Color(0xFF0D0D10),
                            borderRadius: BorderRadius.circular(8),
                            border: Border.all(
                                color: Colors.white.withValues(alpha: 0.08))),
                        child: SelectableText(
                          stack,
                          style: const TextStyle(
                              fontFamily: 'monospace',
                              fontSize: 11,
                              color: Color(0xFFE2E8F0),
                              height: 1.7),
                        ),
                      ),
                      const SizedBox(height: 24),
                    ],

                    if (breadcrumbs.isNotEmpty) ...[
                      Text('BREADCRUMBS',
                          style: TextStyle(
                              color: colors.textMuted,
                              fontSize: 11,
                              fontWeight: FontWeight.w600,
                              letterSpacing: 0.6)),
                      const SizedBox(height: 8),
                      ...breadcrumbs
                          .map((b) => _BreadcrumbRow(crumb: b, colors: colors)),
                      const SizedBox(height: 24),
                    ],
                  ],
                ),
              ),

              const SizedBox(width: 24),

              // Right column — context panels
              SizedBox(
                width: 280,
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    if (tags.isNotEmpty) ...[
                      _TagsPanel(tags: tags, colors: colors),
                      const SizedBox(height: 16),
                    ],
                    ObContextPanel(
                        title: 'USER', data: userCtx, colors: colors),
                    if (userCtx.isNotEmpty) const SizedBox(height: 16),
                    ObContextPanel(
                        title: 'REQUEST', data: reqCtx, colors: colors),
                    if (reqCtx.isNotEmpty) const SizedBox(height: 16),
                    ObContextPanel(
                        title: 'RUNTIME', data: runtimeCtx, colors: colors),
                    if (runtimeCtx.isNotEmpty) const SizedBox(height: 16),

                    if (activity.isNotEmpty) ...[
                      Text('ACTIVITY',
                          style: TextStyle(
                              color: colors.textMuted,
                              fontSize: 11,
                              fontWeight: FontWeight.w600,
                              letterSpacing: 0.6)),
                      const SizedBox(height: 8),
                      ...activity.map(
                          (a) => _ActivityItem(item: a, colors: colors)),
                    ],
                  ],
                ),
              ),
            ],
          ),
        ),
      ),
    ]);
  }
}

class _DetailStat extends StatelessWidget {
  final String label, value;
  final ConsoleColors colors;
  const _DetailStat(this.label, this.value, this.colors);

  @override
  Widget build(BuildContext context) => Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(value,
              style: TextStyle(
                  color: colors.textPrimary,
                  fontSize: 18,
                  fontWeight: FontWeight.w700)),
          Text(label,
              style: TextStyle(color: colors.textSubtle, fontSize: 11)),
        ],
      );
}

class _BreadcrumbRow extends StatelessWidget {
  final Map<String, dynamic> crumb;
  final ConsoleColors colors;
  const _BreadcrumbRow({required this.crumb, required this.colors});

  @override
  Widget build(BuildContext context) {
    final type    = crumb['type'] as String? ?? 'default';
    final message = crumb['message'] as String? ?? '';
    final ts      = crumb['timestamp'] as String? ?? '';
    final color   = switch (type) {
      'error'   => obRed,
      'warning' => obOrange,
      'http'    => obAccent,
      'ui'      => obPurple,
      _         => colors.textSubtle,
    };
    final icon = switch (type) {
      'error'   => LucideIcons.alertCircle,
      'warning' => LucideIcons.alertTriangle,
      'http'    => LucideIcons.globe,
      'ui'      => LucideIcons.mousePointer2,
      'console' => LucideIcons.terminal,
      _         => LucideIcons.circle,
    };

    return Padding(
      padding: const EdgeInsets.only(bottom: 6),
      child: Row(children: [
        Column(children: [
          Icon(icon, size: 13, color: color),
          Container(width: 1, height: 20, color: colors.border),
        ]),
        const SizedBox(width: 10),
        Expanded(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(message,
                  style: TextStyle(
                      color: colors.textPrimary, fontSize: 12)),
              Text(obTimeAgo(ts),
                  style: TextStyle(
                      color: colors.textSubtle, fontSize: 10)),
            ],
          ),
        ),
      ]),
    );
  }
}

class _TagsPanel extends StatelessWidget {
  final Map<String, dynamic> tags;
  final ConsoleColors colors;
  const _TagsPanel({required this.tags, required this.colors});

  @override
  Widget build(BuildContext context) => Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text('TAGS',
              style: TextStyle(
                  color: colors.textMuted,
                  fontSize: 11,
                  fontWeight: FontWeight.w600,
                  letterSpacing: 0.6)),
          const SizedBox(height: 8),
          Wrap(
            spacing: 6,
            runSpacing: 6,
            children: tags.entries
                .map((e) => Container(
                      padding: const EdgeInsets.symmetric(
                          horizontal: 8, vertical: 4),
                      decoration: BoxDecoration(
                          color: colors.surface,
                          borderRadius: BorderRadius.circular(4),
                          border: Border.all(color: colors.border)),
                      child: Text('${e.key}: ${e.value}',
                          style: TextStyle(
                              color: colors.textSecondary,
                              fontSize: 11,
                              fontFamily: 'monospace')),
                    ))
                .toList(),
          ),
        ],
      );
}

class _ActivityItem extends StatelessWidget {
  final Map<String, dynamic> item;
  final ConsoleColors colors;
  const _ActivityItem({required this.item, required this.colors});

  @override
  Widget build(BuildContext context) {
    final type  = item['type'] as String? ?? 'note';
    final user  = item['user'] as String? ?? 'System';
    final text  = item['text'] as String? ?? '';
    final ts    = item['timestamp'] as String? ?? '';
    final icon  = switch (type) {
      'resolved' => LucideIcons.checkCircle2,
      'ignored'  => LucideIcons.bellOff,
      'assigned' => LucideIcons.userCheck,
      _          => LucideIcons.messageCircle,
    };
    final color = switch (type) {
      'resolved' => obGreen,
      'ignored'  => colors.textSubtle,
      _          => obAccent,
    };

    return Padding(
      padding: const EdgeInsets.only(bottom: 10),
      child:
          Row(crossAxisAlignment: CrossAxisAlignment.start, children: [
        Icon(icon, size: 13, color: color),
        const SizedBox(width: 8),
        Expanded(
          child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Row(children: [
                  Text(user,
                      style: TextStyle(
                          color: colors.textPrimary,
                          fontSize: 12,
                          fontWeight: FontWeight.w500)),
                  const SizedBox(width: 6),
                  Text(obTimeAgo(ts),
                      style: TextStyle(
                          color: colors.textSubtle, fontSize: 11)),
                ]),
                if (text.isNotEmpty)
                  Text(text,
                      style: TextStyle(
                          color: colors.textSecondary, fontSize: 12)),
              ]),
        ),
      ]),
    );
  }
}
