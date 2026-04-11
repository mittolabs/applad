import 'dart:async';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:lucide_icons/lucide_icons.dart';
import '../../core/api/client.dart';
import '../../core/providers/project_provider.dart';
import '../../core/theme/console_colors.dart';
import '../../core/utils/url_utils.dart';
import '../../core/widgets/page_tabs.dart';
import '../../core/widgets/app_empty_state.dart';
import '../../core/widgets/app_error_state.dart';

// ── Constants ─────────────────────────────────────────────────────────────────

const _accent = Color(0xFF6C47FF);

// ── Providers ─────────────────────────────────────────────────────────────────

final _realtimeStatsProvider = FutureProvider<Map<String, dynamic>>((ref) async {
  final pid = ref.watch(currentProjectProvider);
  if (pid == null) return {'connections': 0, 'channels': 0, 'channelList': []};
  final api = ref.read(apiClientProvider);
  final res = await api.get('/realtime/stats');
  return res.data as Map<String, dynamic>;
});

// ── Page ──────────────────────────────────────────────────────────────────────

class RealtimePage extends ConsumerStatefulWidget {
  const RealtimePage({super.key});

  @override
  ConsumerState<RealtimePage> createState() => _RealtimePageState();
}

class _RealtimePageState extends ConsumerState<RealtimePage> {
  static const _tabNames = ['overview', 'channels'];
  Timer? _refreshTimer;

  @override
  void initState() {
    super.initState();
    _refreshTimer = Timer.periodic(const Duration(seconds: 5), (_) {
      ref.invalidate(_realtimeStatsProvider);
    });
  }

  @override
  void dispose() {
    _refreshTimer?.cancel();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final colors = consoleColors(context);
    final tabName = tabFromQuery(context, defaultTab: 'overview');
    final tab = _tabNames.indexOf(tabName).clamp(0, _tabNames.length - 1);

    return Scaffold(
      backgroundColor: colors.background,
      body: Padding(
        padding: EdgeInsets.symmetric(horizontal: pageHPad(context)),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const SizedBox(height: 32),
            Text(
              'Realtime',
              style: TextStyle(
                color: colors.textPrimary,
                fontSize: 22,
                fontWeight: FontWeight.w600,
              ),
            ),
            const SizedBox(height: 4),
            Text('Broadcast and subscribe to live events over WebSocket connections',
                style: TextStyle(color: colors.textSecondary, fontSize: 13)),
            const SizedBox(height: 20),
            PageTabs(
              tabs: const ['Overview', 'Channels'],
              selected: tab,
              onChanged: (i) => context.go(
                withQuery(context, {'tab': _tabNames[i]}),
              ),
            ),
            const SizedBox(height: 20),
            Expanded(child: _tabBody(tab, colors)),
          ],
        ),
      ),
    );
  }

  Widget _tabBody(int tab, dynamic colors) {
    switch (tab) {
      case 0: return const _OverviewTab();
      case 1: return const _ChannelsTab();
      default: return const SizedBox.shrink();
    }
  }
}

// ── Overview tab ──────────────────────────────────────────────────────────────

class _OverviewTab extends ConsumerWidget {
  const _OverviewTab();

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final colors = consoleColors(context);
    final stats = ref.watch(_realtimeStatsProvider);

    return ListView(
      padding: EdgeInsets.zero,
      children: [
        stats.when(
          loading: () => const SizedBox(
            height: 80,
            child: Center(child: CircularProgressIndicator(strokeWidth: 2)),
          ),
          error: (_, __) => _errorCard(colors),
          data: (data) => Row(children: [
            _StatCard(
              icon: LucideIcons.users,
              label: 'Active connections',
              value: '${data['connections'] ?? 0}',
              colors: colors,
            ),
            const SizedBox(width: 16),
            _StatCard(
              icon: LucideIcons.radio,
              label: 'Active channels',
              value: '${data['channels'] ?? 0}',
              colors: colors,
            ),
          ]),
        ),
        const SizedBox(height: 32),

        Text(
          'How it works',
          style: TextStyle(
            fontSize: 14,
            fontWeight: FontWeight.w600,
            color: colors.textPrimary,
          ),
        ),
        const SizedBox(height: 12),
        _InfoRow(
          icon: LucideIcons.plug,
          title: 'WebSocket connection',
          body: 'Clients connect via WebSocket and subscribe to channels. The server pushes events in real time — no polling needed.',
          colors: colors,
        ),
        const SizedBox(height: 10),
        _InfoRow(
          icon: LucideIcons.database,
          title: 'Database changes',
          body: 'Row inserts, updates, and deletes in your databases are automatically broadcast to subscribers via PostgreSQL LISTEN/NOTIFY.',
          colors: colors,
        ),
        const SizedBox(height: 10),
        _InfoRow(
          icon: LucideIcons.gitBranch,
          title: 'Workflow events',
          body: 'Workflow execution status changes are published to realtime channels so clients can react instantly.',
          colors: colors,
        ),
        const SizedBox(height: 32),

        Text(
          'Channel patterns',
          style: TextStyle(
            fontSize: 14,
            fontWeight: FontWeight.w600,
            color: colors.textPrimary,
          ),
        ),
        const SizedBox(height: 12),
        _ChannelPatternRow(
          pattern: 'projects.{projectId}.databases.rows',
          description: 'All database row changes across the project',
          colors: colors,
        ),
        const SizedBox(height: 8),
        _ChannelPatternRow(
          pattern: 'databases.{projectId}.{databaseId}',
          description: 'All row changes in a specific database',
          colors: colors,
        ),
        const SizedBox(height: 8),
        _ChannelPatternRow(
          pattern: 'databases.{projectId}.{databaseId}.{tableName}',
          description: 'Row changes in a specific table',
          colors: colors,
        ),
        const SizedBox(height: 32),
      ],
    );
  }

  Widget _errorCard(dynamic colors) => Container(
        padding: const EdgeInsets.all(16),
        decoration: BoxDecoration(
          color: Colors.redAccent.withValues(alpha: 0.08),
          borderRadius: BorderRadius.circular(8),
          border: Border.all(color: Colors.redAccent.withValues(alpha: 0.2)),
        ),
        child: Row(children: [
          const Icon(LucideIcons.alertTriangle, size: 14, color: Colors.redAccent),
          const SizedBox(width: 8),
          Text(
            'Could not load stats',
            style: TextStyle(fontSize: 12, color: colors.textSecondary),
          ),
        ]),
      );
}

class _StatCard extends StatelessWidget {
  final IconData icon;
  final String label;
  final String value;
  final dynamic colors;

  const _StatCard({
    required this.icon,
    required this.label,
    required this.value,
    required this.colors,
  });

  @override
  Widget build(BuildContext context) {
    return Expanded(
      child: Container(
        padding: const EdgeInsets.all(20),
        decoration: BoxDecoration(
          color: colors.surface,
          borderRadius: BorderRadius.circular(8),
          border: Border.all(color: colors.border),
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Icon(icon, size: 16, color: colors.textMuted),
            const SizedBox(height: 12),
            Text(
              value,
              style: TextStyle(
                fontSize: 28,
                fontWeight: FontWeight.w700,
                color: colors.textPrimary,
              ),
            ),
            const SizedBox(height: 4),
            Text(
              label,
              style: TextStyle(fontSize: 12, color: colors.textSecondary),
            ),
          ],
        ),
      ),
    );
  }
}

class _InfoRow extends StatelessWidget {
  final IconData icon;
  final String title;
  final String body;
  final dynamic colors;

  const _InfoRow({
    required this.icon,
    required this.title,
    required this.body,
    required this.colors,
  });

  @override
  Widget build(BuildContext context) {
    return Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Container(
          width: 30,
          height: 30,
          decoration: BoxDecoration(
            color: colors.fill,
            borderRadius: BorderRadius.circular(6),
          ),
          child: Icon(icon, size: 14, color: colors.textMuted),
        ),
        const SizedBox(width: 12),
        Expanded(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                title,
                style: TextStyle(
                  fontSize: 13,
                  fontWeight: FontWeight.w500,
                  color: colors.textPrimary,
                ),
              ),
              const SizedBox(height: 3),
              Text(
                body,
                style: TextStyle(fontSize: 12, color: colors.textSecondary),
              ),
            ],
          ),
        ),
      ],
    );
  }
}

class _ChannelPatternRow extends StatelessWidget {
  final String pattern;
  final String description;
  final dynamic colors;

  const _ChannelPatternRow({
    required this.pattern,
    required this.description,
    required this.colors,
  });

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 10),
      decoration: BoxDecoration(
        color: colors.surface,
        borderRadius: BorderRadius.circular(6),
        border: Border.all(color: colors.border),
      ),
      child: Row(children: [
        Expanded(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                pattern,
                style: const TextStyle(
                  fontFamily: 'monospace',
                  fontSize: 12,
                  color: _accent,
                ),
              ),
              const SizedBox(height: 3),
              Text(
                description,
                style: TextStyle(fontSize: 11, color: colors.textSecondary),
              ),
            ],
          ),
        ),
        GestureDetector(
          onTap: () => Clipboard.setData(ClipboardData(text: pattern)),
          child: Icon(LucideIcons.copy, size: 13, color: colors.textMuted),
        ),
      ]),
    );
  }
}

// ── Channels tab ─────────────────────────────────────────────────────────────

class _ChannelsTab extends ConsumerWidget {
  const _ChannelsTab();

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final colors = consoleColors(context);
    final stats = ref.watch(_realtimeStatsProvider);

    return stats.when(
      loading: () => const Center(child: CircularProgressIndicator(strokeWidth: 2)),
      error: (e, __) => AppErrorState(error: e),
      data: (data) {
        final channels =
            (data['channelList'] as List? ?? []).cast<Map<String, dynamic>>();
        if (channels.isEmpty) {
          return const AppEmptyState(
            icon: LucideIcons.radio,
            title: 'No active channels',
            subtitle: 'Channels appear here when clients connect and subscribe.',
          );
        }

        return ListView(
          padding: EdgeInsets.zero,
          children: [
            Text(
              '${channels.length} active channel${channels.length == 1 ? '' : 's'}',
              style: TextStyle(fontSize: 12, color: colors.textSecondary),
            ),
            const SizedBox(height: 12),
            ...channels.map((ch) => Padding(
                  padding: const EdgeInsets.only(bottom: 6),
                  child: Container(
                    padding: const EdgeInsets.symmetric(
                        horizontal: 14, vertical: 10),
                    decoration: BoxDecoration(
                      color: colors.surface,
                      borderRadius: BorderRadius.circular(6),
                      border: Border.all(color: colors.border),
                    ),
                    child: Row(children: [
                      Container(
                        width: 6,
                        height: 6,
                        decoration: const BoxDecoration(
                          color: Color(0xFF34D399),
                          shape: BoxShape.circle,
                        ),
                      ),
                      const SizedBox(width: 10),
                      Expanded(
                        child: Text(
                          ch['channel'] as String? ?? '',
                          style: TextStyle(
                            fontFamily: 'monospace',
                            fontSize: 12,
                            color: colors.textPrimary,
                          ),
                        ),
                      ),
                      Container(
                        padding: const EdgeInsets.symmetric(
                            horizontal: 8, vertical: 3),
                        decoration: BoxDecoration(
                          color: colors.fill,
                          borderRadius: BorderRadius.circular(4),
                        ),
                        child: Text(
                          '${ch['subscribers']} subscriber${(ch['subscribers'] as int? ?? 0) == 1 ? '' : 's'}',
                          style: TextStyle(
                            fontSize: 11,
                            color: colors.textSecondary,
                          ),
                        ),
                      ),
                    ]),
                  ),
                )),
          ],
        );
      },
    );
  }
}

