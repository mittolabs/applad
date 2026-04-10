import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:lucide_icons/lucide_icons.dart';
import '../../core/api/client.dart';
import '../../core/providers/auth_provider.dart' show consoleAuthProvider;
import '../../core/theme/console_colors.dart';

// --- Constants ---------------------------------------------------------------

const _accent = Color(0xFF3472A4);
const _green = Color(0xFF10B981);

// --- Providers ---------------------------------------------------------------

final _getStartedDataProvider =
    FutureProvider.family<Map<String, dynamic>, String>((ref, projectId) async {
  final api = ref.read(apiClientProvider);

  Future<int> count(String path, String key) async {
    try {
      final res = await api.get(path);
      final data = res.data;
      if (data is Map && data[key] is List) return (data[key] as List).length;
      if (data is Map && data['total'] is int) return data['total'] as int;
    } catch (_) {}
    return 0;
  }

  Future<String> fetchFirstKey() async {
    try {
      final res = await api.get('/projects/$projectId/keys');
      final data = res.data as Map<String, dynamic>?;
      final keys = data?['keys'] as List?;
      if (keys != null && keys.isNotEmpty) {
        return (keys.first as Map<String, dynamic>)['secret'] as String? ?? '';
      }
    } catch (_) {}
    return '';
  }

  final results = await Future.wait([
    count('/databases', 'databases'),
    count('/account/users', 'users'),
    count('/storage/buckets', 'buckets'),
    count('/functions', 'functions'),
    count('/deploy/targets', 'targets'),
    count('/workflows', 'workflows'),
    count('/projects/$projectId/keys', 'keys'),
    count('/projects/$projectId/platforms', 'platforms'),
  ]);

  final apiKey = await fetchFirstKey();

  return {
    'databases': results[0],
    'users': results[1],
    'buckets': results[2],
    'functions': results[3],
    'deployments': results[4],
    'workflows': results[5],
    'apiKeys': results[6],
    'platforms': results[7],
    'firstKey': apiKey,
  };
});

// --- SDK snippets ------------------------------------------------------------

const _sdkPlatforms = ['Flutter', 'JavaScript', 'Node.js', 'Python'];

String _sdkSnippet(String platform, String projectId) {
  switch (platform) {
    case 'JavaScript':
      return '''import { Applad } from 'applad';

const client = new Applad()
  .setEndpoint('https://your-domain.com/v1')
  .setProject('$projectId');''';
    case 'Node.js':
      return '''import { Client } from 'applad-node';

const client = new Client()
  .setEndpoint('https://your-domain.com/v1')
  .setProject('$projectId')
  .setKey('YOUR_API_KEY');''';
    case 'Python':
      return '''from applad import Client

client = Client()
client.set_endpoint('https://your-domain.com/v1')
client.set_project('$projectId')
client.set_key('YOUR_API_KEY')''';
    case 'Flutter':
    default:
      return '''import 'package:applad/applad.dart';

final client = Applad()
  .setEndpoint('https://your-domain.com/v1')
  .setProject('$projectId');''';
  }
}

// --- Page --------------------------------------------------------------------

class GetStartedPage extends ConsumerStatefulWidget {
  const GetStartedPage({super.key});

  @override
  ConsumerState<GetStartedPage> createState() => _GetStartedPageState();
}

class _GetStartedPageState extends ConsumerState<GetStartedPage> {
  String _selectedPlatform = 'Flutter';

  @override
  Widget build(BuildContext context) {
    final cs = consoleColors(context);
    final routerState = GoRouterState.of(context);
    final projectId = routerState.pathParameters['projectId'];
    final userName = ref.watch(consoleAuthProvider).valueOrNull?.name ?? '';

    if (projectId == null) {
      return Scaffold(
        backgroundColor: cs.background,
        body: Center(
            child: Text('No project selected',
                style: TextStyle(color: cs.textMuted))),
      );
    }

    final dataAsync = ref.watch(_getStartedDataProvider(projectId));

    return Scaffold(
      backgroundColor: cs.background,
      body: dataAsync.when(
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (e, _) => Center(
            child: Text('Error: $e',
                style: const TextStyle(color: Colors.red))),
        data: (counts) {
          final steps = _buildSteps(projectId, counts);
          final completed = steps.where((s) => s.done).length;
          final firstKey = counts['firstKey'] as String? ?? '';

          return SingleChildScrollView(
            padding: EdgeInsets.symmetric(
              horizontal:
                  MediaQuery.of(context).size.width > 1400 ? 80 : 40,
              vertical: 32,
            ),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                // ── Header ──────────────────────────────────────────────
                Row(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Expanded(
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text(
                            userName.isNotEmpty
                                ? 'Welcome, $userName'
                                : 'Welcome',
                            style: TextStyle(
                                color: cs.textPrimary,
                                fontSize: 24,
                                fontWeight: FontWeight.w700),
                          ),
                          const SizedBox(height: 4),
                          Text(
                            'Follow a few quick steps to get started with Applad',
                            style: TextStyle(
                                color: cs.textSubtle, fontSize: 14),
                          ),
                        ],
                      ),
                    ),
                    TextButton(
                      onPressed: () =>
                          context.go('/project/$projectId/overview'),
                      style: TextButton.styleFrom(
                          foregroundColor: cs.textMuted),
                      child: const Text('Dismiss this page',
                          style: TextStyle(fontSize: 13)),
                    ),
                  ],
                ),
                const SizedBox(height: 6),

                // ── Progress bar ─────────────────────────────────────────
                _ProgressBar(
                    completed: completed, total: steps.length),
                const SizedBox(height: 28),

                // ── Two-column body ──────────────────────────────────────
                LayoutBuilder(
                  builder: (_, constraints) {
                    final wide = constraints.maxWidth > 860;
                    final leftCol = Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        // Step cards
                        ...steps.asMap().entries.map((entry) {
                          final i = entry.key;
                          final step = entry.value;
                          final isCurrent =
                              !step.done && (i == 0 || steps[i - 1].done);
                          return Padding(
                            padding: const EdgeInsets.only(bottom: 12),
                            child: _StepCard(
                              number: i + 1,
                              step: step,
                              isCurrent: isCurrent,
                              onTap: () => context.go(step.route),
                            ),
                          );
                        }),
                        const SizedBox(height: 28),

                        // Services grid
                        Text('Services',
                            style: TextStyle(
                                color: cs.textSecondary,
                                fontSize: 12,
                                fontWeight: FontWeight.w600,
                                letterSpacing: 0.5)),
                        const SizedBox(height: 12),
                        _ServicesGrid(projectId: projectId),
                      ],
                    );

                    final rightCol = Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        // Project ID copy
                        _CopyCard(
                          label: 'Project ID',
                          value: projectId,
                          icon: LucideIcons.fingerprint,
                        ),
                        if (firstKey.isNotEmpty) ...[
                          const SizedBox(height: 12),
                          _CopyCard(
                            label: 'API Key',
                            value: firstKey,
                            icon: LucideIcons.key,
                            obscure: true,
                          ),
                        ],
                        const SizedBox(height: 16),

                        // SDK snippet
                        _SdkPanel(
                          projectId: projectId,
                          selectedPlatform: _selectedPlatform,
                          onPlatformChanged: (p) =>
                              setState(() => _selectedPlatform = p),
                        ),
                      ],
                    );

                    if (wide) {
                      return Row(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Expanded(flex: 58, child: leftCol),
                          const SizedBox(width: 24),
                          SizedBox(width: 320, child: rightCol),
                        ],
                      );
                    }
                    return Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        leftCol,
                        const SizedBox(height: 24),
                        rightCol,
                      ],
                    );
                  },
                ),
                const SizedBox(height: 40),
              ],
            ),
          );
        },
      ),
    );
  }

  List<_Step> _buildSteps(String projectId, Map<String, dynamic> counts) {
    final hasAnyResource = (counts['databases'] as int? ?? 0) > 0 ||
        (counts['buckets'] as int? ?? 0) > 0 ||
        (counts['functions'] as int? ?? 0) > 0 ||
        (counts['deployments'] as int? ?? 0) > 0 ||
        (counts['workflows'] as int? ?? 0) > 0;

    final dbCount = counts['databases'] as int? ?? 0;
    final userCount = counts['users'] as int? ?? 0;

    String buildSummary() {
      if (dbCount > 0 && userCount > 0) return '$dbCount databases · $userCount users';
      if (dbCount > 0) return '$dbCount database${dbCount > 1 ? 's' : ''} created';
      if (userCount > 0) return '$userCount user${userCount > 1 ? 's' : ''} signed up';
      return 'Resources configured';
    }

    return [
      _Step(
        title: 'Create project',
        subtitle: 'Your Applad project is ready to use.',
        done: true,
        doneSummary: 'Project created',
        route: '/project/$projectId/overview',
        actionLabel: 'View overview',
      ),
      _Step(
        title: 'Connect your platform',
        subtitle:
            'Register your web, iOS, Android, or Flutter app to enable API access.',
        done: (counts['platforms'] as int? ?? 0) > 0,
        doneSummary: '${counts['platforms']} platform${(counts['platforms'] as int? ?? 0) != 1 ? 's' : ''} connected',
        route: '/project/$projectId/settings',
        actionLabel: 'Go to Settings',
      ),
      _Step(
        title: 'Build your app',
        subtitle:
            'Set up Auth, Databases, Storage, Functions, or Workflows to power your app.',
        done: hasAnyResource,
        doneSummary: buildSummary(),
        route: '/project/$projectId/databases',
        actionLabel: 'Explore services',
      ),
    ];
  }
}

// --- Models ------------------------------------------------------------------

class _Step {
  final String title;
  final String subtitle;
  final bool done;
  final String doneSummary;
  final String route;
  final String actionLabel;

  const _Step({
    required this.title,
    required this.subtitle,
    required this.done,
    required this.doneSummary,
    required this.route,
    required this.actionLabel,
  });
}

// --- Widgets -----------------------------------------------------------------

class _ProgressBar extends StatelessWidget {
  final int completed;
  final int total;

  const _ProgressBar({required this.completed, required this.total});

  @override
  Widget build(BuildContext context) {
    final cs = consoleColors(context);
    final pct = total == 0 ? 0.0 : completed / total;
    return Row(
      children: [
        Expanded(
          child: ClipRRect(
            borderRadius: BorderRadius.circular(2),
            child: LinearProgressIndicator(
              value: pct,
              minHeight: 3,
              backgroundColor: cs.fill,
              valueColor: const AlwaysStoppedAnimation(_green),
            ),
          ),
        ),
        const SizedBox(width: 12),
        Text(
          '$completed of $total steps complete',
          style: TextStyle(color: cs.textSubtle, fontSize: 12),
        ),
      ],
    );
  }
}

// ── Step card ─────────────────────────────────────────────────────────────────

class _StepCard extends StatefulWidget {
  final int number;
  final _Step step;
  final bool isCurrent;
  final VoidCallback onTap;

  const _StepCard({
    required this.number,
    required this.step,
    required this.isCurrent,
    required this.onTap,
  });

  @override
  State<_StepCard> createState() => _StepCardState();
}

class _StepCardState extends State<_StepCard> {
  bool _hovered = false;

  @override
  Widget build(BuildContext context) {
    final cs = consoleColors(context);
    final step = widget.step;
    final isDone = step.done;
    final isCurrent = widget.isCurrent;
    final isPending = !isDone && !isCurrent;

    // Left border color
    Color leftBorder = Colors.transparent;
    if (isDone) leftBorder = _green;
    if (isCurrent) leftBorder = _accent;

    // Background
    Color bgColor = cs.surface;
    if (isPending) bgColor = cs.surfaceAlt;
    if (isCurrent && _hovered) bgColor = cs.fillHover;

    // Number badge color
    Color badgeBg = isDone
        ? _green.withValues(alpha: 0.15)
        : isCurrent
            ? _accent.withValues(alpha: 0.15)
            : cs.fill;
    Color badgeFg = isDone
        ? _green
        : isCurrent
            ? _accent
            : cs.textSubtle;

    return MouseRegion(
      onEnter: (_) => setState(() => _hovered = true),
      onExit: (_) => setState(() => _hovered = false),
      cursor: isCurrent ? SystemMouseCursors.click : SystemMouseCursors.basic,
      child: GestureDetector(
        onTap: isCurrent ? widget.onTap : null,
        child: AnimatedContainer(
          duration: const Duration(milliseconds: 150),
          decoration: BoxDecoration(
            color: bgColor,
            borderRadius: BorderRadius.circular(8),
            border: Border(
              left: BorderSide(color: leftBorder, width: 3),
              top: BorderSide(color: cs.border),
              right: BorderSide(color: cs.border),
              bottom: BorderSide(color: cs.border),
            ),
            boxShadow: isCurrent
                ? [
                    BoxShadow(
                      color: cs.shadow,
                      blurRadius: 8,
                      offset: const Offset(0, 2),
                    )
                  ]
                : null,
          ),
          child: Padding(
            padding: const EdgeInsets.all(20),
            child: Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                // Number badge
                Container(
                  width: 32,
                  height: 32,
                  decoration: BoxDecoration(
                    color: badgeBg,
                    borderRadius: BorderRadius.circular(6),
                  ),
                  alignment: Alignment.center,
                  child: isDone
                      ? Icon(LucideIcons.check, size: 14, color: _green)
                      : Text(
                          '0${widget.number}',
                          style: TextStyle(
                            color: badgeFg,
                            fontSize: 12,
                            fontWeight: FontWeight.w700,
                            fontFamily: 'monospace',
                          ),
                        ),
                ),
                const SizedBox(width: 16),

                // Content
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Row(
                        children: [
                          Expanded(
                            child: Text(
                              step.title,
                              style: TextStyle(
                                color: isPending
                                    ? cs.textSubtle
                                    : cs.textPrimary,
                                fontSize: 15,
                                fontWeight: isCurrent
                                    ? FontWeight.w600
                                    : FontWeight.w500,
                              ),
                            ),
                          ),
                          if (isDone) ...[
                            const SizedBox(width: 8),
                            Container(
                              padding: const EdgeInsets.symmetric(
                                  horizontal: 8, vertical: 3),
                              decoration: BoxDecoration(
                                color: _green.withValues(alpha: 0.12),
                                borderRadius: BorderRadius.circular(4),
                              ),
                              child: Text(
                                step.doneSummary,
                                style: const TextStyle(
                                  color: _green,
                                  fontSize: 11,
                                  fontWeight: FontWeight.w500,
                                ),
                              ),
                            ),
                          ],
                          if (isCurrent) ...[
                            const SizedBox(width: 8),
                            Container(
                              padding: const EdgeInsets.symmetric(
                                  horizontal: 8, vertical: 3),
                              decoration: BoxDecoration(
                                color: _accent.withValues(alpha: 0.12),
                                borderRadius: BorderRadius.circular(4),
                              ),
                              child: const Text(
                                'Up next',
                                style: TextStyle(
                                  color: Color(0xFF60A5FA),
                                  fontSize: 11,
                                  fontWeight: FontWeight.w500,
                                ),
                              ),
                            ),
                          ],
                        ],
                      ),
                      const SizedBox(height: 4),
                      Text(
                        step.subtitle,
                        style: TextStyle(
                          color: isPending ? cs.textSubtle : cs.textMuted,
                          fontSize: 13,
                          height: 1.4,
                        ),
                      ),
                      if (isCurrent) ...[
                        const SizedBox(height: 14),
                        GestureDetector(
                          onTap: widget.onTap,
                          child: Row(
                            mainAxisSize: MainAxisSize.min,
                            children: [
                              Text(
                                step.actionLabel,
                                style: const TextStyle(
                                  color: _accent,
                                  fontSize: 13,
                                  fontWeight: FontWeight.w500,
                                ),
                              ),
                              const SizedBox(width: 4),
                              const Icon(LucideIcons.arrowRight,
                                  size: 13, color: _accent),
                            ],
                          ),
                        ),
                      ],
                      if (isDone) ...[
                        const SizedBox(height: 10),
                        GestureDetector(
                          onTap: widget.onTap,
                          child: Text(
                            step.actionLabel,
                            style: TextStyle(
                              color: cs.textSubtle,
                              fontSize: 12,
                              decoration: TextDecoration.underline,
                              decorationColor: cs.textSubtle,
                            ),
                          ),
                        ),
                      ],
                    ],
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

// ── Services grid ─────────────────────────────────────────────────────────────

class _ServicesGrid extends StatelessWidget {
  final String projectId;

  const _ServicesGrid({required this.projectId});

  static const _services = [
    (LucideIcons.users, 'Auth', 'auth'),
    (LucideIcons.database, 'Databases', 'databases'),
    (LucideIcons.hardDrive, 'Storage', 'storage'),
    (LucideIcons.zap, 'Functions', 'functions'),
    (LucideIcons.messageSquare, 'Messaging', 'messaging'),
    (LucideIcons.gitBranch, 'Workflows', 'workflows'),
  ];

  @override
  Widget build(BuildContext context) {
    return Wrap(
      spacing: 10,
      runSpacing: 10,
      children: _services.map((s) {
        final icon = s.$1;
        final label = s.$2;
        final route = s.$3;
        return _ServiceChip(
          icon: icon,
          label: label,
          onTap: () => GoRouter.of(context)
              .go('/project/$projectId/$route'),
        );
      }).toList(),
    );
  }
}

class _ServiceChip extends StatefulWidget {
  final IconData icon;
  final String label;
  final VoidCallback onTap;

  const _ServiceChip({
    required this.icon,
    required this.label,
    required this.onTap,
  });

  @override
  State<_ServiceChip> createState() => _ServiceChipState();
}

class _ServiceChipState extends State<_ServiceChip> {
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
          duration: const Duration(milliseconds: 120),
          padding:
              const EdgeInsets.symmetric(horizontal: 14, vertical: 10),
          decoration: BoxDecoration(
            color: _hovered ? cs.fillHover : cs.surface,
            borderRadius: BorderRadius.circular(8),
            border: Border.all(
                color: _hovered ? cs.border : cs.border),
          ),
          child: Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              Icon(widget.icon,
                  size: 14,
                  color: _hovered ? _accent : cs.textMuted),
              const SizedBox(width: 8),
              Text(
                widget.label,
                style: TextStyle(
                  color: _hovered ? cs.textPrimary : cs.textSecondary,
                  fontSize: 13,
                  fontWeight: FontWeight.w500,
                ),
              ),
              const SizedBox(width: 6),
              Icon(LucideIcons.arrowRight,
                  size: 11,
                  color: _hovered ? _accent : cs.textSubtle),
            ],
          ),
        ),
      ),
    );
  }
}

// ── Copy card ─────────────────────────────────────────────────────────────────

class _CopyCard extends StatefulWidget {
  final String label;
  final String value;
  final IconData icon;
  final bool obscure;

  const _CopyCard({
    required this.label,
    required this.value,
    required this.icon,
    this.obscure = false,
  });

  @override
  State<_CopyCard> createState() => _CopyCardState();
}

class _CopyCardState extends State<_CopyCard> {
  bool _copied = false;

  void _copy() async {
    await Clipboard.setData(ClipboardData(text: widget.value));
    setState(() => _copied = true);
    await Future.delayed(const Duration(seconds: 2));
    if (mounted) setState(() => _copied = false);
  }

  @override
  Widget build(BuildContext context) {
    final cs = consoleColors(context);
    final display = widget.obscure
        ? '${widget.value.substring(0, widget.value.length.clamp(0, 8))}••••••••'
        : widget.value;

    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 12),
      decoration: BoxDecoration(
        color: cs.surface,
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: cs.border),
      ),
      child: Row(
        children: [
          Icon(widget.icon, size: 14, color: cs.textSubtle),
          const SizedBox(width: 10),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(widget.label,
                    style: TextStyle(
                        color: cs.textSubtle,
                        fontSize: 10,
                        fontWeight: FontWeight.w500,
                        letterSpacing: 0.4)),
                const SizedBox(height: 2),
                Text(
                  display,
                  style: TextStyle(
                    color: cs.textPrimary,
                    fontSize: 12,
                    fontFamily: 'monospace',
                  ),
                  overflow: TextOverflow.ellipsis,
                ),
              ],
            ),
          ),
          const SizedBox(width: 8),
          GestureDetector(
            onTap: _copy,
            child: AnimatedSwitcher(
              duration: const Duration(milliseconds: 200),
              child: _copied
                  ? Icon(LucideIcons.check,
                      key: const ValueKey('check'),
                      size: 14,
                      color: _green)
                  : Icon(LucideIcons.copy,
                      key: const ValueKey('copy'),
                      size: 14,
                      color: cs.textSubtle),
            ),
          ),
        ],
      ),
    );
  }
}

// ── SDK panel ─────────────────────────────────────────────────────────────────

class _SdkPanel extends StatelessWidget {
  final String projectId;
  final String selectedPlatform;
  final ValueChanged<String> onPlatformChanged;

  const _SdkPanel({
    required this.projectId,
    required this.selectedPlatform,
    required this.onPlatformChanged,
  });

  @override
  Widget build(BuildContext context) {
    final cs = consoleColors(context);
    final snippet = _sdkSnippet(selectedPlatform, projectId);

    return Container(
      decoration: BoxDecoration(
        color: cs.surface,
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: cs.border),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          // Title bar
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 14, 16, 0),
            child: Row(
              children: [
                Icon(LucideIcons.code2, size: 14, color: cs.textSubtle),
                const SizedBox(width: 8),
                Text('Quick start',
                    style: TextStyle(
                        color: cs.textSecondary,
                        fontSize: 13,
                        fontWeight: FontWeight.w500)),
              ],
            ),
          ),
          const SizedBox(height: 12),

          // Platform tabs
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 12),
            child: Row(
              children: _sdkPlatforms.map((p) {
                final sel = selectedPlatform == p;
                return GestureDetector(
                  onTap: () => onPlatformChanged(p),
                  child: AnimatedContainer(
                    duration: const Duration(milliseconds: 120),
                    margin: const EdgeInsets.only(right: 4),
                    padding: const EdgeInsets.symmetric(
                        horizontal: 10, vertical: 5),
                    decoration: BoxDecoration(
                      color: sel ? _accent.withValues(alpha: 0.15) : Colors.transparent,
                      borderRadius: BorderRadius.circular(5),
                      border: Border.all(
                          color: sel
                              ? _accent.withValues(alpha: 0.5)
                              : Colors.transparent),
                    ),
                    child: Text(
                      p,
                      style: TextStyle(
                        color: sel ? _accent : cs.textSubtle,
                        fontSize: 11,
                        fontWeight: sel ? FontWeight.w600 : FontWeight.w400,
                      ),
                    ),
                  ),
                );
              }).toList(),
            ),
          ),
          const SizedBox(height: 10),

          // Code block
          Container(
            width: double.infinity,
            margin: const EdgeInsets.fromLTRB(12, 0, 12, 12),
            padding: const EdgeInsets.all(14),
            decoration: BoxDecoration(
              color: cs.surfaceAlt,
              borderRadius: BorderRadius.circular(6),
            ),
            child: Stack(
              children: [
                SelectableText(
                  snippet,
                  style: TextStyle(
                    color: cs.textSecondary,
                    fontSize: 11.5,
                    fontFamily: 'monospace',
                    height: 1.7,
                  ),
                ),
                Positioned(
                  top: 0,
                  right: 0,
                  child: _CodeCopyButton(code: snippet),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

class _CodeCopyButton extends StatefulWidget {
  final String code;
  const _CodeCopyButton({required this.code});

  @override
  State<_CodeCopyButton> createState() => _CodeCopyButtonState();
}

class _CodeCopyButtonState extends State<_CodeCopyButton> {
  bool _copied = false;

  void _copy() async {
    await Clipboard.setData(ClipboardData(text: widget.code));
    setState(() => _copied = true);
    await Future.delayed(const Duration(seconds: 2));
    if (mounted) setState(() => _copied = false);
  }

  @override
  Widget build(BuildContext context) {
    final cs = consoleColors(context);
    return GestureDetector(
      onTap: _copy,
      child: MouseRegion(
        cursor: SystemMouseCursors.click,
        child: AnimatedSwitcher(
          duration: const Duration(milliseconds: 180),
          child: _copied
              ? Icon(LucideIcons.check,
                  key: const ValueKey('ck'), size: 13, color: _green)
              : Icon(LucideIcons.copy,
                  key: const ValueKey('cp'),
                  size: 13,
                  color: cs.textSubtle),
        ),
      ),
    );
  }
}
