import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:lucide_icons_flutter/lucide_icons.dart';
import '../../core/api/client.dart';
import '../../core/providers/auth_provider.dart' show consoleAuthProvider;
import '../../core/providers/get_started_provider.dart';
import '../../core/theme/console_colors.dart';
import '../../core/utils/url_utils.dart';
import '../../core/widgets/app_error_state.dart';
import '../../core/widgets/code_block.dart';

// --- Constants ---------------------------------------------------------------

const _accent = Color(0xFF3472A4);
const _green = Color(0xFF10B981);

// --- Provider ----------------------------------------------------------------

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
      return "import { Applad } from 'applad';\n\nconst client = new Applad()\n  .setEndpoint('https://your-domain.com/v1')\n  .setProject('$projectId');";
    case 'Node.js':
      return "import { Client } from 'applad-node';\n\nconst client = new Client()\n  .setEndpoint('https://your-domain.com/v1')\n  .setProject('$projectId')\n  .setKey('YOUR_API_KEY');";
    case 'Python':
      return "from applad import Client\n\nclient = Client()\nclient.set_endpoint('https://your-domain.com/v1')\nclient.set_project('$projectId')\nclient.set_key('YOUR_API_KEY')";
    case 'Flutter':
    default:
      return "import 'package:applad/applad.dart';\n\nfinal client = Applad()\n  .setEndpoint('https://your-domain.com/v1')\n  .setProject('$projectId');";
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
  bool _markedDone = false;

  @override
  Widget build(BuildContext context) {
    final cs = consoleColors(context);
    final projectId = GoRouterState.of(context).pathParameters['projectId'];
    final userName = ref.watch(consoleAuthProvider).valueOrNull?.name ?? '';

    if (projectId == null) {
      return Scaffold(
        backgroundColor: cs.background,
        body: Center(
          child: Text('No project selected',
              style: TextStyle(color: cs.textMuted)),
        ),
      );
    }

    // If already permanently dismissed, redirect immediately.
    final isDone = ref.watch(getStartedDoneProvider(projectId));
    if (isDone) {
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (mounted) context.go('/project/$projectId/overview');
      });
      return const SizedBox.shrink();
    }

    final dataAsync = ref.watch(_getStartedDataProvider(projectId));

    return Scaffold(
      backgroundColor: cs.background,
      body: dataAsync.when(
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (e, _) => AppErrorState(error: e),
        data: (counts) {
          final steps = _buildSteps(projectId, counts);
          final completed = steps.where((s) => s.done).length;
          final allDone = completed == steps.length;
          final firstKey = counts['firstKey'] as String? ?? '';

          // All steps done for the first time → record permanently and leave.
          if (allDone && !_markedDone) {
            _markedDone = true;
            WidgetsBinding.instance.addPostFrameCallback((_) {
              if (!mounted) return;
              markGetStartedDone(projectId, ref);
              context.go('/project/$projectId/overview');
            });
            return const SizedBox.shrink();
          }

          return SingleChildScrollView(
            padding: EdgeInsets.symmetric(
              horizontal:
                  pageHPad(context),
              vertical: 32,
            ),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                // ── Page header ─────────────────────────────────────────
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
                                : 'Get started with Applad',
                            style: TextStyle(
                                color: cs.textPrimary,
                                fontSize: 24,
                                fontWeight: FontWeight.w700),
                          ),
                          const SizedBox(height: 4),
                          Text(
                            'Complete these steps to set up your project',
                            style: TextStyle(
                                color: cs.textSubtle, fontSize: 14),
                          ),
                        ],
                      ),
                    ),
                  ],
                ),
                const SizedBox(height: 28),

                // ── Body ────────────────────────────────────────────────
                LayoutBuilder(
                  builder: (_, constraints) {
                    final wide = constraints.maxWidth > 860;

                    final leftCol = Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        // Unified steps card (Dyser-style)
                        _StepsCard(
                          steps: steps,
                          projectId: projectId,
                          completed: completed,
                        ),
                        const SizedBox(height: 28),

                        // Explore section
                        Text('Explore Applad',
                            style: TextStyle(
                                color: cs.textPrimary,
                                fontSize: 15,
                                fontWeight: FontWeight.w600)),
                        const SizedBox(height: 4),
                        Text(
                          'Dive deeper into what you can build',
                          style: TextStyle(
                              color: cs.textSubtle, fontSize: 13),
                        ),
                        const SizedBox(height: 14),
                        _ExploreGrid(projectId: projectId),
                      ],
                    );

                    final rightCol = Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
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

    return [
      _Step(
        title: 'Create your project',
        description:
            'Your Applad project is created and ready to use. '
            'Each project has its own database, storage, and API keys.',
        done: true,
        doneSummary: 'Project created',
        route: '/project/$projectId/overview',
        actionLabel: 'View overview',
        icon: LucideIcons.folderCheck,
      ),
      _Step(
        title: 'Connect a platform',
        description:
            'Register your web, iOS, Android, or Flutter app to enable '
            'API access and unlock OAuth, push notifications, and more.',
        done: (counts['platforms'] as int? ?? 0) > 0,
        doneSummary:
            '${counts['platforms']} platform${(counts['platforms'] as int? ?? 0) != 1 ? 's' : ''} connected',
        route: '/project/$projectId/settings',
        actionLabel: 'Go to Settings',
        icon: LucideIcons.smartphone,
      ),
      _Step(
        title: 'Build your first feature',
        description:
            'Add users with Auth, store data in Databases, serve files '
            'from Storage, or trigger logic with Functions.',
        done: hasAnyResource,
        doneSummary: _buildSummary(counts),
        route: '/project/$projectId/databases',
        actionLabel: 'Explore services',
        icon: LucideIcons.layers,
      ),
    ];
  }

  static String _buildSummary(Map<String, dynamic> counts) {
    final db = counts['databases'] as int? ?? 0;
    final u = counts['users'] as int? ?? 0;
    if (db > 0 && u > 0) return '$db databases · $u users';
    if (db > 0) return '$db database${db > 1 ? 's' : ''} created';
    if (u > 0) return '$u user${u > 1 ? 's' : ''} signed up';
    return 'Resources configured';
  }
}

// --- Models ------------------------------------------------------------------

class _Step {
  final String title;
  final String description;
  final bool done;
  final String doneSummary;
  final String route;
  final String actionLabel;
  final IconData icon;

  const _Step({
    required this.title,
    required this.description,
    required this.done,
    required this.doneSummary,
    required this.route,
    required this.actionLabel,
    required this.icon,
  });
}

// =============================================================================
// Unified Steps Card (Dyser-style)
// =============================================================================

class _StepsCard extends StatelessWidget {
  final List<_Step> steps;
  final String projectId;
  final int completed;

  const _StepsCard({
    required this.steps,
    required this.projectId,
    required this.completed,
  });

  @override
  Widget build(BuildContext context) {
    final cs = consoleColors(context);
    final total = steps.length;

    return Container(
      decoration: BoxDecoration(
        color: cs.surface,
        borderRadius: BorderRadius.circular(10),
        border: Border.all(color: cs.border),
      ),
      child: Column(
        children: [
          // Card header with progress
          Padding(
            padding: const EdgeInsets.fromLTRB(20, 18, 20, 14),
            child: Row(
              crossAxisAlignment: CrossAxisAlignment.center,
              children: [
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        'Setup checklist',
                        style: TextStyle(
                            color: cs.textPrimary,
                            fontSize: 15,
                            fontWeight: FontWeight.w600),
                      ),
                      const SizedBox(height: 2),
                      Text(
                        '$completed of $total steps complete',
                        style: TextStyle(
                            color: cs.textMuted, fontSize: 12),
                      ),
                    ],
                  ),
                ),
                // Compact progress bar
                SizedBox(
                  width: 120,
                  child: ClipRRect(
                    borderRadius: BorderRadius.circular(2),
                    child: LinearProgressIndicator(
                      value: total == 0 ? 0 : completed / total,
                      minHeight: 4,
                      backgroundColor: cs.fill,
                      valueColor: AlwaysStoppedAnimation(
                        completed == total ? _green : _accent,
                      ),
                    ),
                  ),
                ),
              ],
            ),
          ),
          Divider(height: 1, color: cs.border),

          // Steps
          ...steps.asMap().entries.map((entry) {
            final i = entry.key;
            final step = entry.value;
            final isCurrent = !step.done && (i == 0 || steps[i - 1].done);
            final isLast = i == steps.length - 1;

            return Column(
              children: [
                _StepRow(
                  number: i + 1,
                  step: step,
                  isCurrent: isCurrent,
                  onTap: () => context.go(step.route),
                ),
                if (!isLast) Divider(height: 1, color: cs.border),
              ],
            );
          }),
        ],
      ),
    );
  }
}

// --- Step row ----------------------------------------------------------------

class _StepRow extends StatefulWidget {
  final int number;
  final _Step step;
  final bool isCurrent;
  final VoidCallback onTap;

  const _StepRow({
    required this.number,
    required this.step,
    required this.isCurrent,
    required this.onTap,
  });

  @override
  State<_StepRow> createState() => _StepRowState();
}

class _StepRowState extends State<_StepRow> {
  bool _hovered = false;

  @override
  Widget build(BuildContext context) {
    final cs = consoleColors(context);
    final step = widget.step;
    final isDone = step.done;
    final isCurrent = widget.isCurrent;
    final isPending = !isDone && !isCurrent;

    return MouseRegion(
      onEnter: (_) => setState(() => _hovered = true),
      onExit: (_) => setState(() => _hovered = false),
      cursor: isCurrent ? SystemMouseCursors.click : SystemMouseCursors.basic,
      child: AnimatedContainer(
        duration: const Duration(milliseconds: 130),
        color: _hovered && isCurrent
            ? cs.fillHover
            : Colors.transparent,
        child: Padding(
          padding: const EdgeInsets.fromLTRB(20, 18, 20, 18),
          child: Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              // Circle indicator
              _StepCircle(
                number: widget.number,
                isDone: isDone,
                isCurrent: isCurrent,
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
                              fontSize: 14,
                              fontWeight: isCurrent || isDone
                                  ? FontWeight.w600
                                  : FontWeight.w400,
                            ),
                          ),
                        ),
                        if (isDone) ...[
                          const SizedBox(width: 10),
                          Container(
                            padding: const EdgeInsets.symmetric(
                                horizontal: 8, vertical: 3),
                            decoration: BoxDecoration(
                              color: _green.withValues(alpha: 0.1),
                              borderRadius: BorderRadius.circular(4),
                            ),
                            child: Text(
                              step.doneSummary,
                              style: const TextStyle(
                                color: _green,
                                fontSize: 11,
                                fontWeight: FontWeight.w500,
                                decoration: TextDecoration.none,
                              ),
                            ),
                          ),
                        ],
                      ],
                    ),
                    const SizedBox(height: 4),
                    Text(
                      step.description,
                      style: TextStyle(
                        color: isPending ? cs.textSubtle : cs.textMuted,
                        fontSize: 13,
                        height: 1.45,
                      ),
                    ),
                    if (isCurrent) ...[
                      const SizedBox(height: 14),
                      GestureDetector(
                        onTap: widget.onTap,
                        child: MouseRegion(
                          cursor: SystemMouseCursors.click,
                          child: Container(
                            padding: const EdgeInsets.symmetric(
                                horizontal: 14, vertical: 8),
                            decoration: BoxDecoration(
                              color: _accent,
                              borderRadius: BorderRadius.circular(7),
                            ),
                            child: Row(
                              mainAxisSize: MainAxisSize.min,
                              children: [
                                Text(
                                  step.actionLabel,
                                  style: const TextStyle(
                                    color: Colors.white,
                                    fontSize: 13,
                                    fontWeight: FontWeight.w500,
                                    decoration: TextDecoration.none,
                                  ),
                                ),
                                const SizedBox(width: 6),
                                const Icon(LucideIcons.arrowRight,
                                    size: 13, color: Colors.white),
                              ],
                            ),
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
    );
  }
}

class _StepCircle extends StatelessWidget {
  final int number;
  final bool isDone;
  final bool isCurrent;

  const _StepCircle({
    required this.number,
    required this.isDone,
    required this.isCurrent,
  });

  @override
  Widget build(BuildContext context) {
    final cs = consoleColors(context);

    if (isDone) {
      return Container(
        width: 28,
        height: 28,
        decoration: BoxDecoration(
          color: _green.withValues(alpha: 0.15),
          shape: BoxShape.circle,
        ),
        child: const Icon(LucideIcons.check, size: 14, color: _green),
      );
    }

    if (isCurrent) {
      return Container(
        width: 28,
        height: 28,
        decoration: BoxDecoration(
          color: _accent.withValues(alpha: 0.15),
          shape: BoxShape.circle,
          border: Border.all(
              color: _accent.withValues(alpha: 0.5), width: 1.5),
        ),
        child: Center(
          child: Text(
            '0$number',
            style: const TextStyle(
              color: _accent,
              fontSize: 11,
              fontWeight: FontWeight.w700,
              fontFamily: 'monospace',
            ),
          ),
        ),
      );
    }

    // Pending
    return Container(
      width: 28,
      height: 28,
      decoration: BoxDecoration(
        color: cs.fill,
        shape: BoxShape.circle,
        border: Border.all(color: cs.border, width: 1.5),
      ),
      child: Center(
        child: Text(
          '0$number',
          style: TextStyle(
            color: cs.textSubtle,
            fontSize: 11,
            fontWeight: FontWeight.w600,
            fontFamily: 'monospace',
          ),
        ),
      ),
    );
  }
}

// =============================================================================
// Explore Grid
// =============================================================================

class _ExploreGrid extends StatelessWidget {
  final String projectId;
  const _ExploreGrid({required this.projectId});

  static const _items = [
    (
      LucideIcons.users,
      'Auth',
      'Sign in, sessions, OAuth, MFA',
      'auth',
    ),
    (
      LucideIcons.database,
      'Databases',
      'Tables, rows, relationships',
      'databases',
    ),
    (
      LucideIcons.hardDrive,
      'Storage',
      'Buckets, files, image transforms',
      'storage',
    ),
    (
      LucideIcons.zap,
      'Functions',
      'Serverless code execution',
      'functions',
    ),
    (
      LucideIcons.mail,
      'Messaging',
      'Email, SMS, push notifications',
      'messaging',
    ),
    (
      LucideIcons.gitBranch,
      'Workflows',
      'DAG automation engine',
      'workflows',
    ),
  ];

  @override
  Widget build(BuildContext context) {
    return LayoutBuilder(
      builder: (context, constraints) {
        final cols = constraints.maxWidth > 700 ? 3 : 2;
        final cellW =
            (constraints.maxWidth - 16.0 * (cols - 1)) / cols;
        return Wrap(
          spacing: 16,
          runSpacing: 14,
          children: _items.map((item) {
            return _ExploreCard(
              width: cellW,
              icon: item.$1,
              title: item.$2,
              description: item.$3,
              onTap: () => context
                  .go('/project/$projectId/${item.$4}'),
            );
          }).toList(),
        );
      },
    );
  }
}

class _ExploreCard extends StatefulWidget {
  final double width;
  final IconData icon;
  final String title;
  final String description;
  final VoidCallback onTap;

  const _ExploreCard({
    required this.width,
    required this.icon,
    required this.title,
    required this.description,
    required this.onTap,
  });

  @override
  State<_ExploreCard> createState() => _ExploreCardState();
}

class _ExploreCardState extends State<_ExploreCard> {
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
          duration: const Duration(milliseconds: 130),
          width: widget.width,
          padding: const EdgeInsets.all(16),
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
                  Container(
                    width: 32,
                    height: 32,
                    decoration: BoxDecoration(
                      color: _accent.withValues(alpha: 0.1),
                      borderRadius: BorderRadius.circular(7),
                    ),
                    child: Icon(widget.icon, size: 15, color: _accent),
                  ),
                  const Spacer(),
                  Icon(LucideIcons.arrowUpRight,
                      size: 13,
                      color: _hovered ? _accent : cs.textSubtle),
                ],
              ),
              const SizedBox(height: 12),
              Text(widget.title,
                  style: TextStyle(
                      color: cs.textPrimary,
                      fontSize: 13,
                      fontWeight: FontWeight.w600)),
              const SizedBox(height: 3),
              Text(widget.description,
                  style:
                      TextStyle(color: cs.textMuted, fontSize: 12)),
            ],
          ),
        ),
      ),
    );
  }
}

// =============================================================================
// Copy Card
// =============================================================================

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
            child: MouseRegion(
              cursor: SystemMouseCursors.click,
              child: AnimatedSwitcher(
                duration: const Duration(milliseconds: 200),
                child: _copied
                    ? const Icon(LucideIcons.check,
                        key: ValueKey('check'),
                        size: 14,
                        color: _green)
                    : Icon(LucideIcons.copy,
                        key: const ValueKey('copy'),
                        size: 14,
                        color: cs.textSubtle),
              ),
            ),
          ),
        ],
      ),
    );
  }
}

// =============================================================================
// SDK Panel
// =============================================================================

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
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 12),
            child: Row(
              children: _sdkPlatforms.map((p) {
                final sel = selectedPlatform == p;
                return GestureDetector(
                  onTap: () => onPlatformChanged(p),
                  child: MouseRegion(
                    cursor: SystemMouseCursors.click,
                    child: AnimatedContainer(
                      duration: const Duration(milliseconds: 120),
                      margin: const EdgeInsets.only(right: 4),
                      padding: const EdgeInsets.symmetric(
                          horizontal: 10, vertical: 5),
                      decoration: BoxDecoration(
                        color: sel
                            ? _accent.withValues(alpha: 0.15)
                            : Colors.transparent,
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
                          fontWeight: sel
                              ? FontWeight.w600
                              : FontWeight.w400,
                          decoration: TextDecoration.none,
                        ),
                      ),
                    ),
                  ),
                );
              }).toList(),
            ),
          ),
          const SizedBox(height: 10),
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
                CodeBlock(
                  code: snippet,
                  language: selectedPlatform == 'Flutter' ? 'dart' : selectedPlatform,
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
              ? const Icon(LucideIcons.check,
                  key: ValueKey('ck'), size: 13, color: _green)
              : Icon(LucideIcons.copy,
                  key: const ValueKey('cp'),
                  size: 13,
                  color: cs.textSubtle),
        ),
      ),
    );
  }
}
