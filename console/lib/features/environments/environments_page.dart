import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:lucide_icons/lucide_icons.dart';
import '../../core/api/client.dart';
import '../../core/providers/environment_provider.dart';
import '../../core/theme/console_colors.dart';
import '../../core/utils/url_utils.dart';
import '../../core/widgets/app_dialog.dart';
import '../../core/widgets/page_tabs.dart';
import '../../core/widgets/app_error_state.dart';

// --- Local constants ---------------------------------------------------------

const _accent = Color(0xFF6C47FF);
const _green = Color(0xFF10B981);
const _amber = Color(0xFFF59E0B);
const _purple = Color(0xFF8B5CF6);

// --- Providers ---------------------------------------------------------------

// _selectedEnvProvider and _envDetailTabProvider removed — state now lives in URL (?envId=, ?tab=)

// --- Page --------------------------------------------------------------------

class EnvironmentsPage extends ConsumerWidget {
  const EnvironmentsPage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final colors = consoleColors(context);
    final envsAsync = ref.watch(environmentsProvider);
    final selectedId = GoRouterState.of(context).uri.queryParameters['envId'];

    return Scaffold(
      backgroundColor: colors.background,
      body: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(24, 20, 24, 0),
            child: Row(
              children: [
                Text('Environments',
                    style: Theme.of(context)
                        .textTheme
                        .headlineSmall
                    ?.copyWith(color: colors.textPrimary)),
                const Spacer(),
                FilledButton.icon(
                  style: FilledButton.styleFrom(
                    backgroundColor: _accent,
                    shape: RoundedRectangleBorder(
                        borderRadius: BorderRadius.circular(8)),
                    padding: const EdgeInsets.symmetric(
                        horizontal: 16, vertical: 10),
                  ),
                  onPressed: () =>
                      _showCreateEnvDialog(context, ref),
                  icon: const Icon(LucideIcons.plus, size: 16),
                  label: const Text('New environment'),
                ),
              ],
            ),
          ),
          const SizedBox(height: 8),
          Container(height: 1, color: colors.border),
          Expanded(
            child: Row(
              children: [
                // Left: environment list
                SizedBox(
                  width: 220,
                  child: _EnvList(
                    envsAsync: envsAsync,
                    selectedId: selectedId,
                    onSelect: (id) => context.go(withQuery(context, {'envId': id, 'tab': null})),
                    onDelete: (id) => _deleteEnv(context, ref, id),
                  ),
                ),
                Container(
                  width: 1, color: colors.border),
                // Right: detail view
                Expanded(
                  child: selectedId != null
                      ? _EnvDetail(envId: selectedId)
                      : const _EnvEmptyState(),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }

  void _showCreateEnvDialog(BuildContext context, WidgetRef ref) {
    final nameCtrl = TextEditingController();
    final slugCtrl = TextEditingController();
    final branchCtrl = TextEditingController();
    showAppDialog(
      context: context,
      title: 'New environment',
      content: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          AppDialogField(
              controller: nameCtrl,
              label: 'Name',
              hint: 'e.g. Staging',
              autofocus: true),
          const SizedBox(height: 12),
          AppDialogField(
              controller: slugCtrl,
              label: 'Slug',
              hint: 'e.g. staging'),
          const SizedBox(height: 12),
          AppDialogField(
              controller: branchCtrl,
              label: 'Branch',
              hint: 'e.g. main (optional)'),
        ],
      ),
      actions: [
        const AppDialogCancel(),
        AppDialogAction(
          label: 'Create',
          onTap: () async {
            final api = ref.read(apiClientProvider);
            await api.post('/deploy/environments', data: {
              'name': nameCtrl.text.trim(),
              'slug': slugCtrl.text.trim().isNotEmpty
                  ? slugCtrl.text.trim()
                  : nameCtrl.text.trim().toLowerCase().replaceAll(' ', '-'),
              'branch': branchCtrl.text.trim(),
            });
            if (context.mounted) {
              Navigator.of(context, rootNavigator: true).pop();
            }
            ref.invalidate(environmentsProvider);
          },
        ),
      ],
    );
  }

  Future<void> _deleteEnv(
      BuildContext context, WidgetRef ref, String id) async {
    final colors = consoleColors(context);
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        backgroundColor: colors.surface,
        title: Text('Delete environment',
          style: TextStyle(color: colors.textPrimary)),
        content: Text(
            'This will permanently remove the environment and its variables.',
          style: TextStyle(color: colors.textSecondary)),
        actions: [
          TextButton(
              onPressed: () => Navigator.pop(ctx, false),
              child: const Text('Cancel')),
          FilledButton(
            style: FilledButton.styleFrom(backgroundColor: Colors.red),
            onPressed: () => Navigator.pop(ctx, true),
            child: const Text('Delete'),
          ),
        ],
      ),
    );
    if (confirmed != true || !context.mounted) return;
    final api = ref.read(apiClientProvider);
    await api.delete('/deploy/environments/$id');
    if (context.mounted) context.go(withQuery(context, {'envId': null, 'tab': null}));
    ref.invalidate(environmentsProvider);
  }
}

// --- Env list ----------------------------------------------------------------

class _EnvList extends StatelessWidget {
  final AsyncValue<List<Map<String, dynamic>>> envsAsync;
  final String? selectedId;
  final ValueChanged<String> onSelect;
  final ValueChanged<String> onDelete;

  const _EnvList({
    required this.envsAsync,
    required this.selectedId,
    required this.onSelect,
    required this.onDelete,
  });

  @override
  Widget build(BuildContext context) {
    final colors = consoleColors(context);
    return envsAsync.when(
      loading: () => const Center(child: CircularProgressIndicator()),
      error: (e, _) => AppErrorState(error: e),
      data: (envs) {
        if (envs.isEmpty) {
          return Center(
            child: Padding(
              padding: const EdgeInsets.all(24),
              child: Text('No environments yet',
                  style: TextStyle(color: colors.textSecondary, fontSize: 13),
                  textAlign: TextAlign.center),
            ),
          );
        }
        return ListView.builder(
          padding: const EdgeInsets.symmetric(vertical: 8),
          itemCount: envs.length,
          itemBuilder: (context, i) {
            final env = envs[i];
            final id = env['\$id'] as String;
            final name =
                env['name'] as String? ?? env['slug'] as String? ?? id;
            final slug = env['slug'] as String? ?? '';
            final isDefault = env['isDefault'] == true;
            final selected = selectedId == id;
            final color = _envColor(slug);

            return ListTile(
              dense: true,
              onTap: () => onSelect(id),
              selected: selected,
              selectedTileColor: _accent.withOpacity(0.08),
              leading: Container(
                width: 8,
                height: 8,
                margin: const EdgeInsets.only(top: 4),
                decoration: BoxDecoration(
                    color: color, shape: BoxShape.circle),
              ),
              title: Text(name,
                  style: TextStyle(
                    color: selected ? colors.textPrimary : colors.textSecondary,
                      fontSize: 13,
                      fontWeight: selected
                          ? FontWeight.w500
                          : FontWeight.w400)),
              subtitle: isDefault
                  ? Text('default',
                    style: TextStyle(color: colors.textSubtle, fontSize: 11))
                  : null,
              trailing: IconButton(
                icon: const Icon(LucideIcons.trash2, size: 14),
                color: colors.textSubtle,
                tooltip: 'Delete',
                onPressed: () => onDelete(id),
              ),
            );
          },
        );
      },
    );
  }

  Color _envColor(String slug) {
    if (slug == 'production' || slug == 'prod') return _green;
    if (slug == 'staging') return _amber;
    if (slug == 'development' || slug == 'dev') return _purple;
    return const Color(0xFF9AA0B4);
  }
}

// --- Env detail --------------------------------------------------------------

final _envDetailProvider =
    FutureProvider.family<Map<String, dynamic>, String>((ref, id) async {
  final api = ref.read(apiClientProvider);
  final res = await api.get('/deploy/environments/$id');
  return res.data as Map<String, dynamic>;
});

class _EnvDetail extends ConsumerWidget {
  final String envId;
  const _EnvDetail({required this.envId});

  @override
  static const _tabNames = ['overview', 'variables', 'settings'];

  Widget build(BuildContext context, WidgetRef ref) {
    final colors = consoleColors(context);
    final tabName = tabFromQuery(context, defaultTab: 'overview');
    final tab = _tabNames.indexOf(tabName).clamp(0, _tabNames.length - 1);
    final envAsync = ref.watch(_envDetailProvider(envId));

    final name = envAsync.valueOrNull?['name'] as String? ?? '…';

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Padding(
          padding: const EdgeInsets.fromLTRB(24, 16, 24, 0),
          child: Text(name,
              style: Theme.of(context)
                  .textTheme
                  .titleLarge
                ?.copyWith(color: colors.textPrimary)),
        ),
        PageTabs(
          tabs: const ['Overview', 'Variables', 'Settings'],
          selected: tab,
          onChanged: (i) => context.go(withQuery(context, {'tab': _tabNames[i]})),
        ),
        Container(height: 1, color: colors.border),
        Expanded(
          child: envAsync.when(
            loading: () =>
                const Center(child: CircularProgressIndicator()),
            error: (e, _) => AppErrorState(error: e),
            data: (env) {
              switch (tab) {
                case 0:
                  return _EnvOverviewTab(env: env);
                case 1:
                  return _EnvVariablesTab(envId: envId, env: env);
                case 2:
                  return _EnvSettingsTab(envId: envId, env: env);
                default:
                  return const SizedBox.shrink();
              }
            },
          ),
        ),
      ],
    );
  }
}

// --- Overview tab ------------------------------------------------------------

class _EnvOverviewTab extends StatelessWidget {
  final Map<String, dynamic> env;
  const _EnvOverviewTab({required this.env});

  @override
  Widget build(BuildContext context) {
    final slug = env['slug'] as String? ?? '';
    final branch = env['branch'] as String? ?? '';
    final domain = env['domain'] as String? ?? '';
    final vars = (env['envVars'] as Map?)?.length ?? 0;

    return SingleChildScrollView(
      padding: const EdgeInsets.all(24),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(children: [
            _InfoCard(
                icon: LucideIcons.tag,
                label: 'Slug',
                value: slug.isNotEmpty ? slug : '—'),
            const SizedBox(width: 12),
            _InfoCard(
                icon: LucideIcons.gitBranch,
                label: 'Branch',
                value: branch.isNotEmpty ? branch : 'all branches'),
            const SizedBox(width: 12),
            _InfoCard(
                icon: LucideIcons.globe,
                label: 'Domain',
                value: domain.isNotEmpty ? domain : '—'),
            const SizedBox(width: 12),
            _InfoCard(
                icon: LucideIcons.keyRound,
                label: 'Variables',
                value: '$vars'),
          ]),
        ],
      ),
    );
  }
}

class _InfoCard extends StatelessWidget {
  final IconData icon;
  final String label;
  final String value;

  const _InfoCard(
      {required this.icon, required this.label, required this.value});

  @override
  Widget build(BuildContext context) {
    final colors = consoleColors(context);
    return Expanded(
      child: Container(
        padding: const EdgeInsets.all(16),
        decoration: BoxDecoration(
          color: colors.surface,
          borderRadius: BorderRadius.circular(8),
          border: Border.all(color: colors.border),
        ),
        child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
          Icon(icon, size: 16, color: colors.textSecondary),
          const SizedBox(height: 8),
          Text(value,
              style: TextStyle(
                  color: colors.textPrimary,
                  fontSize: 15,
                  fontWeight: FontWeight.w600)),
          const SizedBox(height: 2),
          Text(label,
              style: TextStyle(color: colors.textSubtle, fontSize: 11)),
        ]),
      ),
    );
  }
}

// --- Variables tab -----------------------------------------------------------

class _EnvVariablesTab extends ConsumerStatefulWidget {
  final String envId;
  final Map<String, dynamic> env;
  const _EnvVariablesTab({required this.envId, required this.env});

  @override
  ConsumerState<_EnvVariablesTab> createState() => _EnvVariablesTabState();
}

class _EnvVariablesTabState extends ConsumerState<_EnvVariablesTab> {
  late Map<String, String> _vars;
  bool _saving = false;
  bool _dirty = false;

  @override
  void initState() {
    super.initState();
    final raw = widget.env['envVars'];
    _vars = raw is Map
        ? Map<String, String>.from(raw.map(
            (k, v) => MapEntry(k.toString(), v.toString())))
        : {};
  }

  Future<void> _save() async {
    setState(() => _saving = true);
    try {
      final api = ref.read(apiClientProvider);
      await api.put('/deploy/environments/${widget.envId}', data: {
        'name': widget.env['name'],
        'branch': widget.env['branch'] ?? '',
        'domain': widget.env['domain'] ?? '',
        'envVars': _vars,
      });
      ref.invalidate(_envDetailProvider(widget.envId));
      if (mounted) setState(() => _dirty = false);
    } catch (_) {
    } finally {
      if (mounted) setState(() => _saving = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final colors = consoleColors(context);
    return Column(
      children: [
        Padding(
          padding: const EdgeInsets.fromLTRB(24, 16, 24, 8),
          child: Row(
            children: [
              const Text('Environment variables',
                  style: TextStyle(
                    color: Colors.white,
                      fontSize: 13,
                      fontWeight: FontWeight.w500)),
              const Spacer(),
              if (_dirty)
                FilledButton(
                  style: FilledButton.styleFrom(
                    backgroundColor: _accent,
                    shape: RoundedRectangleBorder(
                        borderRadius: BorderRadius.circular(8)),
                    padding: const EdgeInsets.symmetric(
                        horizontal: 16, vertical: 8),
                  ),
                  onPressed: _saving ? null : _save,
                  child: _saving
                      ? const SizedBox(
                          width: 14,
                          height: 14,
                          child: CircularProgressIndicator(
                              strokeWidth: 2, color: Colors.white))
                      : const Text('Save'),
                ),
              const SizedBox(width: 8),
              OutlinedButton.icon(
                style: OutlinedButton.styleFrom(
                  foregroundColor: colors.textSecondary,
                  side: BorderSide(color: colors.border),
                  shape: RoundedRectangleBorder(
                      borderRadius: BorderRadius.circular(8)),
                  padding: const EdgeInsets.symmetric(
                      horizontal: 12, vertical: 8),
                ),
                onPressed: _addVar,
                icon: const Icon(LucideIcons.plus, size: 14),
                label: const Text('Add variable',
                    style: TextStyle(fontSize: 13)),
              ),
            ],
          ),
        ),
        Container(height: 1, color: colors.border),
        Expanded(
          child: _vars.isEmpty
              ? Center(
                  child: Column(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                        Icon(LucideIcons.keyRound,
                            size: 40, color: colors.textSubtle),
                      const SizedBox(height: 12),
                        Text('No variables yet',
                          style: TextStyle(color: colors.textSecondary, fontSize: 13)),
                      const SizedBox(height: 8),
                      OutlinedButton.icon(
                        style: OutlinedButton.styleFrom(
                          foregroundColor: colors.textSecondary,
                          side: BorderSide(color: colors.border),
                          shape: RoundedRectangleBorder(
                              borderRadius: BorderRadius.circular(8)),
                        ),
                        onPressed: _addVar,
                        icon: const Icon(LucideIcons.plus, size: 14),
                        label: const Text('Add variable'),
                      ),
                    ],
                  ),
                )
              : ListView(
                  padding: const EdgeInsets.all(24),
                  children: _vars.entries.map((e) {
                    return _VarRow(
                      varKey: e.key,
                      varValue: e.value,
                      onDelete: () {
                        setState(() {
                          _vars.remove(e.key);
                          _dirty = true;
                        });
                      },
                      onChange: (k, v) {
                        setState(() {
                          _vars.remove(e.key);
                          _vars[k] = v;
                          _dirty = true;
                        });
                      },
                    );
                  }).toList(),
                ),
        ),
      ],
    );
  }

  void _addVar() {
    final keyCtrl = TextEditingController();
    final valCtrl = TextEditingController();
    showAppDialog(
      context: context,
      title: 'Add variable',
      content: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          AppDialogField(
              controller: keyCtrl,
              label: 'Key',
              hint: 'VARIABLE_NAME',
              autofocus: true),
          const SizedBox(height: 12),
          AppDialogField(
              controller: valCtrl, label: 'Value', hint: 'value'),
        ],
      ),
      actions: [
        const AppDialogCancel(),
        AppDialogAction(
          label: 'Add',
          onTap: () async {
            final k = keyCtrl.text.trim();
            final v = valCtrl.text;
            if (k.isNotEmpty) {
              setState(() {
                _vars[k] = v;
                _dirty = true;
              });
            }
            if (context.mounted) {
              Navigator.of(context, rootNavigator: true).pop();
            }
          },
        ),
      ],
    );
  }
}

class _VarRow extends StatefulWidget {
  final String varKey;
  final String varValue;
  final VoidCallback onDelete;
  final void Function(String key, String value) onChange;

  const _VarRow({
    required this.varKey,
    required this.varValue,
    required this.onDelete,
    required this.onChange,
  });

  @override
  State<_VarRow> createState() => _VarRowState();
}

class _VarRowState extends State<_VarRow> {
  bool _obscure = true;

  @override
  Widget build(BuildContext context) {
    final colors = consoleColors(context);
    return Container(
      margin: const EdgeInsets.only(bottom: 8),
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
      decoration: BoxDecoration(
        color: colors.surface,
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: colors.border),
      ),
      child: Row(
        children: [
          SizedBox(
            width: 200,
            child: Text(widget.varKey,
              style: TextStyle(
                    fontFamily: 'monospace',
                    fontSize: 12,
                color: colors.textPrimary)),
          ),
          Container(
              width: 1, height: 20, color: colors.border),
          const SizedBox(width: 12),
          Expanded(
            child: Text(
              _obscure
                  ? '•' * widget.varValue.length.clamp(8, 24)
                  : widget.varValue,
              style: TextStyle(
                  fontFamily: 'monospace',
                  fontSize: 12,
                  color: _obscure ? colors.textSubtle : colors.textSecondary),
              overflow: TextOverflow.ellipsis,
            ),
          ),
          IconButton(
            icon: Icon(
                _obscure ? LucideIcons.eye : LucideIcons.eyeOff,
                size: 14),
            color: colors.textSecondary,
            tooltip: _obscure ? 'Show' : 'Hide',
            onPressed: () => setState(() => _obscure = !_obscure),
          ),
          IconButton(
            icon: const Icon(LucideIcons.trash2, size: 14),
            color: colors.textSecondary,
            tooltip: 'Delete',
            onPressed: widget.onDelete,
          ),
        ],
      ),
    );
  }
}

// --- Settings tab ------------------------------------------------------------

class _EnvSettingsTab extends ConsumerStatefulWidget {
  final String envId;
  final Map<String, dynamic> env;
  const _EnvSettingsTab({required this.envId, required this.env});

  @override
  ConsumerState<_EnvSettingsTab> createState() => _EnvSettingsTabState();
}

class _EnvSettingsTabState extends ConsumerState<_EnvSettingsTab> {
  late TextEditingController _nameCtrl;
  late TextEditingController _branchCtrl;
  late TextEditingController _domainCtrl;
  bool _saving = false;

  @override
  void initState() {
    super.initState();
    _nameCtrl =
        TextEditingController(text: widget.env['name'] as String? ?? '');
    _branchCtrl =
        TextEditingController(text: widget.env['branch'] as String? ?? '');
    _domainCtrl =
        TextEditingController(text: widget.env['domain'] as String? ?? '');
  }

  @override
  void dispose() {
    _nameCtrl.dispose();
    _branchCtrl.dispose();
    _domainCtrl.dispose();
    super.dispose();
  }

  Future<void> _save() async {
    setState(() => _saving = true);
    try {
      final api = ref.read(apiClientProvider);
      await api.put('/deploy/environments/${widget.envId}', data: {
        'name': _nameCtrl.text.trim(),
        'branch': _branchCtrl.text.trim(),
        'domain': _domainCtrl.text.trim(),
        'envVars': widget.env['envVars'] ?? {},
      });
      ref.invalidate(_envDetailProvider(widget.envId));
      ref.invalidate(environmentsProvider);
    } catch (_) {
    } finally {
      if (mounted) setState(() => _saving = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    return SingleChildScrollView(
      padding: const EdgeInsets.all(24),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          _section('General', [
            _field('Name', _nameCtrl, 'Environment name'),
            const SizedBox(height: 12),
            _field('Branch', _branchCtrl, 'e.g. main (optional)'),
            const SizedBox(height: 12),
            _field('Domain', _domainCtrl, 'e.g. staging.example.com'),
          ]),
          const SizedBox(height: 24),
          FilledButton(
            style: FilledButton.styleFrom(
              backgroundColor: _accent,
              shape: RoundedRectangleBorder(
                  borderRadius: BorderRadius.circular(8)),
              padding: const EdgeInsets.symmetric(
                  horizontal: 20, vertical: 10),
            ),
            onPressed: _saving ? null : _save,
            child: _saving
                ? const SizedBox(
                    width: 14,
                    height: 14,
                    child: CircularProgressIndicator(
                        strokeWidth: 2, color: Colors.white))
                : const Text('Save changes'),
          ),
        ],
      ),
    );
  }

  Widget _section(String title, List<Widget> children) {
    final colors = consoleColors(context);
    return Container(
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: colors.surface,
        borderRadius: BorderRadius.circular(10),
        border: Border.all(color: colors.border),
      ),
      child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
        Text(title,
            style: TextStyle(
                color: colors.textPrimary,
                fontSize: 13,
                fontWeight: FontWeight.w600)),
        const SizedBox(height: 16),
        ...children,
      ]),
    );
  }

  Widget _field(String label, TextEditingController ctrl, String hint) {
    final colors = consoleColors(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(label,
            style: TextStyle(color: colors.textSecondary, fontSize: 12)),
        const SizedBox(height: 6),
        TextField(
          controller: ctrl,
          style: TextStyle(color: colors.textPrimary, fontSize: 13),
          decoration: InputDecoration(
            hintText: hint,
            hintStyle: TextStyle(color: colors.textSubtle, fontSize: 13),
            filled: true,
            fillColor: colors.fieldFill,
            isDense: true,
            contentPadding: const EdgeInsets.symmetric(
                horizontal: 12, vertical: 10),
            enabledBorder: OutlineInputBorder(
              borderRadius: BorderRadius.circular(8),
              borderSide: BorderSide(color: colors.fieldBorder),
            ),
            focusedBorder: OutlineInputBorder(
              borderRadius: BorderRadius.circular(8),
              borderSide: const BorderSide(color: _accent),
            ),
          ),
        ),
      ],
    );
  }
}

// --- Empty state -------------------------------------------------------------

class _EnvEmptyState extends StatelessWidget {
  const _EnvEmptyState();

  @override
  Widget build(BuildContext context) {
    final colors = consoleColors(context);
    return Center(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Container(
            padding: const EdgeInsets.all(20),
            decoration: BoxDecoration(
              color: colors.surface,
              borderRadius: BorderRadius.circular(16),
              border: Border.all(color: colors.border),
            ),
            child: Icon(LucideIcons.layers,
                size: 32, color: colors.textSecondary),
          ),
          const SizedBox(height: 16),
          Text('Select an environment',
              style: TextStyle(
                  color: colors.textPrimary,
                  fontSize: 15,
                  fontWeight: FontWeight.w500)),
          const SizedBox(height: 6),
          Text('Choose from the left or create a new one.',
              style: TextStyle(color: colors.textSecondary, fontSize: 13)),
        ],
      ),
    );
  }
}
