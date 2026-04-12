import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:lucide_icons/lucide_icons.dart';
import '../../core/api/client.dart';
import '../../core/theme/console_colors.dart';
import '../../core/widgets/app_dialog.dart';
import '../../core/widgets/app_error_state.dart';

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

const _accent = Color(0xFF3472A4);
const _red = Color(0xFFEF4444);

const _kExpiryOptions = <String, String>{
  'never': 'Never',
  '1d': '1 Day',
  '7d': '7 Days',
  '30d': '30 Days',
  '90d': '90 Days',
  '1y': '1 Year',
};

const _kScopeGroups = <String, List<String>>{
  'Auth': ['auth.read', 'auth.write'],
  'Databases': ['databases.read', 'databases.write'],
  'Storage': ['storage.read', 'storage.write'],
  'Functions': [
    'functions.read',
    'functions.write',
    'functions.execute',
  ],
  'Messaging': ['messaging.read', 'messaging.write'],
  'Deploy': ['deploy.read', 'deploy.write'],
  'Workflows': [
    'workflows.read',
    'workflows.write',
    'workflows.execute',
  ],
};

// ---------------------------------------------------------------------------
// Provider
// ---------------------------------------------------------------------------

final _keyDetailProvider =
    FutureProvider.family<Map<String, dynamic>, ({String projectId, String keyId})>(
        (ref, args) async {
  final api = ref.read(apiClientProvider);
  final res =
      await api.get('/projects/${args.projectId}/keys/${args.keyId}');
  return res.data as Map<String, dynamic>;
});

// ---------------------------------------------------------------------------
// Page
// ---------------------------------------------------------------------------

class ApiKeyDetailPage extends ConsumerStatefulWidget {
  const ApiKeyDetailPage({super.key});

  @override
  ConsumerState<ApiKeyDetailPage> createState() => _ApiKeyDetailPageState();
}

class _ApiKeyDetailPageState extends ConsumerState<ApiKeyDetailPage> {
  String get _projectId =>
      GoRouterState.of(context).pathParameters['projectId']!;
  String get _keyId => GoRouterState.of(context).pathParameters['keyId']!;

  @override
  Widget build(BuildContext context) {
    final colors = consoleColors(context);
    final args = (projectId: _projectId, keyId: _keyId);
    final keyAsync = ref.watch(_keyDetailProvider(args));

    return keyAsync.when(
      loading: () => const Center(child: CircularProgressIndicator()),
      error: (e, _) => AppErrorState(error: e),
      data: (keyData) => _KeyDetailBody(
        projectId: _projectId,
        keyData: keyData,
        colors: colors,
        onSaved: () => ref.invalidate(_keyDetailProvider(args)),
        onDeleted: () => context.go('/project/$_projectId/settings?tab=api-keys'),
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Body
// ---------------------------------------------------------------------------

class _KeyDetailBody extends ConsumerStatefulWidget {
  final String projectId;
  final Map<String, dynamic> keyData;
  final ConsoleColors colors;
  final VoidCallback onSaved;
  final VoidCallback onDeleted;

  const _KeyDetailBody({
    required this.projectId,
    required this.keyData,
    required this.colors,
    required this.onSaved,
    required this.onDeleted,
  });

  @override
  ConsumerState<_KeyDetailBody> createState() => _KeyDetailBodyState();
}

class _KeyDetailBodyState extends ConsumerState<_KeyDetailBody> {
  // Name
  late final TextEditingController _nameCtrl;
  bool _nameDirty = false;
  bool _nameSaving = false;

  // Scopes
  late Set<String> _scopes;
  bool _scopesDirty = false;
  bool _scopesSaving = false;

  // Expiry
  late String _expiry;
  bool _expiryDirty = false;
  bool _expirySaving = false;

  ConsoleColors get colors => widget.colors;

  @override
  void initState() {
    super.initState();
    final d = widget.keyData;
    _nameCtrl = TextEditingController(text: d['name'] as String? ?? '');
    _nameCtrl.addListener(() => setState(() => _nameDirty = true));
    _scopes = Set<String>.from(
        (d['scopes'] as List?)?.cast<String>() ?? <String>[]);
    _expiry = _expiryKeyFromData(d['expire'] as String?);
  }

  @override
  void dispose() {
    _nameCtrl.dispose();
    super.dispose();
  }

  String _expiryKeyFromData(String? isoExpiry) {
    if (isoExpiry == null || isoExpiry.isEmpty) return 'never';
    return 'never'; // existing keys keep their date; dropdown is for editing
  }

  String? _expiresAtIso() {
    if (_expiry == 'never') return '';
    final now = DateTime.now().toUtc();
    final DateTime t;
    switch (_expiry) {
      case '1d':
        t = now.add(const Duration(days: 1));
        break;
      case '7d':
        t = now.add(const Duration(days: 7));
        break;
      case '30d':
        t = now.add(const Duration(days: 30));
        break;
      case '90d':
        t = now.add(const Duration(days: 90));
        break;
      case '1y':
        t = now.add(const Duration(days: 365));
        break;
      default:
        return '';
    }
    return t.toIso8601String();
  }

  String? _expiryPreview() {
    final iso = _expiresAtIso();
    if (iso == null || iso.isEmpty) return null;
    final t = DateTime.parse(iso).toLocal();
    final months = [
      'Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun',
      'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec',
    ];
    return 'Your key will expire on ${months[t.month - 1]} ${t.day}, ${t.year}';
  }

  Future<void> _saveName() async {
    if (_nameCtrl.text.trim().isEmpty) return;
    setState(() => _nameSaving = true);
    try {
      await ref.read(apiClientProvider).patch(
        '/projects/${widget.projectId}/keys/${widget.keyData['\$id']}',
        data: {'name': _nameCtrl.text.trim()},
      );
      setState(() => _nameDirty = false);
      widget.onSaved();
    } finally {
      if (mounted) setState(() => _nameSaving = false);
    }
  }

  Future<void> _saveScopes() async {
    setState(() => _scopesSaving = true);
    try {
      await ref.read(apiClientProvider).patch(
        '/projects/${widget.projectId}/keys/${widget.keyData['\$id']}',
        data: {'scopes': _scopes.toList()},
      );
      setState(() => _scopesDirty = false);
      widget.onSaved();
    } finally {
      if (mounted) setState(() => _scopesSaving = false);
    }
  }

  Future<void> _saveExpiry() async {
    setState(() => _expirySaving = true);
    try {
      await ref.read(apiClientProvider).patch(
        '/projects/${widget.projectId}/keys/${widget.keyData['\$id']}',
        data: {'expiresAt': _expiresAtIso() ?? ''},
      );
      setState(() => _expiryDirty = false);
      widget.onSaved();
    } finally {
      if (mounted) setState(() => _expirySaving = false);
    }
  }

  Future<void> _deleteKey() async {
    final confirmed = await showAppDialog<bool>(
      context: context,
      title: 'Delete API key',
      content: Text(
        'Any applications using this key will lose access immediately. This action is irreversible.',
        style: TextStyle(color: colors.textSecondary, fontSize: 13),
      ),
      actions: [
        const AppDialogCancel(),
        AppDialogAction(
          label: 'Delete',
          destructive: true,
          onTap: () => Navigator.of(context, rootNavigator: true).pop(true),
        ),
      ],
    );
    if (confirmed == true) {
      await ref.read(apiClientProvider).delete(
        '/projects/${widget.projectId}/keys/${widget.keyData['\$id']}',
      );
      widget.onDeleted();
    }
  }

  void _toggleScope(String scope) =>
      setState(() {
        _scopesDirty = true;
        if (_scopes.contains(scope)) {
          _scopes.remove(scope);
        } else {
          _scopes.add(scope);
        }
      });

  void _toggleGroup(String group) {
    final groupScopes = _kScopeGroups[group]!;
    final allSelected = groupScopes.every(_scopes.contains);
    setState(() {
      _scopesDirty = true;
      if (allSelected) {
        _scopes.removeAll(groupScopes);
      } else {
        _scopes.addAll(groupScopes);
      }
    });
  }

  @override
  Widget build(BuildContext context) {
    final d = widget.keyData;
    final name = d['name'] as String? ?? 'Unnamed';
    final keyId = d['\$id'] as String? ?? '';
    final secretPrefix = d['secretPrefix'] as String? ?? '';
    final expiresAt = d['expire'] as String?;
    final createdAt = d['\$createdAt'] as String?;

    String? expiryDisplay;
    if (expiresAt != null && expiresAt.isNotEmpty) {
      final t = DateTime.tryParse(expiresAt);
      if (t != null) {
        final l = t.toLocal();
        final months = [
          'Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun',
          'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec',
        ];
        expiryDisplay = '${months[l.month - 1]} ${l.day}, ${l.year}';
      }
    }

    return Scaffold(
      backgroundColor: colors.background,
      body: SingleChildScrollView(
        padding: EdgeInsets.symmetric(
          horizontal: _hPad(context),
          vertical: 32,
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            // ── Breadcrumb / back ────────────────────────────────────────
            GestureDetector(
              onTap: () => context.go(
                '/project/${widget.projectId}/settings?tab=api-keys',
              ),
              child: Row(
                mainAxisSize: MainAxisSize.min,
                children: [
                  Icon(LucideIcons.chevronLeft,
                      size: 14, color: colors.textSubtle),
                  const SizedBox(width: 4),
                  Text(name,
                      style: TextStyle(
                          color: colors.textPrimary,
                          fontSize: 16,
                          fontWeight: FontWeight.w600)),
                  const SizedBox(width: 10),
                  Container(
                    padding: const EdgeInsets.symmetric(
                        horizontal: 8, vertical: 3),
                    decoration: BoxDecoration(
                      color: _accent.withValues(alpha: 0.12),
                      borderRadius: BorderRadius.circular(4),
                    ),
                    child: const Text('API Secret',
                        style: TextStyle(
                            color: _accent,
                            fontSize: 11,
                            fontWeight: FontWeight.w500)),
                  ),
                ],
              ),
            ),
            const SizedBox(height: 32),

            // ── Key details ──────────────────────────────────────────────
            _SectionRow(
              label: 'Key details',
              colors: colors,
              child: _DetailsCard(
                keyId: keyId,
                secretPrefix: secretPrefix,
                name: name,
                createdAt: createdAt,
                expiryDisplay: expiryDisplay,
                colors: colors,
              ),
            ),
            const SizedBox(height: 24),

            // ── Name ─────────────────────────────────────────────────────
            _SectionRow(
              label: 'Name',
              description: 'Choose any name that will help you distinguish '
                  'between API keys.',
              colors: colors,
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  _field(
                    controller: _nameCtrl,
                    hint: 'Key name',
                  ),
                  const SizedBox(height: 12),
                  Align(
                    alignment: Alignment.centerRight,
                    child: _SaveButton(
                      onTap: _nameDirty ? _saveName : null,
                      loading: _nameSaving,
                    ),
                  ),
                ],
              ),
            ),
            const SizedBox(height: 24),

            // ── Scopes ───────────────────────────────────────────────────
            _SectionRow(
              label: 'Scopes',
              description: 'Choose which permission scopes to grant your '
                  'application. Only grant the permissions you need.',
              colors: colors,
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Row(
                    children: [
                      GestureDetector(
                        onTap: () => setState(() {
                          _scopesDirty = true;
                          for (final g in _kScopeGroups.values) {
                            _scopes.addAll(g);
                          }
                        }),
                        child: const Text('Select all',
                            style: TextStyle(
                                color: _accent, fontSize: 12)),
                      ),
                      const SizedBox(width: 14),
                      GestureDetector(
                        onTap: () => setState(() {
                          _scopesDirty = true;
                          _scopes.clear();
                        }),
                        child: Text('Deselect all',
                            style: TextStyle(
                                color: colors.textSubtle,
                                fontSize: 12)),
                      ),
                    ],
                  ),
                  const SizedBox(height: 10),
                  ..._kScopeGroups.entries.map((entry) =>
                      _ScopeGroupRow(
                        group: entry.key,
                        scopes: entry.value,
                        selected: _scopes,
                        onToggleGroup: () => _toggleGroup(entry.key),
                        onToggleScope: _toggleScope,
                        colors: colors,
                      )),
                  const SizedBox(height: 12),
                  Align(
                    alignment: Alignment.centerRight,
                    child: _SaveButton(
                      onTap: _scopesDirty ? _saveScopes : null,
                      loading: _scopesSaving,
                    ),
                  ),
                ],
              ),
            ),
            const SizedBox(height: 24),

            // ── Expiration date ──────────────────────────────────────────
            _SectionRow(
              label: 'Expiration date',
              description:
                  'Set a date after which your API key will expire.',
              colors: colors,
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  AppSelectField<String>(
                    label: 'Expiration date',
                    value: _expiry,
                    items: _kExpiryOptions.entries
                        .map((e) => DropdownMenuItem(
                              value: e.key,
                              child: Text(e.value),
                            ))
                        .toList(),
                    onChanged: (v) => setState(() {
                      _expiry = v ?? 'never';
                      _expiryDirty = true;
                    }),
                  ),
                  if (_expiryPreview() != null) ...[
                    const SizedBox(height: 6),
                    Row(
                      children: [
                        Icon(LucideIcons.info,
                            size: 12, color: colors.textSubtle),
                        const SizedBox(width: 5),
                        Text(_expiryPreview()!,
                            style: TextStyle(
                                color: colors.textSubtle,
                                fontSize: 12)),
                      ],
                    ),
                  ],
                  const SizedBox(height: 12),
                  Align(
                    alignment: Alignment.centerRight,
                    child: _SaveButton(
                      onTap: _expiryDirty ? _saveExpiry : null,
                      loading: _expirySaving,
                    ),
                  ),
                ],
              ),
            ),
            const SizedBox(height: 24),

            // ── Delete ───────────────────────────────────────────────────
            _SectionRow(
              label: 'Delete API key',
              description:
                  'The API key will be permanently deleted. '
                  'This action is irreversible.',
              colors: colors,
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  _InfoRow(
                    label: name,
                    sub: createdAt != null ? 'Created $createdAt' : null,
                    colors: colors,
                  ),
                  const SizedBox(height: 16),
                  Divider(color: colors.border),
                  const SizedBox(height: 12),
                  Align(
                    alignment: Alignment.centerRight,
                    child: OutlinedButton(
                      onPressed: _deleteKey,
                      style: OutlinedButton.styleFrom(
                        foregroundColor: _red,
                        side: const BorderSide(color: _red),
                        padding: const EdgeInsets.symmetric(
                            horizontal: 20, vertical: 10),
                        shape: RoundedRectangleBorder(
                            borderRadius: BorderRadius.circular(8)),
                      ),
                      child: const Text('Delete',
                          style: TextStyle(fontSize: 13)),
                    ),
                  ),
                ],
              ),
            ),
            const SizedBox(height: 48),
          ],
        ),
      ),
    );
  }

  Widget _field({
    required TextEditingController controller,
    required String hint,
  }) {
    final colors = widget.colors;
    return TextField(
      controller: controller,
      style: TextStyle(color: colors.textPrimary, fontSize: 13),
      decoration: InputDecoration(
        hintText: hint,
        hintStyle: TextStyle(color: colors.textSubtle, fontSize: 13),
        filled: true,
        fillColor: colors.fieldFill,
        isDense: true,
        contentPadding:
            const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
        border: OutlineInputBorder(
          borderRadius: BorderRadius.circular(8),
          borderSide: BorderSide(color: colors.fieldBorder),
        ),
        enabledBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(8),
          borderSide: BorderSide(color: colors.fieldBorder),
        ),
        focusedBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(8),
          borderSide: const BorderSide(color: _accent),
        ),
      ),
    );
  }
}

double _hPad(BuildContext ctx) {
  final w = MediaQuery.sizeOf(ctx).width;
  if (w > 1200) return (w - 900) / 2;
  if (w > 800) return 48.0;
  return 24.0;
}

// ---------------------------------------------------------------------------
// Sub-widgets
// ---------------------------------------------------------------------------

class _SectionRow extends StatelessWidget {
  final String label;
  final String? description;
  final Widget child;
  final ConsoleColors colors;

  const _SectionRow({
    required this.label,
    this.description,
    required this.child,
    required this.colors,
  });

  @override
  Widget build(BuildContext context) {
    return Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        // Left label column
        SizedBox(
          width: 220,
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(label,
                  style: TextStyle(
                      color: colors.textPrimary,
                      fontSize: 14,
                      fontWeight: FontWeight.w500)),
              if (description != null) ...[
                const SizedBox(height: 6),
                Text(description!,
                    style: TextStyle(
                        color: colors.textSubtle, fontSize: 12)),
              ],
            ],
          ),
        ),
        const SizedBox(width: 32),
        // Right content
        Expanded(
          child: Container(
            padding: const EdgeInsets.all(20),
            decoration: BoxDecoration(
              color: colors.surface,
              borderRadius: BorderRadius.circular(10),
              border: Border.all(color: colors.border),
            ),
            child: child,
          ),
        ),
      ],
    );
  }
}

class _DetailsCard extends StatelessWidget {
  final String keyId;
  final String secretPrefix;
  final String name;
  final String? createdAt;
  final String? expiryDisplay;
  final ConsoleColors colors;

  const _DetailsCard({
    required this.keyId,
    required this.secretPrefix,
    required this.name,
    required this.createdAt,
    required this.expiryDisplay,
    required this.colors,
  });

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          children: [
            Expanded(
              child: _MetaItem(
                label: 'Name',
                value: name,
                colors: colors,
              ),
            ),
            Expanded(
              child: _MetaItem(
                label: 'Created',
                value: createdAt ?? '—',
                colors: colors,
              ),
            ),
          ],
        ),
        const SizedBox(height: 16),
        Row(
          children: [
            Expanded(
              child: _MetaItem(
                label: 'Expiration date',
                value: expiryDisplay ?? 'Never',
                colors: colors,
              ),
            ),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text('Secret',
                      style: TextStyle(
                          color: colors.textSubtle,
                          fontSize: 11,
                          fontWeight: FontWeight.w500)),
                  const SizedBox(height: 4),
                  Row(
                    children: [
                      Text(
                        secretPrefix.isNotEmpty
                            ? '$secretPrefix···'
                            : '•' * 12,
                        style: TextStyle(
                            color: colors.textSecondary,
                            fontSize: 12,
                            fontFamily: 'monospace'),
                      ),
                      if (secretPrefix.isNotEmpty) ...[
                        const SizedBox(width: 6),
                        _CopyIconButton(text: secretPrefix),
                      ],
                    ],
                  ),
                  const SizedBox(height: 4),
                  Text('Full secret only shown once at creation',
                      style: TextStyle(
                          color: colors.textSubtle, fontSize: 10)),
                ],
              ),
            ),
          ],
        ),
      ],
    );
  }
}

class _MetaItem extends StatelessWidget {
  final String label;
  final String value;
  final ConsoleColors colors;

  const _MetaItem(
      {required this.label, required this.value, required this.colors});

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(label,
            style: TextStyle(
                color: colors.textSubtle,
                fontSize: 11,
                fontWeight: FontWeight.w500)),
        const SizedBox(height: 4),
        Text(value,
            style: TextStyle(color: colors.textPrimary, fontSize: 13)),
      ],
    );
  }
}

class _InfoRow extends StatelessWidget {
  final String label;
  final String? sub;
  final ConsoleColors colors;

  const _InfoRow(
      {required this.label, required this.colors, this.sub});

  @override
  Widget build(BuildContext context) {
    return Row(
      children: [
        Icon(LucideIcons.key, size: 14, color: colors.textSubtle),
        const SizedBox(width: 10),
        Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(label,
                style: TextStyle(
                    color: colors.textPrimary,
                    fontSize: 13,
                    fontWeight: FontWeight.w500)),
            if (sub != null)
              Text(sub!,
                  style:
                      TextStyle(color: colors.textSubtle, fontSize: 11)),
          ],
        ),
      ],
    );
  }
}

class _SaveButton extends StatelessWidget {
  final VoidCallback? onTap;
  final bool loading;

  const _SaveButton({this.onTap, this.loading = false});

  @override
  Widget build(BuildContext context) {
    return FilledButton(
      style: FilledButton.styleFrom(
        backgroundColor: _accent,
        disabledBackgroundColor:
            _accent.withValues(alpha: 0.4),
        padding:
            const EdgeInsets.symmetric(horizontal: 20, vertical: 10),
        shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(8)),
      ),
      onPressed: loading ? null : onTap,
      child: loading
          ? const SizedBox(
              width: 14,
              height: 14,
              child: CircularProgressIndicator(
                  strokeWidth: 2, color: Colors.white),
            )
          : const Text('Save', style: TextStyle(fontSize: 13)),
    );
  }
}

class _ScopeGroupRow extends StatefulWidget {
  final String group;
  final List<String> scopes;
  final Set<String> selected;
  final VoidCallback onToggleGroup;
  final void Function(String) onToggleScope;
  final ConsoleColors colors;

  const _ScopeGroupRow({
    required this.group,
    required this.scopes,
    required this.selected,
    required this.onToggleGroup,
    required this.onToggleScope,
    required this.colors,
  });

  @override
  State<_ScopeGroupRow> createState() => _ScopeGroupRowState();
}

class _ScopeGroupRowState extends State<_ScopeGroupRow> {
  bool _expanded = false;

  @override
  Widget build(BuildContext context) {
    final colors = widget.colors;
    final selectedCount =
        widget.scopes.where(widget.selected.contains).length;
    final allSelected = selectedCount == widget.scopes.length;

    return Container(
      margin: const EdgeInsets.only(bottom: 4),
      decoration: BoxDecoration(
        color: colors.fill,
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: colors.border),
      ),
      child: Column(
        children: [
          InkWell(
            onTap: () => setState(() => _expanded = !_expanded),
            borderRadius: BorderRadius.circular(8),
            child: Padding(
              padding: const EdgeInsets.symmetric(
                  horizontal: 12, vertical: 10),
              child: Row(
                children: [
                  Checkbox(
                    value: allSelected
                        ? true
                        : selectedCount > 0
                            ? null
                            : false,
                    tristate: true,
                    onChanged: (_) => widget.onToggleGroup(),
                    activeColor: _accent,
                    checkColor: Colors.white,
                    materialTapTargetSize:
                        MaterialTapTargetSize.shrinkWrap,
                    visualDensity: VisualDensity.compact,
                  ),
                  const SizedBox(width: 8),
                  Text(widget.group,
                      style: TextStyle(
                          color: colors.textPrimary,
                          fontSize: 13,
                          fontWeight: FontWeight.w500)),
                  const SizedBox(width: 8),
                  Text(
                    '$selectedCount ${selectedCount == 1 ? 'Scope' : 'Scopes'}',
                    style:
                        TextStyle(color: colors.textSubtle, fontSize: 12),
                  ),
                  const Spacer(),
                  Icon(
                    _expanded
                        ? LucideIcons.chevronUp
                        : LucideIcons.chevronDown,
                    size: 14,
                    color: colors.textSubtle,
                  ),
                ],
              ),
            ),
          ),
          if (_expanded) ...[
            Divider(height: 1, color: colors.border),
            ...widget.scopes.map((scope) => _ScopeTile(
                  scope: scope,
                  checked: widget.selected.contains(scope),
                  onChanged: () => widget.onToggleScope(scope),
                  colors: colors,
                )),
          ],
        ],
      ),
    );
  }
}

class _ScopeTile extends StatelessWidget {
  final String scope;
  final bool checked;
  final VoidCallback onChanged;
  final ConsoleColors colors;

  const _ScopeTile({
    required this.scope,
    required this.checked,
    required this.onChanged,
    required this.colors,
  });

  @override
  Widget build(BuildContext context) {
    return InkWell(
      onTap: onChanged,
      child: Padding(
        padding: const EdgeInsets.only(
            left: 40, right: 12, top: 8, bottom: 8),
        child: Row(
          children: [
            Checkbox(
              value: checked,
              onChanged: (_) => onChanged(),
              activeColor: _accent,
              checkColor: Colors.white,
              materialTapTargetSize: MaterialTapTargetSize.shrinkWrap,
              visualDensity: VisualDensity.compact,
            ),
            const SizedBox(width: 8),
            Text(scope,
                style: TextStyle(
                    color: colors.textSecondary,
                    fontSize: 12,
                    fontFamily: 'monospace')),
          ],
        ),
      ),
    );
  }
}

// Copy icon with green-tick feedback
class _CopyIconButton extends StatefulWidget {
  final String text;
  const _CopyIconButton({required this.text});

  @override
  State<_CopyIconButton> createState() => _CopyIconButtonState();
}

class _CopyIconButtonState extends State<_CopyIconButton> {
  bool _copied = false;

  Future<void> _copy() async {
    await Clipboard.setData(ClipboardData(text: widget.text));
    if (!mounted) return;
    setState(() => _copied = true);
    await Future.delayed(const Duration(seconds: 2));
    if (mounted) setState(() => _copied = false);
  }

  @override
  Widget build(BuildContext context) {
    return GestureDetector(
      onTap: _copied ? null : _copy,
      child: Icon(
        _copied ? LucideIcons.check : LucideIcons.copy,
        size: 13,
        color: _copied
            ? const Color(0xFF10B981)
            : consoleColors(context).textSubtle,
      ),
    );
  }
}
