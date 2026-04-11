import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:lucide_icons/lucide_icons.dart';
import '../../core/api/client.dart';
import '../../core/theme/console_colors.dart';
import '../../core/utils/url_utils.dart';
import '../../core/widgets/app_data_table.dart';
import '../../core/widgets/app_dialog.dart';
import '../../core/widgets/page_tabs.dart';

// ── Colors ──────────────────────────────────────────────────────────────────

const _accent = Color(0xFF3472A4);

// ── Providers ────────────────────────────────────────────────────────────────

final _vaultSearchProvider = StateProvider<String>((ref) => '');
final _vaultPerPageProvider = StateProvider<int>((ref) => 25);
final _vaultPageProvider = StateProvider<int>((ref) => 1);

final vaultProvider = FutureProvider<Map<String, dynamic>>((ref) async {
  final api = ref.read(apiClientProvider);
  final limit = ref.watch(_vaultPerPageProvider);
  final page = ref.watch(_vaultPageProvider);
  final offset = (page - 1) * limit;
  final res = await api.get('/credentials', params: {'limit': limit, 'offset': offset});
  return res.data as Map<String, dynamic>;
});

final _accessLogProvider =
    FutureProvider.family<Map<String, dynamic>, String>((ref, credId) async {
  final api = ref.read(apiClientProvider);
  final res = await api.get('/credentials/$credId/accesses', params: {'limit': 50});
  return res.data as Map<String, dynamic>;
});

// ── Credential types ─────────────────────────────────────────────────────────

const _credTypes = [
  'generic',
  'api_key',
  'database',
  'ssh',
  'webhook',
  'tls',
  'oauth2',
];

IconData _typeIcon(String type) => switch (type) {
      'api_key'  => LucideIcons.key,
      'database' => LucideIcons.database,
      'ssh'      => LucideIcons.terminal,
      'webhook'  => LucideIcons.webhook,
      'tls'      => LucideIcons.lock,
      'oauth2'   => LucideIcons.fingerprint,
      _          => LucideIcons.shieldCheck,
    };

Color _typeBadgeColor(String type) => switch (type) {
      'api_key'  => const Color(0xFF6C47FF),
      'database' => const Color(0xFF0EA5E9),
      'ssh'      => const Color(0xFF10B981),
      'webhook'  => const Color(0xFFF59E0B),
      'tls'      => const Color(0xFFEF4444),
      'oauth2'   => const Color(0xFF8B5CF6),
      _          => const Color(0xFF6B7280),
    };

// ── Page ─────────────────────────────────────────────────────────────────────

class VaultPage extends ConsumerStatefulWidget {
  const VaultPage({super.key});

  @override
  ConsumerState<VaultPage> createState() => _VaultPageState();
}

class _VaultPageState extends ConsumerState<VaultPage> {
  final _searchCtrl = TextEditingController();
  late ConsoleColors _cs;

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    final urlPage = pageFromQuery(context);
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted) return;
      if (ref.read(_vaultPageProvider) != urlPage) {
        ref.read(_vaultPageProvider.notifier).state = urlPage;
      }
    });
  }

  @override
  void dispose() {
    _searchCtrl.dispose();
    super.dispose();
  }

  void _doSearch() {
    ref.read(_vaultSearchProvider.notifier).state = _searchCtrl.text.trim();
    ref.read(_vaultPageProvider.notifier).state = 1;
  }

  Future<void> _deleteCredential(Map<String, dynamic> cred) async {
    final name = cred['name'] as String? ?? '';
    final ok = await showDialog<bool>(
      context: context,
      builder: (_) => AppDialog(
        title: 'Delete credential',
        content: Text('Delete "$name"? This cannot be undone.'),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(context).pop(false),
            child: const Text('Cancel'),
          ),
          ElevatedButton(
            style: ElevatedButton.styleFrom(backgroundColor: Colors.red),
            onPressed: () => Navigator.of(context).pop(true),
            child: const Text('Delete'),
          ),
        ],
      ),
    );
    if (ok != true) return;
    final api = ref.read(apiClientProvider);
    await api.delete('/credentials/${cred[r'$id']}');
    ref.invalidate(vaultProvider);
  }

  @override
  Widget build(BuildContext context) {
    _cs = consoleColors(context);
    final creds = ref.watch(vaultProvider);
    final search = ref.watch(_vaultSearchProvider);
    final limit = ref.watch(_vaultPerPageProvider);
    final page = ref.watch(_vaultPageProvider);

    return Scaffold(
      backgroundColor: _cs.background,
      body: Padding(
        padding: EdgeInsets.symmetric(
          horizontal: MediaQuery.of(context).size.width > 1400 ? 80 : 40,
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const SizedBox(height: 32),
            Row(
              children: [
                Icon(LucideIcons.shieldCheck, size: 20, color: _accent),
                const SizedBox(width: 10),
                Text(
                  'Vault',
                  style: TextStyle(
                    color: _cs.textPrimary,
                    fontSize: 22,
                    fontWeight: FontWeight.w600,
                  ),
                ),
              ],
            ),
            const SizedBox(height: 24),
            Expanded(
              child: creds.when(
                loading: () => const Center(
                    child: CircularProgressIndicator(strokeWidth: 2)),
                error: (e, _) => Center(
                  child: Text('Error: $e',
                      style: const TextStyle(color: Color(0xFFEF4444))),
                ),
                data: (data) {
                  final items = (data['credentials'] as List? ?? [])
                      .cast<Map<String, dynamic>>();
                  final total = data['total'] as int? ?? 0;

                  // Client-side search filter
                  final filtered = search.isEmpty
                      ? items
                      : items
                          .where((c) =>
                              (c['name'] as String? ?? '')
                                  .toLowerCase()
                                  .contains(search.toLowerCase()) ||
                              (c['type'] as String? ?? '')
                                  .toLowerCase()
                                  .contains(search.toLowerCase()))
                          .toList();

                  return AppDataTable(
                    columns: const [
                      AppTableColumn(
                          key: r'$id',
                          label: 'ID',
                          flex: 3,
                          defaultVisible: false),
                      AppTableColumn(key: 'name',        label: 'Name',        flex: 3),
                      AppTableColumn(key: 'type',        label: 'Type',        flex: 2, sortable: false),
                      AppTableColumn(key: 'description', label: 'Description', flex: 3),
                      AppTableColumn(key: 'expiresAt',   label: 'Expires',     flex: 2, sortable: false),
                    ],
                    rows: filtered,
                    getCellValue: (row, key) => switch (key) {
                      r'$id'        => row[r'$id'] as String? ?? '',
                      'name'        => row['name'] as String? ?? '',
                      'type'        => row['type'] as String? ?? 'generic',
                      'description' => row['description'] as String? ?? '',
                      'expiresAt'   => row['expiresAt'] as String? ?? '',
                      _             => '',
                    },
                    cellBuilder: (row, key) {
                      if (key == 'type') {
                        return _TypeBadge(
                            type: row['type'] as String? ?? 'generic');
                      }
                      if (key == 'expiresAt') {
                        final exp = row['expiresAt'] as String?;
                        if (exp == null || exp.isEmpty) {
                          return const SizedBox.shrink();
                        }
                        return _ExpiryChip(expiresAt: exp, cs: _cs);
                      }
                      return null;
                    },
                    getRowIcon: (row) =>
                        _typeIcon(row['type'] as String? ?? 'generic'),
                    getRowIconColor: (row) =>
                        _typeBadgeColor(row['type'] as String? ?? 'generic'),
                    onRowTap: (row) => showDialog(
                      context: context,
                      builder: (_) => _CredentialDetailModal(
                        cred: row,
                        onSaved: () => ref.invalidate(vaultProvider),
                      ),
                    ),
                    onDeleteRow: _deleteCredential,
                    createWidget: Row(
                      children: [
                        _RotateKeysButton(cs: _cs),
                        const SizedBox(width: 8),
                        _CreateCredentialButton(cs: _cs),
                      ],
                    ),
                    total: total,
                    perPage: limit,
                    currentPage: page,
                    onPrev: () {
                      final p = page - 1;
                      ref.read(_vaultPageProvider.notifier).state = p;
                      context.go(withQuery(context, {'page': '$p'}));
                    },
                    onNext: () {
                      final p = page + 1;
                      ref.read(_vaultPageProvider.notifier).state = p;
                      context.go(withQuery(context, {'page': '$p'}));
                    },
                    onPerPageChanged: (pp) {
                      ref.read(_vaultPerPageProvider.notifier).state = pp;
                      ref.read(_vaultPageProvider.notifier).state = 1;
                    },
                    itemLabel: 'credentials',
                    searchController: _searchCtrl,
                    onSearch: _doSearch,
                    searchHint: 'Search by name or type',
                    emptyIcon: LucideIcons.shieldOff,
                    emptyTitle: 'No credentials yet',
                    emptySubtitle:
                        'Store API keys, database passwords, SSH keys and more.',
                    filters: const [
                      AppTableFilter(
                        key: 'type',
                        label: 'Type',
                        options: [
                          'generic',
                          'api_key',
                          'database',
                          'ssh',
                          'webhook',
                          'tls',
                          'oauth2',
                        ],
                      ),
                    ],
                  );
                },
              ),
            ),
          ],
        ),
      ),
    );
  }
}

// ── Rotate keys button ────────────────────────────────────────────────────────

class _RotateKeysButton extends ConsumerWidget {
  final ConsoleColors cs;
  const _RotateKeysButton({required this.cs});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return OutlinedButton.icon(
      onPressed: () => _confirm(context, ref),
      icon: Icon(LucideIcons.refreshCw, size: 14, color: cs.textSecondary),
      label: Text('Rotate keys', style: TextStyle(color: cs.textSecondary, fontSize: 13)),
      style: OutlinedButton.styleFrom(
        side: BorderSide(color: cs.border),
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(8)),
      ),
    );
  }

  Future<void> _confirm(BuildContext context, WidgetRef ref) async {
    final ok = await showDialog<bool>(
      context: context,
      builder: (_) => AppDialog(
        title: 'Rotate encryption keys',
        content: const Text(
          'All credentials will be re-encrypted with the current CREDENTIALS_ENCRYPTION_KEY. '
          'This is safe and non-destructive. Continue?',
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(context).pop(false),
            child: const Text('Cancel'),
          ),
          ElevatedButton(
            style: ElevatedButton.styleFrom(backgroundColor: _accent),
            onPressed: () => Navigator.of(context).pop(true),
            child: const Text('Rotate'),
          ),
        ],
      ),
    );
    if (ok != true) return;

    try {
      final api = ref.read(apiClientProvider);
      final res = await api.post('/credentials/rotate');
      final rotated = (res.data as Map<String, dynamic>)['rotated'] as int? ?? 0;
      if (context.mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('Rotated $rotated credential${rotated == 1 ? '' : 's'}')),
        );
        ref.invalidate(vaultProvider);
      }
    } catch (e) {
      if (context.mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('Rotation failed: $e'), backgroundColor: Colors.red),
        );
      }
    }
  }
}

// ── Create credential button ──────────────────────────────────────────────────

class _CreateCredentialButton extends ConsumerWidget {
  final ConsoleColors cs;
  const _CreateCredentialButton({required this.cs});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return ElevatedButton.icon(
      onPressed: () => _showModal(context, ref),
      icon: const Icon(LucideIcons.plus, size: 14, color: Colors.white),
      label: const Text('Add credential', style: TextStyle(color: Colors.white, fontSize: 13)),
      style: ElevatedButton.styleFrom(
        backgroundColor: _accent,
        padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 10),
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(8)),
        elevation: 0,
      ),
    );
  }

  void _showModal(BuildContext context, WidgetRef ref) {
    showDialog(
      context: context,
      builder: (_) => _CredentialModal(onSaved: () => ref.invalidate(vaultProvider)),
    );
  }
}

// ── Type badge ────────────────────────────────────────────────────────────────

class _TypeBadge extends StatelessWidget {
  final String type;
  const _TypeBadge({required this.type});

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
      decoration: BoxDecoration(
        color: _typeBadgeColor(type).withValues(alpha: 0.15),
        borderRadius: BorderRadius.circular(4),
      ),
      child: Text(
        type.replaceAll('_', ' '),
        style: TextStyle(
          color: _typeBadgeColor(type),
          fontSize: 10,
          fontWeight: FontWeight.w600,
          letterSpacing: 0.4,
        ),
      ),
    );
  }
}

// ── Expiry chip ───────────────────────────────────────────────────────────────

class _ExpiryChip extends StatelessWidget {
  final String expiresAt;
  final ConsoleColors cs;
  const _ExpiryChip({required this.expiresAt, required this.cs});

  @override
  Widget build(BuildContext context) {
    final dt = DateTime.tryParse(expiresAt);
    if (dt == null) return const SizedBox.shrink();
    final expired = dt.isBefore(DateTime.now());
    final days = dt.difference(DateTime.now()).inDays;
    final label = expired
        ? 'Expired'
        : days == 0
            ? 'Expires today'
            : 'Expires in ${days}d';
    final color = expired
        ? Colors.red
        : days < 7
            ? Colors.orange
            : cs.textMuted;
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        Icon(LucideIcons.clock, size: 11, color: color),
        const SizedBox(width: 4),
        Text(label, style: TextStyle(color: color, fontSize: 11)),
      ],
    );
  }
}

// ── Create/Edit modal ─────────────────────────────────────────────────────────

class _CredentialModal extends ConsumerStatefulWidget {
  final Map<String, dynamic>? existing;
  final VoidCallback onSaved;
  const _CredentialModal({this.existing, required this.onSaved});

  @override
  ConsumerState<_CredentialModal> createState() => _CredentialModalState();
}

class _CredentialModalState extends ConsumerState<_CredentialModal> {
  final _nameCtrl = TextEditingController();
  final _descCtrl = TextEditingController();
  final _dataCtrl = TextEditingController();
  String _type = 'generic';
  bool _protected = false;
  bool _hasExpiry = false;
  DateTime? _expiresAt;
  bool _loading = false;
  bool _obscure = true;

  bool get _isEdit => widget.existing != null;

  @override
  void initState() {
    super.initState();
    if (_isEdit) {
      final e = widget.existing!;
      _nameCtrl.text = e['name'] as String? ?? '';
      _descCtrl.text = e['description'] as String? ?? '';
      _type = e['type'] as String? ?? 'generic';
      _protected = e['protected'] as bool? ?? false;
      final exp = e['expiresAt'] as String?;
      if (exp != null) {
        _hasExpiry = true;
        _expiresAt = DateTime.tryParse(exp);
      }
    }
  }

  @override
  void dispose() {
    _nameCtrl.dispose();
    _descCtrl.dispose();
    _dataCtrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final cs = consoleColors(context);
    return AppDialog(
      title: _isEdit ? 'Edit credential' : 'New credential',
      content: SizedBox(
        width: 480,
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            _field('Name', _nameCtrl, hint: 'e.g. stripe-secret-key', cs: cs),
            const SizedBox(height: 14),
            _field('Description (optional)', _descCtrl, hint: 'What is this used for?', cs: cs),
            const SizedBox(height: 14),
            _typeDropdown(cs),
            const SizedBox(height: 14),
            _dataField(cs),
            const SizedBox(height: 14),
            _protectedToggle(cs),
            const SizedBox(height: 14),
            _expirySection(cs),
          ],
        ),
      ),
      actions: [
        TextButton(
          onPressed: _loading ? null : () => Navigator.of(context).pop(),
          child: const Text('Cancel'),
        ),
        ElevatedButton(
          style: ElevatedButton.styleFrom(
            backgroundColor: _accent,
            padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 10),
          ),
          onPressed: _loading ? null : _save,
          child: _loading
              ? const SizedBox(width: 16, height: 16, child: CircularProgressIndicator(strokeWidth: 2, color: Colors.white))
              : Text(_isEdit ? 'Save' : 'Create'),
        ),
      ],
    );
  }

  Widget _field(String label, TextEditingController ctrl,
      {String? hint, required ConsoleColors cs}) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(label, style: TextStyle(color: cs.textSecondary, fontSize: 12, fontWeight: FontWeight.w500)),
        const SizedBox(height: 6),
        TextField(
          controller: ctrl,
          style: TextStyle(color: cs.textPrimary, fontSize: 13),
          decoration: InputDecoration(
            hintText: hint,
            hintStyle: TextStyle(color: cs.textMuted),
            filled: true,
            fillColor: cs.surface,
            contentPadding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
            border: OutlineInputBorder(
              borderRadius: BorderRadius.circular(8),
              borderSide: BorderSide(color: cs.border),
            ),
            enabledBorder: OutlineInputBorder(
              borderRadius: BorderRadius.circular(8),
              borderSide: BorderSide(color: cs.border),
            ),
          ),
        ),
      ],
    );
  }

  Widget _typeDropdown(ConsoleColors cs) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text('Type', style: TextStyle(color: cs.textSecondary, fontSize: 12, fontWeight: FontWeight.w500)),
        const SizedBox(height: 6),
        DropdownButtonFormField<String>(
          initialValue: _type,
          dropdownColor: cs.surface,
          style: TextStyle(color: cs.textPrimary, fontSize: 13),
          decoration: InputDecoration(
            filled: true,
            fillColor: cs.surface,
            contentPadding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
            border: OutlineInputBorder(
              borderRadius: BorderRadius.circular(8),
              borderSide: BorderSide(color: cs.border),
            ),
            enabledBorder: OutlineInputBorder(
              borderRadius: BorderRadius.circular(8),
              borderSide: BorderSide(color: cs.border),
            ),
          ),
          items: _credTypes.map((t) => DropdownMenuItem(
            value: t,
            child: Row(
              children: [
                Icon(_typeIcon(t), size: 14, color: _typeBadgeColor(t)),
                const SizedBox(width: 8),
                Text(t.replaceAll('_', ' '), style: TextStyle(color: cs.textPrimary)),
              ],
            ),
          )).toList(),
          onChanged: (v) => setState(() => _type = v ?? _type),
        ),
      ],
    );
  }

  Widget _dataField(ConsoleColors cs) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text('Secret value', style: TextStyle(color: cs.textSecondary, fontSize: 12, fontWeight: FontWeight.w500)),
        const SizedBox(height: 6),
        TextField(
          controller: _dataCtrl,
          obscureText: _obscure,
          maxLines: _obscure ? 1 : 4,
          style: TextStyle(color: cs.textPrimary, fontSize: 13, fontFamily: 'monospace'),
          decoration: InputDecoration(
            hintText: _isEdit ? '(unchanged — leave blank to keep current)' : 'Paste the secret value',
            hintStyle: TextStyle(color: cs.textMuted),
            filled: true,
            fillColor: cs.surface,
            contentPadding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
            border: OutlineInputBorder(
              borderRadius: BorderRadius.circular(8),
              borderSide: BorderSide(color: cs.border),
            ),
            enabledBorder: OutlineInputBorder(
              borderRadius: BorderRadius.circular(8),
              borderSide: BorderSide(color: cs.border),
            ),
            suffixIcon: IconButton(
              icon: Icon(_obscure ? LucideIcons.eye : LucideIcons.eyeOff, size: 16, color: cs.textMuted),
              onPressed: () => setState(() => _obscure = !_obscure),
            ),
          ),
        ),
      ],
    );
  }

  Widget _protectedToggle(ConsoleColors cs) {
    return Row(
      children: [
        Switch(
          value: _protected,
          onChanged: (v) => setState(() => _protected = v),
          activeColor: _accent,
        ),
        const SizedBox(width: 8),
        Expanded(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text('Protected', style: TextStyle(color: cs.textPrimary, fontSize: 13)),
              Text(
                'Requires API key authentication to read. Client-side session tokens are blocked.',
                style: TextStyle(color: cs.textMuted, fontSize: 11),
              ),
            ],
          ),
        ),
      ],
    );
  }

  Widget _expirySection(ConsoleColors cs) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          children: [
            Checkbox(
              value: _hasExpiry,
              onChanged: (v) => setState(() {
                _hasExpiry = v ?? false;
                if (!_hasExpiry) _expiresAt = null;
              }),
              activeColor: _accent,
            ),
            Text('Set expiry date', style: TextStyle(color: cs.textPrimary, fontSize: 13)),
          ],
        ),
        if (_hasExpiry)
          Padding(
            padding: const EdgeInsets.only(left: 8, top: 4),
            child: OutlinedButton.icon(
              onPressed: () async {
                final dt = await showDatePicker(
                  context: context,
                  initialDate: _expiresAt ?? DateTime.now().add(const Duration(days: 30)),
                  firstDate: DateTime.now(),
                  lastDate: DateTime.now().add(const Duration(days: 365 * 5)),
                );
                if (dt != null) setState(() => _expiresAt = dt);
              },
              icon: Icon(LucideIcons.calendar, size: 14, color: cs.textSecondary),
              label: Text(
                _expiresAt == null
                    ? 'Pick date'
                    : '${_expiresAt!.year}-${_expiresAt!.month.toString().padLeft(2, '0')}-${_expiresAt!.day.toString().padLeft(2, '0')}',
                style: TextStyle(color: cs.textSecondary, fontSize: 13),
              ),
              style: OutlinedButton.styleFrom(
                side: BorderSide(color: cs.border),
                shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(8)),
              ),
            ),
          ),
      ],
    );
  }

  Future<void> _save() async {
    final name = _nameCtrl.text.trim();
    final data = _dataCtrl.text.trim();

    if (name.isEmpty) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('Name is required')),
      );
      return;
    }
    if (!_isEdit && data.isEmpty) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('Secret value is required')),
      );
      return;
    }

    setState(() => _loading = true);
    try {
      final api = ref.read(apiClientProvider);
      final body = <String, dynamic>{
        'name': name,
        'type': _type,
        'description': _descCtrl.text.trim(),
        'protected': _protected,
        if (data.isNotEmpty) 'data': data,
        if (_hasExpiry && _expiresAt != null) 'expiresAt': _expiresAt!.toUtc().toIso8601String(),
      };

      if (_isEdit) {
        if (data.isEmpty) {
          // Fetch current data to re-submit (update requires data)
          // Workaround: send a space as placeholder won't work. We require data for updates.
          // Show snackbar instead.
          ScaffoldMessenger.of(context).showSnackBar(
            const SnackBar(content: Text('Enter the secret value to update it')),
          );
          setState(() => _loading = false);
          return;
        }
        await api.put('/credentials/${widget.existing!['\$id']}', data: body);
      } else {
        await api.post('/credentials', data: body);
      }

      if (mounted) Navigator.of(context).pop();
      widget.onSaved();
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('Failed: $e'), backgroundColor: Colors.red),
        );
      }
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }
}

// ── Detail modal (view + access log) ─────────────────────────────────────────

class _CredentialDetailModal extends ConsumerStatefulWidget {
  final Map<String, dynamic> cred;
  final VoidCallback onSaved;
  const _CredentialDetailModal({required this.cred, required this.onSaved});

  @override
  ConsumerState<_CredentialDetailModal> createState() =>
      _CredentialDetailModalState();
}

class _CredentialDetailModalState
    extends ConsumerState<_CredentialDetailModal> {
  int _tab = 0;
  String? _revealedData;
  bool _revealing = false;

  @override
  Widget build(BuildContext context) {
    final cs = consoleColors(context);
    final credId = widget.cred['\$id'] as String? ?? '';

    return AppDialog(
      title: widget.cred['name'] as String? ?? 'Credential',
      content: SizedBox(
        width: 520,
        height: 380,
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            PageTabs(
              tabs: const ['Details', 'Access log'],
              selected: _tab,
              onChanged: (i) => setState(() => _tab = i),
            ),
            const SizedBox(height: 16),
            Expanded(
              child: _tab == 0
                  ? _detailsTab(cs)
                  : _accessLogTab(credId, cs),
            ),
          ],
        ),
      ),
      actions: [
        TextButton(
          onPressed: () => Navigator.of(context).pop(),
          child: const Text('Close'),
        ),
        ElevatedButton.icon(
          style: ElevatedButton.styleFrom(backgroundColor: _accent),
          onPressed: () {
            Navigator.of(context).pop();
            showDialog(
              context: context,
              builder: (_) => _CredentialModal(
                existing: widget.cred,
                onSaved: widget.onSaved,
              ),
            );
          },
          icon: const Icon(LucideIcons.pencil, size: 14),
          label: const Text('Edit'),
        ),
      ],
    );
  }

  Widget _detailsTab(ConsoleColors cs) {
    final c = widget.cred;
    final type = c['type'] as String? ?? 'generic';
    final protected_ = c['protected'] as bool? ?? false;
    final keyVersion = c['keyVersion'] as int? ?? 0;
    final expiresAt = c['expiresAt'] as String?;
    final createdAt = c['\$createdAt'] as String?;

    return SingleChildScrollView(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          _metaRow('Type', Row(children: [
            Icon(_typeIcon(type), size: 13, color: _typeBadgeColor(type)),
            const SizedBox(width: 6),
            _TypeBadge(type: type),
          ]), cs),
          _metaRow('Protected', Row(children: [
            Icon(
              protected_ ? LucideIcons.lock : LucideIcons.unlock,
              size: 13,
              color: protected_ ? Colors.orange : cs.textMuted,
            ),
            const SizedBox(width: 6),
            Text(protected_ ? 'Yes (API key required)' : 'No',
                style: TextStyle(color: cs.textPrimary, fontSize: 13)),
          ]), cs),
          _metaRow('Key version', Text('v$keyVersion', style: TextStyle(color: cs.textPrimary, fontSize: 13)), cs),
          if (expiresAt != null)
            _metaRow('Expires', _ExpiryChip(expiresAt: expiresAt, cs: cs), cs),
          if (createdAt != null)
            _metaRow('Created', Text(createdAt.substring(0, 10),
                style: TextStyle(color: cs.textPrimary, fontSize: 13)), cs),
          const SizedBox(height: 16),
          _revealSection(cs),
        ],
      ),
    );
  }

  Widget _metaRow(String label, Widget value, ConsoleColors cs) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 10),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          SizedBox(
            width: 110,
            child: Text(label,
                style: TextStyle(color: cs.textMuted, fontSize: 12)),
          ),
          Expanded(child: value),
        ],
      ),
    );
  }

  Widget _revealSection(ConsoleColors cs) {
    if (_revealedData != null) {
      return Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text('Secret value', style: TextStyle(color: cs.textMuted, fontSize: 12)),
          const SizedBox(height: 6),
          Container(
            width: double.infinity,
            padding: const EdgeInsets.all(12),
            decoration: BoxDecoration(
              color: cs.surface,
              borderRadius: BorderRadius.circular(8),
              border: Border.all(color: cs.border),
            ),
            child: Row(
              children: [
                Expanded(
                  child: SelectableText(
                    _revealedData!,
                    style: TextStyle(
                      color: cs.textPrimary,
                      fontSize: 12,
                      fontFamily: 'monospace',
                    ),
                  ),
                ),
                const SizedBox(width: 8),
                IconButton(
                  icon: Icon(LucideIcons.copy, size: 14, color: cs.textMuted),
                  tooltip: 'Copy',
                  onPressed: () {
                    Clipboard.setData(ClipboardData(text: _revealedData!));
                    ScaffoldMessenger.of(context).showSnackBar(
                      const SnackBar(content: Text('Copied to clipboard')),
                    );
                  },
                ),
              ],
            ),
          ),
        ],
      );
    }

    return OutlinedButton.icon(
      onPressed: _revealing ? null : _reveal,
      icon: _revealing
          ? const SizedBox(
              width: 14, height: 14,
              child: CircularProgressIndicator(strokeWidth: 2))
          : Icon(LucideIcons.eye, size: 14, color: cs.textSecondary),
      label: Text('Reveal secret value',
          style: TextStyle(color: cs.textSecondary, fontSize: 13)),
      style: OutlinedButton.styleFrom(
        side: BorderSide(color: cs.border),
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(8)),
      ),
    );
  }

  Future<void> _reveal() async {
    setState(() => _revealing = true);
    try {
      final api = ref.read(apiClientProvider);
      final credId = widget.cred['\$id'] as String? ?? '';
      final res = await api.get('/credentials/$credId');
      final data = (res.data as Map<String, dynamic>)['data'] as String? ?? '';
      setState(() {
        _revealedData = data;
        _revealing = false;
      });
    } catch (e) {
      setState(() => _revealing = false);
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('Could not reveal: $e'), backgroundColor: Colors.red),
        );
      }
    }
  }

  Widget _accessLogTab(String credId, ConsoleColors cs) {
    final logAsync = ref.watch(_accessLogProvider(credId));
    return logAsync.when(
      loading: () => const Center(child: CircularProgressIndicator()),
      error: (e, _) => Center(child: Text('Error: $e')),
      data: (data) {
        final accesses = (data['accesses'] as List? ?? [])
            .cast<Map<String, dynamic>>();
        if (accesses.isEmpty) {
          return Center(
            child: Text('No access events yet',
                style: TextStyle(color: cs.textMuted, fontSize: 13)),
          );
        }
        return ListView.separated(
          itemCount: accesses.length,
          separatorBuilder: (_, __) =>
              Divider(height: 1, color: cs.border.withValues(alpha: 0.4)),
          itemBuilder: (ctx, i) {
            final a = accesses[i];
            final action = a['action'] as String? ?? '';
            final ip = a['ip'] as String? ?? '';
            final at = a['accessedAt'] as String? ?? '';
            return Padding(
              padding: const EdgeInsets.symmetric(vertical: 8),
              child: Row(
                children: [
                  _ActionBadge(action: action),
                  const SizedBox(width: 10),
                  Expanded(
                    child: Text(
                      ip.isNotEmpty ? ip : 'unknown',
                      style: TextStyle(color: cs.textPrimary, fontSize: 12),
                    ),
                  ),
                  Text(
                    at.length >= 16 ? at.substring(0, 16).replaceFirst('T', ' ') : at,
                    style: TextStyle(color: cs.textMuted, fontSize: 11),
                  ),
                ],
              ),
            );
          },
        );
      },
    );
  }
}

// ── Action badge ──────────────────────────────────────────────────────────────

class _ActionBadge extends StatelessWidget {
  final String action;
  const _ActionBadge({required this.action});

  @override
  Widget build(BuildContext context) {
    final color = switch (action) {
      'create' => Colors.green,
      'read'   => const Color(0xFF3472A4),
      'update' => Colors.orange,
      'delete' => Colors.red,
      'rotate' => Colors.purple,
      _        => Colors.grey,
    };
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 7, vertical: 2),
      decoration: BoxDecoration(
        color: color.withValues(alpha: 0.15),
        borderRadius: BorderRadius.circular(4),
      ),
      child: Text(
        action,
        style: TextStyle(
          color: color,
          fontSize: 10,
          fontWeight: FontWeight.w600,
          letterSpacing: 0.4,
        ),
      ),
    );
  }
}
