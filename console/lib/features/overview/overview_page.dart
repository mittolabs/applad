import 'dart:math' as math;
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:lucide_icons/lucide_icons.dart';
import '../../core/api/client.dart';
import '../../core/providers/project_provider.dart';
import '../../core/theme/console_colors.dart';
import '../../core/utils/url_utils.dart';
import '../../core/widgets/id_text.dart';
import '../../core/widgets/page_tabs.dart';
import '../../core/widgets/app_error_state.dart';

// --- Constants ---------------------------------------------------------------

const _accent = Color(0xFF3472A4);

// --- Providers ---------------------------------------------------------------

final _projectStatsProvider =
    FutureProvider.family<Map<String, dynamic>, String>((ref, projectId) async {
  final api = ref.read(apiClientProvider);

  Future<dynamic> safeFetch(String path) async {
    try {
      final res = await api.get(path);
      return res.data;
    } catch (_) {
      return null;
    }
  }

  final futures = await Future.wait([
    safeFetch('/databases'),
    safeFetch('/functions'),
    safeFetch('/storage/buckets'),
    safeFetch('/account/users'),
    safeFetch('/deploy/targets'),
    safeFetch('/workflows'),
    safeFetch('/projects/$projectId/usage'),
    safeFetch('/projects/$projectId/platforms'),
    safeFetch('/projects/$projectId/keys'),
    safeFetch('/deploy/releases?limit=5'),
  ]);

  int count(dynamic data, String key) {
    if (data == null) return 0;
    if (data is Map && data[key] is List) return (data[key] as List).length;
    if (data is Map && data['total'] is int) return data['total'] as int;
    return 0;
  }

  return {
    'databases': count(futures[0], 'databases'),
    'functions': count(futures[1], 'functions'),
    'buckets': count(futures[2], 'buckets'),
    'users': count(futures[3], 'users'),
    'deployments': count(futures[4], 'targets'),
    'workflows': count(futures[5], 'workflows'),
    'usage': futures[6] ?? {},
    'platforms': count(futures[7], 'platforms'),
    'apiKeys': count(futures[8], 'keys'),
    'releases': (futures[9] as Map?)?['releases'] as List? ?? <dynamic>[],
  };
});

// --- Page --------------------------------------------------------------------

class OverviewPage extends ConsumerStatefulWidget {
  const OverviewPage({super.key});

  @override
  ConsumerState<OverviewPage> createState() => _OverviewPageState();
}

class _OverviewPageState extends ConsumerState<OverviewPage> {
  int _tabIndex = 0;

  @override
  Widget build(BuildContext context) {
    final cs = consoleColors(context);
    final projectId = ref.watch(currentProjectProvider);

    if (projectId == null) {
      return Scaffold(
        backgroundColor: cs.background,
        body: Center(
          child: Text('Select a project',
              style: TextStyle(color: cs.textMuted, fontSize: 15)),
        ),
      );
    }

    final projectsAsync = ref.watch(projectsProvider);
    final projectName = projectsAsync.valueOrNull
            ?.firstWhere((p) => p['\$id'] == projectId,
                orElse: () => <String, dynamic>{})['name'] as String? ??
        'Project';

    return Scaffold(
      backgroundColor: cs.background,
      body: Padding(
        padding: EdgeInsets.symmetric(
          horizontal: pageHPad(context),
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const SizedBox(height: 32),
            // Project name + ID
            Row(
              crossAxisAlignment: CrossAxisAlignment.end,
              children: [
                Text(projectName,
                    style: TextStyle(
                        color: cs.textPrimary,
                        fontSize: 24,
                        fontWeight: FontWeight.w700)),
                const SizedBox(width: 12),
                Padding(
                  padding: const EdgeInsets.only(bottom: 3),
                  child: Row(
                    children: [
                      Icon(LucideIcons.folder,
                          size: 13,
                          color: cs.textSubtle),
                      const SizedBox(width: 4),
                      IdText(id: projectId),
                    ],
                  ),
                ),
              ],
            ),
            const SizedBox(height: 24),
            PageTabs(
              tabs: const ['Overview', 'Activity'],
              selected: _tabIndex,
              onChanged: (i) => setState(() => _tabIndex = i),
            ),
            const SizedBox(height: 24),
            Expanded(
              child: SingleChildScrollView(
                child: _tabIndex == 0
                    ? _OverviewTab(projectId: projectId)
                    : _ActivityTab(projectId: projectId),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

// =============================================================================
// Overview Tab
// =============================================================================

class _OverviewTab extends ConsumerWidget {
  final String projectId;
  const _OverviewTab({required this.projectId});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final statsAsync = ref.watch(_projectStatsProvider(projectId));

    return statsAsync.when(
      loading: () => const Padding(
        padding: EdgeInsets.only(top: 80),
        child: Center(child: CircularProgressIndicator()),
      ),
      error: (e, _) => AppErrorState(error: e),
      data: (stats) {
        final usage = stats['usage'] as Map<String, dynamic>? ?? {};
        final releases = stats['releases'] as List? ?? [];

        return Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            // Usage graphs row
            LayoutBuilder(
              builder: (context, constraints) {
                final wide = constraints.maxWidth > 700;
                final graphWidth = wide
                    ? (constraints.maxWidth - 16) / 2
                    : constraints.maxWidth;
                return Wrap(
                  spacing: 16,
                  runSpacing: 16,
                  children: [
                    _UsageGraph(
                      width: graphWidth,
                      title: 'Requests',
                      value: _formatNumber(
                          usage['requests'] as int? ?? 0),
                      unit: '',
                      period: '30d',
                      data: _generateGraphData(
                          usage['requestsHistory'] as List? ?? [],
                          30),
                    ),
                    _UsageGraph(
                      width: graphWidth,
                      title: 'Bandwidth',
                      value: _formatBytes(
                          usage['bandwidth'] as int? ?? 0),
                      unit: '',
                      period: '30d',
                      data: _generateGraphData(
                          usage['bandwidthHistory'] as List? ?? [],
                          30),
                    ),
                  ],
                );
              },
            ),
            const SizedBox(height: 16),

            // Stat cards row
            LayoutBuilder(
              builder: (context, constraints) {
                final cardWidth =
                    (constraints.maxWidth - 16 * 3) / 4;
                return Wrap(
                  spacing: 16,
                  runSpacing: 16,
                  children: [
                    _StatCard(
                      icon: LucideIcons.database,
                      label: 'DATABASE',
                      value: '${stats['databases'] ?? 0}',
                      sublabel: 'Databases',
                      width: cardWidth,
                      onTap: () =>
                          context.go('/project/$projectId/databases'),
                    ),
                    _StatCard(
                      icon: LucideIcons.folderClosed,
                      label: 'STORAGE',
                      value: _formatBytes(
                          usage['storageBytes'] as int? ?? 0),
                      sublabel: 'Storage',
                      width: cardWidth,
                      onTap: () =>
                          context.go('/project/$projectId/storage'),
                    ),
                    _StatCard(
                      icon: LucideIcons.users,
                      label: 'AUTH',
                      value: '${stats['users'] ?? 0}',
                      sublabel: 'Users',
                      width: cardWidth,
                      onTap: () =>
                          context.go('/project/$projectId/auth'),
                    ),
                    _StatCard(
                      icon: LucideIcons.zap,
                      label: 'FUNCTIONS',
                      value: '${stats['functions'] ?? 0}',
                      sublabel: 'Executions',
                      width: cardWidth,
                      onTap: () =>
                          context.go('/project/$projectId/functions'),
                    ),
                  ],
                );
              },
            ),
            const SizedBox(height: 32),

            // Project ID + API Endpoint info cards
            _InfoCardsRow(projectId: projectId),
            const SizedBox(height: 16),

            // Recent Deployments
            _RecentDeployments(
              releases: releases,
              projectId: projectId,
            ),
            const SizedBox(height: 16),

            // Services overview grid
            _ServicesGrid(stats: stats, projectId: projectId),

            const SizedBox(height: 40),
          ],
        );
      },
    );
  }

  String _formatNumber(int n) {
    if (n >= 1000000) return '${(n / 1000000).toStringAsFixed(1)}M';
    if (n >= 1000) return '${(n / 1000).toStringAsFixed(1)}K';
    return '$n';
  }

  String _formatBytes(int bytes) {
    if (bytes <= 0) return '0 B';
    const units = ['B', 'KB', 'MB', 'GB', 'TB'];
    final i = (math.log(bytes) / math.log(1024)).floor().clamp(0, 4);
    final v = bytes / math.pow(1024, i);
    return '${v.toStringAsFixed(v < 10 ? 2 : 1)} ${units[i]}';
  }

  List<double> _generateGraphData(List<dynamic> history, int points) {
    if (history.isNotEmpty) {
      return history
          .take(points)
          .map((e) => (e is num ? e.toDouble() : 0.0))
          .toList();
    }
    // Return zeros if no history
    return List.filled(points, 0);
  }
}

// =============================================================================
// Usage Graph
// =============================================================================

class _UsageGraph extends StatelessWidget {
  final double width;
  final String title;
  final String value;
  final String unit;
  final String period;
  final List<double> data;

  const _UsageGraph({
    required this.width,
    required this.title,
    required this.value,
    required this.unit,
    required this.period,
    required this.data,
  });

  @override
  Widget build(BuildContext context) {
    final cs = consoleColors(context);
    return Container(
      width: width,
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: cs.surface,
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: cs.border),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Row(
                    crossAxisAlignment: CrossAxisAlignment.end,
                    children: [
                      Text(value,
                          style: TextStyle(
                              color: cs.textPrimary,
                              fontSize: 28,
                              fontWeight: FontWeight.w700)),
                      if (unit.isNotEmpty) ...[
                        const SizedBox(width: 4),
                        Padding(
                          padding: const EdgeInsets.only(bottom: 4),
                          child: Text(unit,
                              style: TextStyle(
                                  color: cs.textSecondary, fontSize: 14)),
                        ),
                      ],
                    ],
                  ),
                  const SizedBox(height: 2),
                  Text(title,
                      style: TextStyle(
                          color: cs.textSecondary, fontSize: 13)),
                ],
              ),
              const Spacer(),
              Container(
                padding: const EdgeInsets.symmetric(
                    horizontal: 10, vertical: 5),
                decoration: BoxDecoration(
                  color: cs.fill,
                  borderRadius: BorderRadius.circular(6),
                  border: Border.all(color: cs.border),
                ),
                child: Row(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    Text(period,
                        style: TextStyle(
                            color: cs.textSecondary, fontSize: 12)),
                    const SizedBox(width: 4),
                    Icon(LucideIcons.chevronDown,
                        size: 12, color: cs.textSecondary),
                  ],
                ),
              ),
            ],
          ),
          const SizedBox(height: 20),
          // Graph
          SizedBox(
            height: 100,
            child: CustomPaint(
              size: Size(width - 40, 100),
              painter: _GraphPainter(data: data, color: _accent),
            ),
          ),
        ],
      ),
    );
  }
}

class _GraphPainter extends CustomPainter {
  final List<double> data;
  final Color color;

  _GraphPainter({required this.data, required this.color});

  @override
  void paint(Canvas canvas, Size size) {
    if (data.isEmpty) return;

    final maxVal = data.reduce(math.max);
    final effectiveMax = maxVal > 0 ? maxVal : 1.0;

    // Grid lines
    final gridPaint = Paint()
      ..color = const Color(0xFF1A1A1A).withValues(alpha: 0.5)
      ..strokeWidth = 1;
    for (var i = 0; i < 5; i++) {
      final y = size.height * i / 4;
      canvas.drawLine(Offset(0, y), Offset(size.width, y), gridPaint);
    }

    // Bars
    final barWidth = (size.width / data.length) * 0.6;
    final gap = (size.width / data.length) * 0.4;
    final barPaint = Paint()
      ..color = color
      ..style = PaintingStyle.fill;

    for (var i = 0; i < data.length; i++) {
      final x = i * (barWidth + gap) + gap / 2;
      final h = (data[i] / effectiveMax) * (size.height - 4);
      if (h > 0) {
        final rect = RRect.fromRectAndRadius(
          Rect.fromLTWH(x, size.height - h, barWidth, h),
          const Radius.circular(2),
        );
        canvas.drawRRect(rect, barPaint);
      }
    }
  }

  @override
  bool shouldRepaint(covariant _GraphPainter old) =>
      data != old.data || color != old.color;
}

// =============================================================================
// Stat Card
// =============================================================================

class _StatCard extends StatefulWidget {
  final IconData icon;
  final String label;
  final String value;
  final String sublabel;
  final double width;
  final VoidCallback onTap;

  const _StatCard({
    required this.icon,
    required this.label,
    required this.value,
    required this.sublabel,
    required this.width,
    required this.onTap,
  });

  @override
  State<_StatCard> createState() => _StatCardState();
}

class _StatCardState extends State<_StatCard> {
  bool _hovered = false;

  @override
  Widget build(BuildContext context) {
    final cs = consoleColors(context);
    return MouseRegion(
      onEnter: (_) => setState(() => _hovered = true),
      onExit: (_) => setState(() => _hovered = false),
      cursor: SystemMouseCursors.click,
      child: GestureDetector(
        onTap: widget.onTap,
        child: AnimatedContainer(
          duration: const Duration(milliseconds: 150),
          width: widget.width,
          padding: const EdgeInsets.all(20),
          decoration: BoxDecoration(
            color: _hovered ? cs.fillHover : cs.surface,
            borderRadius: BorderRadius.circular(8),
            border: Border.all(
                color: _hovered ? cs.fieldBorder : cs.border),
          ),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Row(
                children: [
                  Icon(widget.icon, size: 14, color: _accent),
                  const SizedBox(width: 8),
                  Text(widget.label,
                      style: TextStyle(
                          color: cs.textMuted,
                          fontSize: 11,
                          fontWeight: FontWeight.w600,
                          letterSpacing: 0.5)),
                ],
              ),
              const SizedBox(height: 16),
              Text(widget.value,
                  style: TextStyle(
                      color: cs.textPrimary,
                      fontSize: 28,
                      fontWeight: FontWeight.w700)),
              const SizedBox(height: 2),
              Text(widget.sublabel,
                  style: TextStyle(color: cs.textSecondary, fontSize: 13)),
            ],
          ),
        ),
      ),
    );
  }
}

// =============================================================================
// Info Cards Row (Project ID + API Endpoint)
// =============================================================================

class _InfoCardsRow extends StatelessWidget {
  final String projectId;
  const _InfoCardsRow({required this.projectId});

  @override
  Widget build(BuildContext context) {
    final endpoint = '${Uri.base.origin}/v1';
    return LayoutBuilder(
      builder: (context, constraints) {
        final wide = constraints.maxWidth > 700;
        final w = wide ? (constraints.maxWidth - 16) / 2 : constraints.maxWidth;
        return Wrap(
          spacing: 16,
          runSpacing: 16,
          children: [
            _InfoCard(width: w, label: 'Project ID', value: projectId, icon: LucideIcons.folder),
            _InfoCard(width: w, label: 'API Endpoint', value: endpoint, icon: LucideIcons.link),
          ],
        );
      },
    );
  }
}

class _InfoCard extends StatelessWidget {
  final double width;
  final String label;
  final String value;
  final IconData icon;

  const _InfoCard({
    required this.width,
    required this.label,
    required this.value,
    required this.icon,
  });

  @override
  Widget build(BuildContext context) {
    final cs = consoleColors(context);
    return Container(
      width: width,
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: cs.surface,
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: cs.border),
      ),
      child: Row(
        children: [
          Icon(icon, size: 14, color: cs.textSecondary),
          const SizedBox(width: 10),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(label,
                    style: TextStyle(
                        color: cs.textSubtle,
                        fontSize: 11,
                        fontWeight: FontWeight.w500)),
                const SizedBox(height: 2),
                SelectableText(value,
                    style: TextStyle(
                        color: cs.textPrimary,
                        fontSize: 13,
                        fontFamily: 'monospace')),
              ],
            ),
          ),
          GestureDetector(
            onTap: () {
              Clipboard.setData(ClipboardData(text: value));
              ScaffoldMessenger.of(context).showSnackBar(
                const SnackBar(content: Text('Copied to clipboard')),
              );
            },
            child: MouseRegion(
              cursor: SystemMouseCursors.click,
              child: Icon(LucideIcons.copy, size: 14, color: cs.textSubtle),
            ),
          ),
        ],
      ),
    );
  }
}

// =============================================================================
// Recent Deployments
// =============================================================================

class _RecentDeployments extends StatelessWidget {
  final List<dynamic> releases;
  final String projectId;

  const _RecentDeployments({
    required this.releases,
    required this.projectId,
  });

  static Color _statusColor(String status) {
    switch (status) {
      case 'success':
        return const Color(0xFF22C55E);
      case 'failed':
        return const Color(0xFFEF4444);
      case 'building':
      case 'deploying':
        return const Color(0xFFF59E0B);
      case 'rolled_back':
        return const Color(0xFF8B5CF6);
      default:
        return const Color(0xFF6B7280);
    }
  }

  static String _statusLabel(String status) {
    switch (status) {
      case 'rolled_back':
        return 'Rolled back';
      default:
        return '${status[0].toUpperCase()}${status.substring(1)}';
    }
  }

  static String _timeAgo(String? iso) {
    if (iso == null) return '';
    try {
      final dt = DateTime.parse(iso).toLocal();
      final diff = DateTime.now().difference(dt);
      if (diff.inMinutes < 1) return 'just now';
      if (diff.inMinutes < 60) return '${diff.inMinutes}m ago';
      if (diff.inHours < 24) return '${diff.inHours}h ago';
      if (diff.inDays < 7) return '${diff.inDays}d ago';
      return '${(diff.inDays / 7).floor()}w ago';
    } catch (_) {
      return '';
    }
  }

  @override
  Widget build(BuildContext context) {
    final cs = consoleColors(context);

    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: cs.surface,
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: cs.border),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Text('Recent Deployments',
                  style: TextStyle(
                      color: cs.textPrimary,
                      fontSize: 15,
                      fontWeight: FontWeight.w600)),
              const Spacer(),
              MouseRegion(
                cursor: SystemMouseCursors.click,
                child: GestureDetector(
                  onTap: () => context.go('/project/$projectId/deploy'),
                  child: const Text('View all',
                      style: TextStyle(
                          color: _accent,
                          fontSize: 13,
                          decoration: TextDecoration.none)),
                ),
              ),
            ],
          ),
          const SizedBox(height: 16),
          if (releases.isEmpty)
            Padding(
              padding: const EdgeInsets.symmetric(vertical: 16),
              child: Row(
                children: [
                  Icon(LucideIcons.rocket, size: 16, color: cs.textMuted),
                  const SizedBox(width: 10),
                  Text('No deployments yet',
                      style: TextStyle(color: cs.textMuted, fontSize: 13)),
                ],
              ),
            )
          else
            ...releases.map<Widget>((r) {
              final rel = r as Map<String, dynamic>? ?? {};
              final status = rel['status'] as String? ?? 'pending';
              final color = _statusColor(status);
              final name = rel['name'] as String? ??
                  rel['\$id'] as String? ??
                  'Release';
              final time = _timeAgo(rel['createdAt'] as String?);

              return Padding(
                padding: const EdgeInsets.only(bottom: 10),
                child: Row(
                  children: [
                    // Status dot
                    Container(
                      width: 8,
                      height: 8,
                      decoration: BoxDecoration(
                        color: color,
                        shape: BoxShape.circle,
                      ),
                    ),
                    const SizedBox(width: 12),
                    // Name
                    Expanded(
                      child: Text(name,
                          style: TextStyle(
                              color: cs.textPrimary,
                              fontSize: 13),
                          overflow: TextOverflow.ellipsis),
                    ),
                    const SizedBox(width: 12),
                    // Status pill
                    Container(
                      padding: const EdgeInsets.symmetric(
                          horizontal: 8, vertical: 3),
                      decoration: BoxDecoration(
                        color: color.withValues(alpha: 0.12),
                        borderRadius: BorderRadius.circular(4),
                      ),
                      child: Text(_statusLabel(status),
                          style: TextStyle(
                              color: color,
                              fontSize: 11,
                              fontWeight: FontWeight.w500,
                              decoration: TextDecoration.none)),
                    ),
                    if (time.isNotEmpty) ...[
                      const SizedBox(width: 12),
                      Text(time,
                          style: TextStyle(
                              color: cs.textMuted, fontSize: 12)),
                    ],
                  ],
                ),
              );
            }),
        ],
      ),
    );
  }
}

// =============================================================================
// Services Overview Grid
// =============================================================================

class _ServicesGrid extends StatelessWidget {
  final Map<String, dynamic> stats;
  final String projectId;

  const _ServicesGrid({
    required this.stats,
    required this.projectId,
  });

  @override
  Widget build(BuildContext context) {
    final cs = consoleColors(context);

    final services = [
      _ServiceRow(
        icon: LucideIcons.users,
        label: 'Auth',
        value: '${stats['users'] ?? 0}',
        sublabel: 'users',
        route: '/project/$projectId/auth',
      ),
      _ServiceRow(
        icon: LucideIcons.database,
        label: 'Databases',
        value: '${stats['databases'] ?? 0}',
        sublabel: 'databases',
        route: '/project/$projectId/databases',
      ),
      _ServiceRow(
        icon: LucideIcons.folderClosed,
        label: 'Storage',
        value: '${stats['buckets'] ?? 0}',
        sublabel: 'buckets',
        route: '/project/$projectId/storage',
      ),
      _ServiceRow(
        icon: LucideIcons.zap,
        label: 'Functions',
        value: '${stats['functions'] ?? 0}',
        sublabel: 'functions',
        route: '/project/$projectId/functions',
      ),
      _ServiceRow(
        icon: LucideIcons.rocket,
        label: 'Deploy',
        value: '${stats['deployments'] ?? 0}',
        sublabel: 'targets',
        route: '/project/$projectId/deploy',
      ),
      _ServiceRow(
        icon: LucideIcons.gitBranch,
        label: 'Workflows',
        value: '${stats['workflows'] ?? 0}',
        sublabel: 'workflows',
        route: '/project/$projectId/workflows',
      ),
      _ServiceRow(
        icon: LucideIcons.mail,
        label: 'Messaging',
        value: '—',
        sublabel: 'email · sms · push',
        route: '/project/$projectId/messaging',
      ),
    ];

    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: cs.surface,
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: cs.border),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text('Services',
              style: TextStyle(
                  color: cs.textPrimary,
                  fontSize: 15,
                  fontWeight: FontWeight.w600)),
          const SizedBox(height: 16),
          LayoutBuilder(
            builder: (context, constraints) {
              final cols = constraints.maxWidth > 900
                  ? 3
                  : constraints.maxWidth > 600
                      ? 2
                      : 1;
              final cellW = (constraints.maxWidth - 16.0 * (cols - 1)) / cols;
              return Wrap(
                spacing: 16,
                runSpacing: 12,
                children: services
                    .map((s) => _ServiceCell(
                          service: s,
                          width: cellW,
                          onTap: () => context.go(s.route),
                        ))
                    .toList(),
              );
            },
          ),
        ],
      ),
    );
  }
}

class _ServiceRow {
  final IconData icon;
  final String label;
  final String value;
  final String sublabel;
  final String route;

  const _ServiceRow({
    required this.icon,
    required this.label,
    required this.value,
    required this.sublabel,
    required this.route,
  });
}

class _ServiceCell extends StatefulWidget {
  final _ServiceRow service;
  final double width;
  final VoidCallback onTap;

  const _ServiceCell({
    required this.service,
    required this.width,
    required this.onTap,
  });

  @override
  State<_ServiceCell> createState() => _ServiceCellState();
}

class _ServiceCellState extends State<_ServiceCell> {
  bool _hovered = false;

  @override
  Widget build(BuildContext context) {
    final cs = consoleColors(context);
    final s = widget.service;
    return MouseRegion(
      onEnter: (_) => setState(() => _hovered = true),
      onExit: (_) => setState(() => _hovered = false),
      cursor: SystemMouseCursors.click,
      child: GestureDetector(
        onTap: widget.onTap,
        child: AnimatedContainer(
          duration: const Duration(milliseconds: 130),
          width: widget.width,
          padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 12),
          decoration: BoxDecoration(
            color: _hovered ? cs.fillHover : cs.fill,
            borderRadius: BorderRadius.circular(8),
            border: Border.all(
                color: _hovered ? cs.fieldBorder : cs.border),
          ),
          child: Row(
            children: [
              Container(
                width: 32,
                height: 32,
                decoration: BoxDecoration(
                  color: _accent.withValues(alpha: 0.1),
                  borderRadius: BorderRadius.circular(7),
                ),
                child: Icon(s.icon, size: 15, color: _accent),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(s.label,
                        style: TextStyle(
                            color: cs.textPrimary,
                            fontSize: 13,
                            fontWeight: FontWeight.w500)),
                    const SizedBox(height: 1),
                    Text(s.sublabel,
                        style: TextStyle(
                            color: cs.textMuted, fontSize: 11)),
                  ],
                ),
              ),
              Text(s.value,
                  style: TextStyle(
                      color: cs.textSecondary,
                      fontSize: 15,
                      fontWeight: FontWeight.w600)),
            ],
          ),
        ),
      ),
    );
  }
}

// =============================================================================
// Activity Tab
// =============================================================================

class _ActivityTab extends ConsumerWidget {
  final String projectId;
  const _ActivityTab({required this.projectId});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final cs = consoleColors(context);
    final statsAsync = ref.watch(_projectStatsProvider(projectId));

    return statsAsync.when(
      loading: () => const Padding(
        padding: EdgeInsets.only(top: 80),
        child: Center(child: CircularProgressIndicator()),
      ),
      error: (e, _) => const SizedBox(),
      data: (stats) {
        final items = <_ActivityItem>[];

        void addIf(String key, String singular, String plural,
            String service, IconData icon) {
          final count = stats[key] as int? ?? 0;
          if (count > 0) {
            items.add(_ActivityItem(
              icon: icon,
              title: '$count ${count == 1 ? singular : plural}',
              subtitle: service,
            ));
          }
        }

        addIf('databases', 'database created', 'databases created',
            'Databases', LucideIcons.database);
        addIf('users', 'user registered', 'users registered', 'Auth',
            LucideIcons.users);
        addIf('buckets', 'bucket created', 'buckets created', 'Storage',
            LucideIcons.folderClosed);
        addIf('functions', 'function deployed', 'functions deployed',
            'Functions', LucideIcons.zap);
        addIf('deployments', 'deployment active', 'deployments active',
            'Deploy', LucideIcons.rocket);
        addIf('workflows', 'workflow configured', 'workflows configured',
            'Workflows', LucideIcons.gitBranch);

        if (items.isEmpty) {
          return Padding(
            padding: const EdgeInsets.only(top: 80),
            child: Center(
              child: Column(
                children: [
                  Container(
                    width: 48,
                    height: 48,
                    decoration: BoxDecoration(
                      color: cs.fill,
                      borderRadius: BorderRadius.circular(12),
                    ),
                    child: Icon(LucideIcons.activity,
                        size: 22, color: cs.textSubtle),
                  ),
                  const SizedBox(height: 16),
                  Text('No activity yet',
                      style: TextStyle(
                          color: cs.textPrimary,
                          fontSize: 15,
                          fontWeight: FontWeight.w500)),
                  const SizedBox(height: 6),
                  Text(
                      'Activity will appear here as you use your project',
                      style: TextStyle(color: cs.textSecondary, fontSize: 13)),
                ],
              ),
            ),
          );
        }

        return Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            ...items.map((item) => Container(
                  margin: const EdgeInsets.only(bottom: 8),
                  padding: const EdgeInsets.all(16),
                  decoration: BoxDecoration(
                    color: cs.surface,
                    borderRadius: BorderRadius.circular(8),
                    border: Border.all(color: cs.border),
                  ),
                  child: Row(
                    children: [
                      Container(
                        width: 36,
                        height: 36,
                        decoration: BoxDecoration(
                          color: _accent.withValues(alpha: 0.1),
                          borderRadius: BorderRadius.circular(8),
                        ),
                        child: Icon(item.icon,
                            size: 18, color: _accent),
                      ),
                      const SizedBox(width: 14),
                      Expanded(
                        child: Column(
                          crossAxisAlignment:
                              CrossAxisAlignment.start,
                          children: [
                            Text(item.title,
                                style: TextStyle(
                                    color: cs.textPrimary,
                                    fontSize: 14,
                                    fontWeight: FontWeight.w500)),
                            const SizedBox(height: 2),
                            Text(item.subtitle,
                                style: TextStyle(
                                    color: cs.textSecondary,
                                    fontSize: 12)),
                          ],
                        ),
                      ),
                    ],
                  ),
                )),
            const SizedBox(height: 40),
          ],
        );
      },
    );
  }
}

class _ActivityItem {
  final IconData icon;
  final String title;
  final String subtitle;

  const _ActivityItem({
    required this.icon,
    required this.title,
    required this.subtitle,
  });
}
