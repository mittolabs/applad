import 'dart:math' as math;
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:lucide_icons/lucide_icons.dart';
import '../../core/api/client.dart';
import '../../core/providers/project_provider.dart';
import '../../core/widgets/page_tabs.dart';

// --- Constants ---------------------------------------------------------------

const _bgColor = Color(0xFF0B0B0F);
const _cardColor = Color(0xFF16171B);
const _accent = Color(0xFF3472A4);
const _dimText = Color(0x80FFFFFF);
const _subtleText = Color(0x40FFFFFF);
const _border = Color(0x0FFFFFFF);

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
    final projectId = ref.watch(currentProjectProvider);

    if (projectId == null) {
      return const Scaffold(
        backgroundColor: _bgColor,
        body: Center(
          child: Text('Select a project',
              style: TextStyle(color: _dimText, fontSize: 15)),
        ),
      );
    }

    final projectsAsync = ref.watch(projectsProvider);
    final projectName = projectsAsync.valueOrNull
            ?.firstWhere((p) => p['\$id'] == projectId,
                orElse: () => <String, dynamic>{})['name'] as String? ??
        'Project';

    return Scaffold(
      backgroundColor: _bgColor,
      body: Padding(
        padding: EdgeInsets.symmetric(
          horizontal: MediaQuery.of(context).size.width > 1400 ? 80 : 40,
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
                    style: const TextStyle(
                        color: Colors.white,
                        fontSize: 24,
                        fontWeight: FontWeight.w700)),
                const SizedBox(width: 12),
                Padding(
                  padding: const EdgeInsets.only(bottom: 3),
                  child: Row(
                    children: [
                      Icon(LucideIcons.folder,
                          size: 13,
                          color: Colors.white.withOpacity(0.3)),
                      const SizedBox(width: 4),
                      SelectableText(
                        projectId.length > 16
                            ? '${projectId.substring(0, 16)}...'
                            : projectId,
                        style: TextStyle(
                            color: Colors.white.withOpacity(0.3),
                            fontSize: 13,
                            fontFamily: 'monospace'),
                      ),
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
      error: (e, _) => Padding(
        padding: const EdgeInsets.only(top: 80),
        child: Center(
            child: Text('Error: $e',
                style: const TextStyle(color: Colors.red))),
      ),
      data: (stats) {
        final usage = stats['usage'] as Map<String, dynamic>? ?? {};

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

            // Project info + SDK quickstart
            _ProjectInfoSection(projectId: projectId),

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
    return Container(
      width: width,
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: _cardColor,
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: _border),
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
                          style: const TextStyle(
                              color: Colors.white,
                              fontSize: 28,
                              fontWeight: FontWeight.w700)),
                      if (unit.isNotEmpty) ...[
                        const SizedBox(width: 4),
                        Padding(
                          padding: const EdgeInsets.only(bottom: 4),
                          child: Text(unit,
                              style: const TextStyle(
                                  color: _dimText, fontSize: 14)),
                        ),
                      ],
                    ],
                  ),
                  const SizedBox(height: 2),
                  Text(title,
                      style: const TextStyle(
                          color: _dimText, fontSize: 13)),
                ],
              ),
              const Spacer(),
              Container(
                padding: const EdgeInsets.symmetric(
                    horizontal: 10, vertical: 5),
                decoration: BoxDecoration(
                  color: Colors.white.withOpacity(0.04),
                  borderRadius: BorderRadius.circular(6),
                  border: Border.all(color: _border),
                ),
                child: Row(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    Text(period,
                        style: const TextStyle(
                            color: _dimText, fontSize: 12)),
                    const SizedBox(width: 4),
                    const Icon(LucideIcons.chevronDown,
                        size: 12, color: _dimText),
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
      ..color = Colors.white.withOpacity(0.04)
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
            color: _hovered
                ? Colors.white.withOpacity(0.04)
                : _cardColor,
            borderRadius: BorderRadius.circular(8),
            border: Border.all(
                color: _hovered
                    ? Colors.white.withOpacity(0.1)
                    : _border),
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
                          color: Colors.white.withOpacity(0.5),
                          fontSize: 11,
                          fontWeight: FontWeight.w600,
                          letterSpacing: 0.5)),
                ],
              ),
              const SizedBox(height: 16),
              Text(widget.value,
                  style: const TextStyle(
                      color: Colors.white,
                      fontSize: 28,
                      fontWeight: FontWeight.w700)),
              const SizedBox(height: 2),
              Text(widget.sublabel,
                  style:
                      const TextStyle(color: _dimText, fontSize: 13)),
            ],
          ),
        ),
      ),
    );
  }
}

// =============================================================================
// Project Info + SDK Quickstart
// =============================================================================

class _ProjectInfoSection extends StatefulWidget {
  final String projectId;
  const _ProjectInfoSection({required this.projectId});

  @override
  State<_ProjectInfoSection> createState() => _ProjectInfoSectionState();
}

class _ProjectInfoSectionState extends State<_ProjectInfoSection> {
  int _sdkTab = 0;

  static const _sdks = [
    _SdkInfo('Flutter', 'dart', 'applad: ^1.0.0',
        'dependencies:', LucideIcons.smartphone),
    _SdkInfo('JavaScript', 'bash', 'npm install applad',
        'Terminal', LucideIcons.terminal),
    _SdkInfo('Node.js', 'bash', 'npm install applad-node',
        'Terminal', LucideIcons.server),
    _SdkInfo('Python', 'bash', 'pip install applad',
        'Terminal', LucideIcons.terminal),
    _SdkInfo('Go', 'bash', 'go get github.com/mittolabs/applad-go',
        'Terminal', LucideIcons.terminal),
  ];

  @override
  Widget build(BuildContext context) {
    final endpoint = '${Uri.base.origin}/v1';
    final sdk = _sdks[_sdkTab];

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        // Project info row
        LayoutBuilder(
          builder: (context, constraints) {
            final wide = constraints.maxWidth > 700;
            final infoWidth = wide
                ? (constraints.maxWidth - 16) / 2
                : constraints.maxWidth;

            return Wrap(
              spacing: 16,
              runSpacing: 16,
              children: [
                _InfoCard(
                  width: infoWidth,
                  label: 'Project ID',
                  value: widget.projectId,
                  icon: LucideIcons.folder,
                ),
                _InfoCard(
                  width: infoWidth,
                  label: 'API Endpoint',
                  value: endpoint,
                  icon: LucideIcons.link,
                ),
              ],
            );
          },
        ),
        const SizedBox(height: 16),

        // SDK quickstart
        Container(
          width: double.infinity,
          padding: const EdgeInsets.all(20),
          decoration: BoxDecoration(
            color: _cardColor,
            borderRadius: BorderRadius.circular(8),
            border: Border.all(color: _border),
          ),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              const Text('Quick start',
                  style: TextStyle(
                      color: Colors.white,
                      fontSize: 15,
                      fontWeight: FontWeight.w600)),
              const SizedBox(height: 4),
              Text('Install an SDK to start building with Applad',
                  style: TextStyle(
                      color: Colors.white.withOpacity(0.4),
                      fontSize: 13)),
              const SizedBox(height: 16),
              // SDK tabs
              SingleChildScrollView(
                scrollDirection: Axis.horizontal,
                child: Row(
                  children: List.generate(_sdks.length, (i) {
                    final s = _sdks[i];
                    final active = _sdkTab == i;
                    return Padding(
                      padding: const EdgeInsets.only(right: 6),
                      child: GestureDetector(
                        onTap: () => setState(() => _sdkTab = i),
                        child: MouseRegion(
                          cursor: SystemMouseCursors.click,
                          child: Container(
                            padding: const EdgeInsets.symmetric(
                                horizontal: 12, vertical: 6),
                            decoration: BoxDecoration(
                              color: active
                                  ? Colors.white.withOpacity(0.08)
                                  : Colors.transparent,
                              borderRadius: BorderRadius.circular(6),
                              border: Border.all(
                                  color: active
                                      ? Colors.white.withOpacity(0.12)
                                      : Colors.white.withOpacity(0.06)),
                            ),
                            child: Text(s.name,
                                style: TextStyle(
                                    color: active
                                        ? Colors.white
                                        : _dimText,
                                    fontSize: 12,
                                    fontWeight: active
                                        ? FontWeight.w500
                                        : FontWeight.w400)),
                          ),
                        ),
                      ),
                    );
                  }),
                ),
              ),
              const SizedBox(height: 14),
              // Install command
              Container(
                width: double.infinity,
                padding: const EdgeInsets.all(14),
                decoration: BoxDecoration(
                  color: Colors.white.withOpacity(0.03),
                  borderRadius: BorderRadius.circular(8),
                  border: Border.all(color: _border),
                ),
                child: Row(
                  children: [
                    Text(sdk.contextLabel,
                        style: TextStyle(
                            color: Colors.white.withOpacity(0.3),
                            fontSize: 12)),
                    const SizedBox(width: 12),
                    Expanded(
                      child: SelectableText(sdk.installCmd,
                          style: const TextStyle(
                              color: Colors.white,
                              fontSize: 13,
                              fontFamily: 'monospace')),
                    ),
                    GestureDetector(
                      onTap: () {
                        Clipboard.setData(
                            ClipboardData(text: sdk.installCmd));
                        ScaffoldMessenger.of(context).showSnackBar(
                          const SnackBar(
                              content: Text('Copied to clipboard')),
                        );
                      },
                      child: MouseRegion(
                        cursor: SystemMouseCursors.click,
                        child: Icon(LucideIcons.copy,
                            size: 14,
                            color: Colors.white.withOpacity(0.3)),
                      ),
                    ),
                  ],
                ),
              ),
              const SizedBox(height: 14),
              // Init snippet
              Container(
                width: double.infinity,
                padding: const EdgeInsets.all(14),
                decoration: BoxDecoration(
                  color: Colors.white.withOpacity(0.03),
                  borderRadius: BorderRadius.circular(8),
                  border: Border.all(color: _border),
                ),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Row(
                      children: [
                        Expanded(
                          child: SelectableText(
                            _initSnippet(sdk.name, endpoint,
                                widget.projectId),
                            style: const TextStyle(
                                color: Colors.white,
                                fontSize: 13,
                                fontFamily: 'monospace',
                                height: 1.5),
                          ),
                        ),
                        Align(
                          alignment: Alignment.topRight,
                          child: GestureDetector(
                            onTap: () {
                              Clipboard.setData(ClipboardData(
                                  text: _initSnippet(sdk.name,
                                      endpoint, widget.projectId)));
                              ScaffoldMessenger.of(context).showSnackBar(
                                const SnackBar(
                                    content:
                                        Text('Copied to clipboard')),
                              );
                            },
                            child: MouseRegion(
                              cursor: SystemMouseCursors.click,
                              child: Icon(LucideIcons.copy,
                                  size: 14,
                                  color:
                                      Colors.white.withOpacity(0.3)),
                            ),
                          ),
                        ),
                      ],
                    ),
                  ],
                ),
              ),
            ],
          ),
        ),
      ],
    );
  }

  String _initSnippet(String sdk, String endpoint, String projectId) {
    switch (sdk) {
      case 'Flutter':
        return "import 'package:applad/applad.dart';\n\nfinal client = AppladClient()\n  .setEndpoint('$endpoint')\n  .setProject('$projectId');";
      case 'JavaScript':
        return "import { Client } from 'applad';\n\nconst client = new Client()\n  .setEndpoint('$endpoint')\n  .setProject('$projectId');";
      case 'Node.js':
        return "const { Client } = require('applad-node');\n\nconst client = new Client()\n  .setEndpoint('$endpoint')\n  .setProject('$projectId')\n  .setKey('YOUR_API_KEY');";
      case 'Python':
        return "from applad import Client\n\nclient = Client()\nclient.set_endpoint('$endpoint')\nclient.set_project('$projectId')\nclient.set_key('YOUR_API_KEY')";
      case 'Go':
        return 'import "github.com/mittolabs/applad-go"\n\nclient := applad.NewClient()\nclient.SetEndpoint("$endpoint")\nclient.SetProject("$projectId")\nclient.SetKey("YOUR_API_KEY")';
      default:
        return '';
    }
  }
}

class _SdkInfo {
  final String name;
  final String lang;
  final String installCmd;
  final String contextLabel;
  final IconData icon;

  const _SdkInfo(
      this.name, this.lang, this.installCmd, this.contextLabel, this.icon);
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
    return Container(
      width: width,
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: _cardColor,
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: _border),
      ),
      child: Row(
        children: [
          Icon(icon, size: 14, color: _dimText),
          const SizedBox(width: 10),
          Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(label,
                  style: TextStyle(
                      color: Colors.white.withOpacity(0.4),
                      fontSize: 11,
                      fontWeight: FontWeight.w500)),
              const SizedBox(height: 2),
              SelectableText(value,
                  style: const TextStyle(
                      color: Colors.white,
                      fontSize: 13,
                      fontFamily: 'monospace')),
            ],
          ),
          const Spacer(),
          GestureDetector(
            onTap: () {
              Clipboard.setData(ClipboardData(text: value));
              ScaffoldMessenger.of(context).showSnackBar(
                const SnackBar(content: Text('Copied to clipboard')),
              );
            },
            child: MouseRegion(
              cursor: SystemMouseCursors.click,
              child: Icon(LucideIcons.copy,
                  size: 14, color: Colors.white.withOpacity(0.3)),
            ),
          ),
        ],
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
                      color: Colors.white.withOpacity(0.04),
                      borderRadius: BorderRadius.circular(12),
                    ),
                    child: const Icon(LucideIcons.activity,
                        size: 22, color: _subtleText),
                  ),
                  const SizedBox(height: 16),
                  const Text('No activity yet',
                      style: TextStyle(
                          color: Colors.white,
                          fontSize: 15,
                          fontWeight: FontWeight.w500)),
                  const SizedBox(height: 6),
                  const Text(
                      'Activity will appear here as you use your project',
                      style:
                          TextStyle(color: _dimText, fontSize: 13)),
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
                    color: _cardColor,
                    borderRadius: BorderRadius.circular(8),
                    border: Border.all(color: _border),
                  ),
                  child: Row(
                    children: [
                      Container(
                        width: 36,
                        height: 36,
                        decoration: BoxDecoration(
                          color: _accent.withOpacity(0.1),
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
                                style: const TextStyle(
                                    color: Colors.white,
                                    fontSize: 14,
                                    fontWeight: FontWeight.w500)),
                            const SizedBox(height: 2),
                            Text(item.subtitle,
                                style: const TextStyle(
                                    color: _dimText,
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
