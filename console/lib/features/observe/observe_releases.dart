import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:lucide_icons/lucide_icons.dart';
import '../../core/theme/console_colors.dart';
import '../../core/utils/url_utils.dart';
import '../../core/widgets/app_data_table.dart';
import '../../core/widgets/app_error_state.dart';
import 'observe_providers.dart';
import 'observe_shared.dart';

class ObReleasesTab extends ConsumerWidget {
  final String projectId;
  const ObReleasesTab({super.key, required this.projectId});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final selectedId = ref.watch(selectedReleaseIdProvider);
    if (selectedId != null) {
      return _ReleaseDetail(
        releaseId: selectedId,
        onBack: () =>
            ref.read(selectedReleaseIdProvider.notifier).state = null,
      );
    }
    return _ReleaseList();
  }
}

// ── Release list ──────────────────────────────────────────────────────────────

class _ReleaseList extends ConsumerStatefulWidget {
  @override
  ConsumerState<_ReleaseList> createState() => _ReleaseListState();
}

class _ReleaseListState extends ConsumerState<_ReleaseList> {
  final _search = TextEditingController();

  @override
  void dispose() {
    _search.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final async = ref.watch(releasesProvider);

    return Padding(
      padding: EdgeInsets.symmetric(horizontal: pageHPad(context)),
      child: async.when(
        loading: () =>
            const Center(child: CircularProgressIndicator(strokeWidth: 2)),
        error: (e, _) => AppErrorState(
            error: e, onRetry: () => ref.invalidate(releasesProvider)),
        data: (data) {
          var releases =
              List<Map<String, dynamic>>.from(data['releases'] ?? []);
          final q = _search.text.trim().toLowerCase();
          if (q.isNotEmpty) {
            releases = releases
                .where((r) =>
                    (r['version'] as String? ?? '').toLowerCase().contains(q) ||
                    (r['environment'] as String? ?? '')
                        .toLowerCase()
                        .contains(q))
                .toList();
          }
          return AppDataTable(
            persistKey: 'ob_releases',
            columns: const [
              AppTableColumn(key: 'version', label: 'Version', flex: 4),
              AppTableColumn(
                  key: 'crashFree', label: 'Crash-free', flex: 2),
              AppTableColumn(
                  key: 'newIssues', label: 'New issues', flex: 2),
              AppTableColumn(
                  key: 'regressed', label: 'Regressed', flex: 2),
              AppTableColumn(key: 'fixed', label: 'Fixed', flex: 2),
              AppTableColumn(key: 'commits', label: 'Commits', flex: 2),
              AppTableColumn(key: 'created', label: 'Created', flex: 2),
            ],
            rows: releases,
            getCellValue: (row, key) => switch (key) {
              'version'   => row['version'] as String? ?? '',
              'crashFree' =>
                '${(row['crashFreeSessionsPct'] ?? 100).toStringAsFixed(2)}%',
              'newIssues' => '${row['newIssues'] ?? 0}',
              'regressed' => '${row['regressedIssues'] ?? 0}',
              'fixed'     => '${row['fixedIssues'] ?? 0}',
              'commits'   => '${row['commitCount'] ?? 0}',
              'created'   => obTimeAgo(row['createdAt']),
              _           => '',
            },
            cellBuilder: (row, key) {
              if (key == 'version') {
                final v   = row['version'] as String? ?? '—';
                final env = row['environment'] as String? ?? '';
                return Row(mainAxisSize: MainAxisSize.min, children: [
                  Icon(LucideIcons.tag,
                      size: 12, color: consoleColors(context).textSubtle),
                  const SizedBox(width: 8),
                  Text(v,
                      style: const TextStyle(
                          color: obAccent,
                          fontSize: 13,
                          fontWeight: FontWeight.w500,
                          fontFamily: 'monospace')),
                  if (env.isNotEmpty) ...[
                    const SizedBox(width: 8),
                    ObMetaBadge(env, obAccent),
                  ],
                ]);
              }
              if (key == 'crashFree') {
                final pct =
                    (row['crashFreeSessionsPct'] ?? 100 as num).toDouble();
                final c = pct >= 99
                    ? obGreen
                    : pct >= 95
                        ? obOrange
                        : obRed;
                return Text('${pct.toStringAsFixed(2)}%',
                    style: TextStyle(
                        color: c,
                        fontSize: 12,
                        fontWeight: FontWeight.w600));
              }
              if (key == 'newIssues') {
                final n = row['newIssues'] ?? 0;
                return Text('$n',
                    style: TextStyle(
                        color: (n as int) > 0
                            ? obOrange
                            : consoleColors(context).textSecondary,
                        fontSize: 12));
              }
              if (key == 'regressed') {
                final n = row['regressedIssues'] ?? 0;
                return Text('$n',
                    style: TextStyle(
                        color: (n as int) > 0
                            ? obRed
                            : consoleColors(context).textSecondary,
                        fontSize: 12));
              }
              if (key == 'fixed') {
                final n = row['fixedIssues'] ?? 0;
                return Text('$n',
                    style: TextStyle(
                        color: (n as int) > 0
                            ? obGreen
                            : consoleColors(context).textSecondary,
                        fontSize: 12));
              }
              return null;
            },
            getRowIcon: (_) => LucideIcons.tag,
            getRowIconColor: (_) => obAccent,
            onRowTap: (row) {
              final id = row[r'$id'] as String? ?? row['version'] as String? ?? '';
              if (id.isNotEmpty) {
                ref.read(selectedReleaseIdProvider.notifier).state = id;
              }
            },
            createLabel: 'Export',
            filters: const [],
            total: releases.length,
            perPage: releases.isEmpty ? 12 : releases.length,
            currentPage: 1,
            onPrev: () {},
            onNext: () {},
            onPerPageChanged: (_) {},
            itemLabel: 'releases',
            searchController: _search,
            onSearch: () => setState(() {}),
            searchHint: 'Search releases…',
            emptyIcon: LucideIcons.tag,
            emptyTitle: 'No releases yet',
            emptySubtitle:
                'Use the SDK to create a release and track its health',
          );
        },
      ),
    );
  }
}

// ── Release detail ────────────────────────────────────────────────────────────

class _ReleaseDetail extends ConsumerWidget {
  final String releaseId;
  final VoidCallback onBack;
  const _ReleaseDetail({required this.releaseId, required this.onBack});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final colors = consoleColors(context);
    final async  = ref.watch(releasesProvider);

    return async.when(
      loading: () =>
          const Center(child: CircularProgressIndicator(color: obAccent)),
      error: (e, _) => AppErrorState(
          error: e, onRetry: () => ref.invalidate(releasesProvider)),
      data: (data) {
        final releases =
            List<Map<String, dynamic>>.from(data['releases'] ?? []);
        final r = releases.firstWhere(
          (rel) =>
              rel[r'$id'] == releaseId || rel['version'] == releaseId,
          orElse: () => {},
        );
        if (r.isEmpty) {
          return Center(
              child: Text('Release not found',
                  style: TextStyle(color: colors.textSecondary)));
        }

        final version = r['version'] as String? ?? '—';
        final commits = List<Map<String, dynamic>>.from(r['commits'] ?? []);
        final issues  = List<Map<String, dynamic>>.from(r['issues'] ?? []);

        return Column(children: [
          // Header
          Container(
            padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 14),
            decoration: BoxDecoration(
                color: colors.background,
                border:
                    Border(bottom: BorderSide(color: colors.border))),
            child: Row(children: [
              InkWell(
                onTap: onBack,
                borderRadius: BorderRadius.circular(6),
                child: Row(mainAxisSize: MainAxisSize.min, children: [
                  Icon(LucideIcons.arrowLeft,
                      size: 14, color: colors.textSecondary),
                  const SizedBox(width: 6),
                  Text('All releases',
                      style: TextStyle(
                          color: colors.textSecondary, fontSize: 13)),
                ]),
              ),
              const SizedBox(width: 20),
              const Icon(LucideIcons.tag, size: 14, color: obAccent),
              const SizedBox(width: 8),
              Text(version,
                  style: TextStyle(
                      color: colors.textPrimary,
                      fontSize: 15,
                      fontWeight: FontWeight.w600,
                      fontFamily: 'monospace')),
            ]),
          ),

          // Body
          Expanded(
            child: SingleChildScrollView(
              padding: EdgeInsets.symmetric(
                  horizontal: pageHPad(context), vertical: 24),
              child: Row(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  // Commits
                  Expanded(
                    flex: 2,
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        obSectionTitle('Commits (${commits.length})', colors),
                        const SizedBox(height: 12),
                        commits.isEmpty
                            ? obEmptyCard('No commits linked', colors)
                            : Column(
                                children: commits
                                    .map((c) =>
                                        _CommitRow(commit: c, colors: colors))
                                    .toList()),
                      ],
                    ),
                  ),
                  const SizedBox(width: 24),

                  // Issues
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        obSectionTitle('Issues', colors),
                        const SizedBox(height: 12),
                        if (issues.isEmpty)
                          obEmptyCard('No issues for this release', colors)
                        else
                          ...issues.map((i) => Padding(
                                padding: const EdgeInsets.only(bottom: 6),
                                child: _IssueRow(issue: i, colors: colors),
                              )),
                      ],
                    ),
                  ),
                ],
              ),
            ),
          ),
        ]);
      },
    );
  }
}

class _CommitRow extends StatelessWidget {
  final Map<String, dynamic> commit;
  final ConsoleColors colors;
  const _CommitRow({required this.commit, required this.colors});

  @override
  Widget build(BuildContext context) {
    final sha     = (commit['sha'] as String? ?? '').take(7);
    final message = commit['message'] as String? ?? '';
    final author  = commit['author'] as String? ?? '';
    return Container(
      margin: const EdgeInsets.only(bottom: 6),
      padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 10),
      decoration: BoxDecoration(
          color: colors.surface,
          borderRadius: BorderRadius.circular(8),
          border: Border.all(color: colors.border)),
      child: Row(children: [
        Text(sha,
            style: const TextStyle(
                color: obAccent,
                fontSize: 11,
                fontFamily: 'monospace',
                fontWeight: FontWeight.w600)),
        const SizedBox(width: 12),
        Expanded(
          child: Text(message,
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
              style: TextStyle(color: colors.textPrimary, fontSize: 12)),
        ),
        const SizedBox(width: 12),
        Text(author,
            style: TextStyle(color: colors.textSubtle, fontSize: 11)),
      ]),
    );
  }
}

class _IssueRow extends StatelessWidget {
  final Map<String, dynamic> issue;
  final ConsoleColors colors;
  const _IssueRow({required this.issue, required this.colors});

  @override
  Widget build(BuildContext context) {
    final title = issue['title'] as String? ?? 'Issue';
    final type  = issue['type'] as String? ?? 'new';
    final color = switch (type) {
      'new'       => obOrange,
      'regressed' => obRed,
      'fixed'     => obGreen,
      _           => colors.textSubtle,
    };
    final icon = switch (type) {
      'new'       => LucideIcons.alertCircle,
      'regressed' => LucideIcons.trendingUp,
      'fixed'     => LucideIcons.checkCircle2,
      _           => LucideIcons.circle,
    };
    return Container(
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
          color: colors.surface,
          borderRadius: BorderRadius.circular(8),
          border: Border.all(color: colors.border)),
      child: Row(children: [
        Icon(icon, size: 13, color: color),
        const SizedBox(width: 8),
        Expanded(
          child: Text(title,
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
              style: TextStyle(color: colors.textPrimary, fontSize: 12)),
        ),
        Text(type,
            style: TextStyle(
                color: color,
                fontSize: 11,
                fontWeight: FontWeight.w500)),
      ]),
    );
  }
}

extension on String {
  String take(int n) => length <= n ? this : substring(0, n);
}
