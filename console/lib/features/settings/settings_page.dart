import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:lucide_icons/lucide_icons.dart';
import '../../core/api/client.dart';
import '../../core/providers/project_provider.dart';
import '../../core/theme/console_colors.dart';
import '../../core/widgets/app_data_table.dart';
import '../../core/widgets/app_dialog.dart';
import '../../core/widgets/page_tabs.dart';
import '../../core/widgets/app_error_state.dart';
import '../../core/utils/url_utils.dart';

// --- Constants ---------------------------------------------------------------

const _accent = Color(0xFF3472A4);
const _green = Color(0xFF10B981);
const _red = Color(0xFFEF4444);

// --- Providers ---------------------------------------------------------------

final _projectDetailProvider =
    FutureProvider.family<Map<String, dynamic>, String>((ref, projectId) async {
  final api = ref.read(apiClientProvider);
  final res = await api.get('/projects/$projectId');
  return res.data as Map<String, dynamic>;
});

final _platformsProvider =
    FutureProvider.family<List<Map<String, dynamic>>, String>(
        (ref, projectId) async {
  final api = ref.read(apiClientProvider);
  final res = await api.get('/projects/$projectId/platforms');
  final data = res.data as Map<String, dynamic>;
  return List<Map<String, dynamic>>.from(data['platforms'] ?? []);
});

final _webhooksProvider =
    FutureProvider.family<List<Map<String, dynamic>>, String>(
        (ref, projectId) async {
  final api = ref.read(apiClientProvider);
  final res = await api.get('/webhooks', params: {'projectId': projectId});
  final data = res.data as Map<String, dynamic>;
  return List<Map<String, dynamic>>.from(data['webhooks'] ?? []);
});


// --- Page --------------------------------------------------------------------

class SettingsPage extends ConsumerStatefulWidget {
  const SettingsPage({super.key});

  @override
  ConsumerState<SettingsPage> createState() => _SettingsPageState();
}

class _SettingsPageState extends ConsumerState<SettingsPage> {
  // General tab
  final _nameCtrl = TextEditingController();
  final _descCtrl = TextEditingController();
  bool _generalDirty = false;
  bool _generalSaving = false;

  // API key search
  final _apiKeySearchCtrl = TextEditingController();

  // Platform search
  final _platformSearchCtrl = TextEditingController();

  // Webhook search
  final _webhookSearchCtrl = TextEditingController();

  @override
  void dispose() {
    _nameCtrl.dispose();
    _descCtrl.dispose();
    _apiKeySearchCtrl.dispose();
    _platformSearchCtrl.dispose();
    _webhookSearchCtrl.dispose();
    super.dispose();
  }

  static const _tabNames = [
    'general', 'api-keys', 'webhooks', 'audit-log',
  ];

  @override
  Widget build(BuildContext context) {
    final colors = consoleColors(context);
    final routerState = GoRouterState.of(context);
    final projectId = routerState.pathParameters['projectId'];
    final tabIndex = _tabNames.indexOf(
      routerState.uri.queryParameters['tab'] ?? 'general',
    ).clamp(0, _tabNames.length - 1);
    if (projectId == null) {
      return Center(
          child: Text('No project selected',
              style: TextStyle(color: colors.textSecondary)));
    }

    return Scaffold(
      backgroundColor: colors.background,
      body: Padding(
        padding: EdgeInsets.symmetric(
          horizontal: pageHPad(context),
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const SizedBox(height: 32),
            // Title
            Text('Settings',
                style: TextStyle(
                color: colors.textPrimary,
                    fontSize: 22,
                    fontWeight: FontWeight.w600)),
            const SizedBox(height: 4),
            Text('Manage your project configuration',
                style: TextStyle(
                color: colors.textSecondary, fontSize: 14)),
            const SizedBox(height: 24),

            // Tabs
            PageTabs(
              tabs: const [
                'General',
                'API Keys',
                'Webhooks',
                'Audit Log',
              ],
              selected: tabIndex,
              onChanged: (i) => context.go(
                withQuery(context, {'tab': _tabNames[i]}),
              ),
            ),
            const SizedBox(height: 24),

            // API keys (1) and audit log (3) fill remaining height without scroll wrapper
            Expanded(
              child: (tabIndex == 1 || tabIndex == 3)
                  ? _buildTabBody(projectId, tabIndex)
                  : SingleChildScrollView(
                      child: _buildTabBody(projectId, tabIndex),
                    ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildTabBody(String projectId, int tabIndex) {
    switch (tabIndex) {
      case 0:
        return _buildGeneralTab(projectId);
      case 1:
        return _buildApiKeysTab(projectId);
      case 2:
        return _buildWebhooksTab(projectId);
      case 3:
        return _buildAuditLogTab(projectId);
      default:
        return const SizedBox();
    }
  }

  // ===========================================================================
  // General Tab
  // ===========================================================================

  Widget _buildGeneralTab(String projectId) {
    final projectAsync = ref.watch(_projectDetailProvider(projectId));

    return projectAsync.when(
      loading: () => const Center(child: CircularProgressIndicator()),
      error: (e, _) => AppErrorState(error: e),
      data: (project) {
        // Sync controllers on first load
        if (!_generalDirty) {
          final name = project['name'] as String? ?? '';
          final desc = project['description'] as String? ?? '';
          if (_nameCtrl.text.isEmpty && name.isNotEmpty) {
            _nameCtrl.text = name;
          }
          if (_descCtrl.text.isEmpty && desc.isNotEmpty) {
            _descCtrl.text = desc;
          }
        }

        return Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            // Project details card
            _SettingsCard(
              title: 'Project details',
              subtitle: 'Update your project name and description',
              trailing: _generalDirty
                  ? FilledButton(
                      style: FilledButton.styleFrom(
                        backgroundColor: _accent,
                        padding: const EdgeInsets.symmetric(
                            horizontal: 20, vertical: 10),
                        shape: RoundedRectangleBorder(
                            borderRadius: BorderRadius.circular(8)),
                      ),
                      onPressed:
                          _generalSaving ? null : () => _saveGeneral(projectId),
                      child: _generalSaving
                          ? const SizedBox(
                              width: 14,
                              height: 14,
                              child: CircularProgressIndicator(
                                  strokeWidth: 2, color: Colors.white))
                          : const Text('Save',
                              style: TextStyle(fontSize: 13)),
                    )
                  : null,
              children: [
                _SettingsField(
                  label: 'Project name',
                  controller: _nameCtrl,
                  hint: 'My project',
                  onChanged: (_) {
                    if (!_generalDirty) setState(() => _generalDirty = true);
                  },
                ),
                const SizedBox(height: 16),
                _SettingsField(
                  label: 'Description',
                  controller: _descCtrl,
                  hint: 'Optional description',
                  onChanged: (_) {
                    if (!_generalDirty) setState(() => _generalDirty = true);
                  },
                ),
                const SizedBox(height: 16),
                _SettingsField(
                  label: 'Project ID',
                  value: projectId,
                  readOnly: true,
                  copyable: true,
                ),
              ],
            ),
            const SizedBox(height: 20),

            // Services card
            _SettingsCard(
              title: 'Services',
              subtitle: 'Enable or disable individual services for this project',
              children: [
                // Core services (enabled by default)
                _ServiceToggle(
                    label: 'Auth',
                    description: 'User authentication and management',
                    icon: LucideIcons.users,
                    enabled: true,
                    onChanged: (_) {}),
                _ServiceToggle(
                    label: 'Databases',
                    description: 'Structured data storage',
                    icon: LucideIcons.database,
                    enabled: true,
                    onChanged: (_) {}),
                _ServiceToggle(
                    label: 'Storage',
                    description: 'File storage and management',
                    icon: LucideIcons.folderClosed,
                    enabled: true,
                    onChanged: (_) {}),
                _ServiceToggle(
                    label: 'Functions',
                    description: 'Serverless function execution',
                    icon: LucideIcons.zap,
                    enabled: true,
                    onChanged: (_) {}),
                _ServiceToggle(
                    label: 'Messaging',
                    description: 'Email, SMS, and push notifications',
                    icon: LucideIcons.messageSquare,
                    enabled: true,
                    onChanged: (_) {}),
                _ServiceToggle(
                    label: 'Workflows',
                    description: 'DAG workflow engine and automation',
                    icon: LucideIcons.gitBranch,
                    enabled: true,
                    onChanged: (_) {}),
                _ServiceToggle(
                    label: 'Realtime',
                    description: 'WebSocket pub/sub subscriptions',
                    icon: LucideIcons.radio,
                    enabled: true,
                    onChanged: (_) {}),
                // Deploy services (enabled by default)
                _ServiceToggle(
                    label: 'Sites',
                    description: 'Static site hosting with custom domains',
                    icon: LucideIcons.globe,
                    enabled: true,
                    onChanged: (_) {}),
                _ServiceToggle(
                    label: 'Containers',
                    description: 'Docker container deployments',
                    icon: LucideIcons.box,
                    enabled: true,
                    onChanged: (_) {}),
                // Experimental services (disabled by default)
                Padding(
                  padding: const EdgeInsets.only(top: 12, bottom: 4),
                  child: Text('Experimental',
                      style: TextStyle(
                          color: Colors.white.withValues(alpha: 0.3),
                          fontSize: 11,
                          fontWeight: FontWeight.w600,
                          letterSpacing: 0.5)),
                ),
                _ServiceToggle(
                    label: 'Mobile',
                    description: 'Mobile app builds and distribution',
                    icon: LucideIcons.smartphone,
                    enabled: false,
                    onChanged: (_) {}),
                _ServiceToggle(
                    label: 'Desktop',
                    description: 'Desktop app builds and distribution',
                    icon: LucideIcons.monitor,
                    enabled: false,
                    onChanged: (_) {}),
                _ServiceToggle(
                    label: 'Feature Flags',
                    description: 'Feature flags and remote config',
                    icon: LucideIcons.toggleRight,
                    enabled: false,
                    onChanged: (_) {}),
                _ServiceToggle(
                    label: 'Environments',
                    description: 'Environment variables and staging',
                    icon: LucideIcons.layers,
                    enabled: false,
                    onChanged: (_) {}),
              ],
            ),
            const SizedBox(height: 20),

            // Danger zone
            _SettingsCard(
              title: 'Danger zone',
              subtitle:
                  'Irreversible actions. Please proceed with caution.',
              danger: true,
              children: [
                Row(
                  children: [
                    Expanded(
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          const Text('Delete project',
                              style: TextStyle(
                                  color: Colors.white,
                                  fontSize: 14,
                                  fontWeight: FontWeight.w500)),
                          const SizedBox(height: 4),
                          Text(
                              'Permanently delete this project and all its data. This action cannot be undone.',
                              style: TextStyle(
                                  color: Colors.white.withValues(alpha: 0.4),
                                  fontSize: 13)),
                        ],
                      ),
                    ),
                    const SizedBox(width: 16),
                    OutlinedButton(
                      style: OutlinedButton.styleFrom(
                        foregroundColor: _red,
                        side: const BorderSide(color: _red),
                        padding: const EdgeInsets.symmetric(
                            horizontal: 20, vertical: 10),
                        shape: RoundedRectangleBorder(
                            borderRadius: BorderRadius.circular(8)),
                      ),
                      onPressed: () => _deleteProject(projectId),
                      child: const Text('Delete project',
                          style: TextStyle(fontSize: 13)),
                    ),
                  ],
                ),
              ],
            ),
            const SizedBox(height: 40),
          ],
        );
      },
    );
  }

  Future<void> _saveGeneral(String projectId) async {
    setState(() => _generalSaving = true);
    try {
      final api = ref.read(apiClientProvider);
      await api.patch('/projects/$projectId', data: {
        'name': _nameCtrl.text.trim(),
        'description': _descCtrl.text.trim(),
      });
      setState(() {
        _generalDirty = false;
        _generalSaving = false;
      });
      ref.invalidate(_projectDetailProvider(projectId));
      ref.invalidate(projectsProvider);
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('Project updated')),
        );
      }
    } catch (e) {
      setState(() => _generalSaving = false);
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('Error: $e')),
        );
      }
    }
  }

  Future<void> _deleteProject(String projectId) async {
    final colors = consoleColors(context);
    final confirmed = await showAppDialog<bool>(
      context: context,
      title: 'Delete project',
      content: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            'This will permanently delete the project and all its data including databases, storage, functions, and deployments.',
            style: TextStyle(color: colors.textSecondary),
          ),
          const SizedBox(height: 16),
          Text('Type the project ID to confirm:',
              style: TextStyle(
                  color: colors.textSecondary, fontSize: 13)),
          const SizedBox(height: 8),
          _DeleteConfirmField(projectId: projectId),
        ],
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
    if (confirmed == true && mounted) {
      try {
        await ref.read(apiClientProvider).delete('/projects/$projectId');
        ref.invalidate(projectsProvider);
        if (mounted) context.go('/projects');
      } catch (e) {
        if (mounted) {
          ScaffoldMessenger.of(context)
              .showSnackBar(SnackBar(content: Text('Error: $e')));
        }
      }
    }
  }

  // ===========================================================================
  // API Keys Tab
  // ===========================================================================

  Widget _buildApiKeysTab(String projectId) {
    final keysAsync = ref.watch(apiKeysProvider(projectId));
    final keys = keysAsync.valueOrNull ?? [];
    final total = keys.length;

    return keysAsync.when(
      loading: () => const Center(child: CircularProgressIndicator()),
      error: (e, _) => AppErrorState(error: e),
      data: (_) => AppDataTable(
        columns: const [
          AppTableColumn(key: 'name',         label: 'Name',    flex: 3),
          AppTableColumn(key: 'secretPrefix', label: 'Secret',  flex: 3, sortable: false),
          AppTableColumn(key: 'scopes',       label: 'Scopes',  flex: 2, sortable: false),
          AppTableColumn(key: 'expire',       label: 'Expires', flex: 2, sortable: false),
          AppTableColumn(key: r'$createdAt',  label: 'Created', flex: 2),
        ],
        rows: keys,
        getCellValue: (row, key) => switch (key) {
          'name'         => row['name'] as String? ?? '',
          'secretPrefix' => row['secretPrefix'] as String? ?? '',
          'scopes'       => _scopeSummary(row),
          'expire'       => _expiryLabel(row['expire'] as String?),
          r'$createdAt'  => _fmtDate(row[r'\$createdAt'] as String? ?? row['createdAt'] as String? ?? ''),
          _              => '',
        },
        cellBuilder: (row, key) {
          if (key == 'secretPrefix') {
            final prefix = row['secretPrefix'] as String? ?? '';
            return prefix.isEmpty ? null : _PrefixCell(prefix: prefix);
          }
          if (key == 'expire') {
            final iso = row['expire'] as String?;
            if (iso == null || iso.isEmpty) {
              return Text('Never',
                  style: TextStyle(color: consoleColors(context).textSubtle, fontSize: 12));
            }
            final expired = DateTime.tryParse(iso)?.isBefore(DateTime.now()) ?? false;
            return Container(
              padding: const EdgeInsets.symmetric(horizontal: 7, vertical: 2),
              decoration: BoxDecoration(
                color: (expired ? _red : _green).withValues(alpha: 0.12),
                borderRadius: BorderRadius.circular(4),
              ),
              child: Text(_expiryLabel(iso),
                  style: TextStyle(
                      color: expired ? _red : _green,
                      fontSize: 11,
                      fontWeight: FontWeight.w500)),
            );
          }
          return null;
        },
        getRowIcon: (_) => LucideIcons.key,
        onRowTap: (row) => context.go(
          '/project/$projectId/settings/keys/${row['\$id']}',
        ),
        onDeleteRow: (row) => _deleteKey(projectId, row['\$id'] as String),
        createLabel: 'Create API key',
        onCreateTap: () => _showCreateKeyDialog(projectId),
        total: total,
        perPage: 25,
        currentPage: 1,
        onPrev: () {},
        onNext: () {},
        onPerPageChanged: (_) {},
        itemLabel: 'API keys',
        searchController: _apiKeySearchCtrl,
        onSearch: () => setState(() {}),
        searchHint: 'Search by name',
        emptyIcon: LucideIcons.key,
        emptyTitle: 'No API keys',
        emptySubtitle: 'Create an API key to authenticate server-side requests',
      ),
    );
  }

  static String _scopeSummary(Map<String, dynamic> row) {
    final scopes = (row['scopes'] as List?)?.cast<String>() ?? [];
    return scopes.isEmpty ? 'All scopes' : '${scopes.length} scope${scopes.length == 1 ? '' : 's'}';
  }

  static String _expiryLabel(String? iso) {
    if (iso == null || iso.isEmpty) return 'Never';
    final t = DateTime.tryParse(iso)?.toLocal();
    if (t == null) return 'Never';
    return '${t.year}-${t.month.toString().padLeft(2, '0')}-${t.day.toString().padLeft(2, '0')}';
  }

  static String _fmtDate(String iso) {
    if (iso.isEmpty) return '';
    final t = DateTime.tryParse(iso)?.toLocal();
    if (t == null) return iso;
    return '${t.year}-${t.month.toString().padLeft(2, '0')}-${t.day.toString().padLeft(2, '0')}';
  }

  void _showCreateKeyDialog(String projectId) {
    showDialog(
      context: context,
      barrierColor: Colors.black.withValues(alpha: 0.6),
      builder: (ctx) => _CreateKeyDialog(
        projectId: projectId,
        api: ref.read(apiClientProvider),
        onCreated: (secret) {
          ref.invalidate(apiKeysProvider(projectId));
          _showSecretDialog(secret);
        },
      ),
    );
  }

  void _showSecretDialog(String secret) {
    showAppDialog(
      context: context,
      title: 'Copy your API key',
      subtitle: 'This is the only time your key will be shown. Copy it now.',
      content: _SecretRevealField(secret: secret),
      actions: [
        AppDialogAction(
          label: 'Done',
          onTap: () => Navigator.of(context, rootNavigator: true).pop(),
        ),
      ],
    );
  }

  Future<void> _deleteKey(String projectId, String keyId) async {
    final colors = consoleColors(context);
    final confirmed = await showAppDialog<bool>(
      context: context,
      title: 'Delete API key',
      content: Text(
          'Any applications using this key will lose access immediately.',
          style: TextStyle(color: colors.textSecondary)),
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
      await ref.read(apiClientProvider).delete('/projects/$projectId/keys/$keyId');
      ref.invalidate(apiKeysProvider(projectId));
    }
  }

  // ===========================================================================
  // Platforms Tab
  // ===========================================================================

  // ignore: unused_element
  Widget _buildPlatformsTab(String projectId) {
    final colors = consoleColors(context);
    final platformsAsync = ref.watch(_platformsProvider(projectId));

    return platformsAsync.when(
      loading: () => const Center(child: CircularProgressIndicator()),
      error: (e, _) => AppErrorState(error: e),
      data: (platforms) {
        final query = _platformSearchCtrl.text.toLowerCase();
        final filtered = query.isEmpty
            ? platforms
            : platforms.where((p) {
                final name = (p['name'] as String? ?? '').toLowerCase();
                final type = (p['type'] as String? ?? '').toLowerCase();
                return name.contains(query) || type.contains(query);
              }).toList();

        return Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                // Search
                SizedBox(
                  width: 280,
                  child: TextField(
                    controller: _platformSearchCtrl,
                    onChanged: (_) => setState(() {}),
                    style: TextStyle(fontSize: 13, color: colors.textPrimary),
                    decoration: InputDecoration(
                      hintText: 'Search platforms...',
                      hintStyle: TextStyle(color: colors.textSubtle, fontSize: 13),
                      prefixIcon: Padding(
                        padding: const EdgeInsets.only(left: 10, right: 6),
                        child: Icon(Icons.search,
                        size: 16, color: colors.textSubtle),
                      ),
                      prefixIconConstraints:
                          const BoxConstraints(minWidth: 32, minHeight: 0),
                      filled: true,
                      fillColor: colors.fieldFill,
                      isDense: true,
                      contentPadding: const EdgeInsets.symmetric(
                          vertical: 10, horizontal: 12),
                      border: OutlineInputBorder(
                          borderRadius: BorderRadius.circular(8),
                        borderSide: BorderSide(color: colors.fieldBorder)),
                      enabledBorder: OutlineInputBorder(
                          borderRadius: BorderRadius.circular(8),
                        borderSide: BorderSide(color: colors.fieldBorder)),
                      focusedBorder: OutlineInputBorder(
                          borderRadius: BorderRadius.circular(8),
                          borderSide: const BorderSide(color: _accent)),
                    ),
                  ),
                ),
                const SizedBox(width: 12),
                Text('${platforms.length} platform${platforms.length == 1 ? '' : 's'}',
                  style: TextStyle(color: colors.textSecondary, fontSize: 13)),
                const Spacer(),
                FilledButton.icon(
                  style: FilledButton.styleFrom(
                    backgroundColor: _accent,
                    padding: const EdgeInsets.symmetric(
                        horizontal: 14, vertical: 8),
                    shape: RoundedRectangleBorder(
                        borderRadius: BorderRadius.circular(8)),
                  ),
                  icon: const Icon(LucideIcons.plus, size: 14),
                  label: const Text('Add platform',
                      style: TextStyle(fontSize: 12)),
                  onPressed: () => _showAddPlatformDialog(projectId),
                ),
              ],
            ),
            const SizedBox(height: 16),

            if (filtered.isEmpty)
              _EmptyState(
                icon: LucideIcons.smartphone,
                title: 'No platforms registered',
                subtitle:
                    'Register platforms to restrict API access to known apps',
                actionLabel: 'Add platform',
                onAction: () => _showAddPlatformDialog(projectId),
              )
            else
              ...filtered.map((p) => _PlatformCard(
                    key: ValueKey(p['\$id']),
                    platform: p,
                    onDelete: () => _deletePlatform(projectId, p['\$id']),
                  )),
            const SizedBox(height: 40),
          ],
        );
      },
    );
  }

  void _showAddPlatformDialog(String projectId) {
    final colors = consoleColors(context);
    final nameCtrl = TextEditingController();
    final hostnameCtrl = TextEditingController();
    String selectedType = 'web';

    showAppDialog(
      context: context,
      title: 'Add platform',
      subtitle: 'Register an application platform',
      content: StatefulBuilder(
        builder: (ctx, setDialogState) => Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            // Type selector
            Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text('Type',
                    style: TextStyle(
                    color: colors.textSecondary,
                        fontSize: 12,
                        fontWeight: FontWeight.w500)),
                const SizedBox(height: 6),
                Wrap(
                  spacing: 8,
                  children: [
                    _PlatformTypeChip(
                        label: 'Web',
                        icon: LucideIcons.globe,
                        selected: selectedType == 'web',
                        onTap: () =>
                            setDialogState(() => selectedType = 'web')),
                    _PlatformTypeChip(
                        label: 'Flutter (iOS)',
                        icon: LucideIcons.smartphone,
                        selected: selectedType == 'flutter-ios',
                        onTap: () => setDialogState(
                            () => selectedType = 'flutter-ios')),
                    _PlatformTypeChip(
                        label: 'Flutter (Android)',
                        icon: LucideIcons.smartphone,
                        selected: selectedType == 'flutter-android',
                        onTap: () => setDialogState(
                            () => selectedType = 'flutter-android')),
                    _PlatformTypeChip(
                        label: 'Apple (iOS)',
                        icon: LucideIcons.smartphone,
                        selected: selectedType == 'apple-ios',
                        onTap: () => setDialogState(
                            () => selectedType = 'apple-ios')),
                    _PlatformTypeChip(
                        label: 'Android',
                        icon: LucideIcons.smartphone,
                        selected: selectedType == 'android',
                        onTap: () => setDialogState(
                            () => selectedType = 'android')),
                  ],
                ),
              ],
            ),
            const SizedBox(height: 16),
            AppDialogField(
                controller: nameCtrl,
                label: 'Name',
                hint: 'My web app',
                autofocus: true),
            const SizedBox(height: 16),
            AppDialogField(
              controller: hostnameCtrl,
              label: selectedType == 'web'
                  ? 'Hostname'
                  : selectedType.contains('android')
                      ? 'Package name'
                      : 'Bundle ID',
              hint: selectedType == 'web'
                  ? 'localhost'
                  : selectedType.contains('android')
                      ? 'com.example.app'
                      : 'com.example.app',
            ),
          ],
        ),
      ),
      actions: [
        const AppDialogCancel(),
        AppDialogAction(
          label: 'Create',
          onTap: () async {
            if (nameCtrl.text.trim().isEmpty) return;
            final api = ref.read(apiClientProvider);
            await api.post('/projects/$projectId/platforms', data: {
              'type': selectedType,
              'name': nameCtrl.text.trim(),
              'hostname': hostnameCtrl.text.trim(),
            });
            if (mounted) {
              Navigator.of(context, rootNavigator: true).pop();
            }
            ref.invalidate(_platformsProvider(projectId));
          },
        ),
      ],
    );
  }

  Future<void> _deletePlatform(String projectId, String platformId) async {
    final colors = consoleColors(context);
    final confirmed = await showAppDialog<bool>(
      context: context,
      title: 'Remove platform',
      content: Text('This platform will no longer have API access.',
          style: TextStyle(color: colors.textSecondary)),
      actions: [
        const AppDialogCancel(),
        AppDialogAction(
          label: 'Remove',
          destructive: true,
          onTap: () => Navigator.of(context, rootNavigator: true).pop(true),
        ),
      ],
    );
    if (confirmed == true) {
      await ref
          .read(apiClientProvider)
          .delete('/projects/$projectId/platforms/$platformId');
      ref.invalidate(_platformsProvider(projectId));
    }
  }

  // ===========================================================================
  // Webhooks Tab
  // ===========================================================================

  Widget _buildWebhooksTab(String projectId) {
    final colors = consoleColors(context);
    final webhooksAsync = ref.watch(_webhooksProvider(projectId));

    return webhooksAsync.when(
      loading: () => const Center(child: CircularProgressIndicator()),
      error: (e, _) => AppErrorState(error: e),
      data: (webhooks) {
        final query = _webhookSearchCtrl.text.toLowerCase();
        final filtered = query.isEmpty
            ? webhooks
            : webhooks.where((w) {
                final name = (w['name'] as String? ?? '').toLowerCase();
                final url = (w['url'] as String? ?? '').toLowerCase();
                return name.contains(query) || url.contains(query);
              }).toList();

        return Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                SizedBox(
                  width: 280,
                  child: TextField(
                    controller: _webhookSearchCtrl,
                    onChanged: (_) => setState(() {}),
                    style: TextStyle(fontSize: 13, color: colors.textPrimary),
                    decoration: InputDecoration(
                      hintText: 'Search webhooks...',
                      hintStyle: TextStyle(color: colors.textSubtle, fontSize: 13),
                      prefixIcon: Padding(
                        padding: const EdgeInsets.only(left: 10, right: 6),
                        child: Icon(Icons.search,
                        size: 16, color: colors.textSubtle),
                      ),
                      prefixIconConstraints:
                          const BoxConstraints(minWidth: 32, minHeight: 0),
                      filled: true,
                      fillColor: colors.fieldFill,
                      isDense: true,
                      contentPadding: const EdgeInsets.symmetric(
                          vertical: 10, horizontal: 12),
                      border: OutlineInputBorder(
                          borderRadius: BorderRadius.circular(8),
                        borderSide: BorderSide(color: colors.fieldBorder)),
                      enabledBorder: OutlineInputBorder(
                          borderRadius: BorderRadius.circular(8),
                        borderSide: BorderSide(color: colors.fieldBorder)),
                      focusedBorder: OutlineInputBorder(
                          borderRadius: BorderRadius.circular(8),
                          borderSide: const BorderSide(color: _accent)),
                    ),
                  ),
                ),
                const SizedBox(width: 12),
                Text(
                    '${webhooks.length} webhook${webhooks.length == 1 ? '' : 's'}',
                  style: TextStyle(color: colors.textSecondary, fontSize: 13)),
                const Spacer(),
                FilledButton.icon(
                  style: FilledButton.styleFrom(
                    backgroundColor: _accent,
                    padding: const EdgeInsets.symmetric(
                        horizontal: 14, vertical: 8),
                    shape: RoundedRectangleBorder(
                        borderRadius: BorderRadius.circular(8)),
                  ),
                  icon: const Icon(LucideIcons.plus, size: 14),
                  label: const Text('Create webhook',
                      style: TextStyle(fontSize: 12)),
                  onPressed: () => _showCreateWebhookDialog(projectId),
                ),
              ],
            ),
            const SizedBox(height: 16),

            if (filtered.isEmpty)
              _EmptyState(
                icon: LucideIcons.webhook,
                title: 'No webhooks configured',
                subtitle:
                    'Webhooks send real-time notifications to your server when events occur',
                actionLabel: 'Create webhook',
                onAction: () => _showCreateWebhookDialog(projectId),
              )
            else
              ...filtered.map((w) => _WebhookCard(
                    key: ValueKey(w['\$id']),
                    webhook: w,
                    onDelete: () => _deleteWebhook(w['\$id']),
                  )),
            const SizedBox(height: 40),
          ],
        );
      },
    );
  }

  void _showCreateWebhookDialog(String projectId) {
    final colors = consoleColors(context);
    final nameCtrl = TextEditingController();
    final urlCtrl = TextEditingController();
    final Set<String> selectedEvents = {};

    const events = [
      'databases.*',
      'storage.*',
      'users.*',
      'teams.*',
      'functions.*',
      'messaging.*',
      'deploy.*',
      'workflows.*',
      'credentials.*',
    ];

    showAppDialog(
      context: context,
      title: 'Create webhook',
      subtitle: 'Receive notifications when events occur in your project',
      width: 520,
      content: StatefulBuilder(
        builder: (ctx, setDialogState) => Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            AppDialogField(
                controller: nameCtrl,
                label: 'Name',
                hint: 'My webhook',
                autofocus: true),
            const SizedBox(height: 16),
            AppDialogField(
              controller: urlCtrl,
              label: 'POST URL',
              hint: 'https://example.com/webhook',
            ),
            const SizedBox(height: 16),
            Text('Events',
                style: TextStyle(
                color: colors.textSecondary,
                    fontSize: 12,
                    fontWeight: FontWeight.w500)),
            const SizedBox(height: 8),
            Wrap(
              spacing: 8,
              runSpacing: 8,
              children: events.map((e) {
                final selected = selectedEvents.contains(e);
                return FilterChip(
                  label: Text(e,
                      style: TextStyle(
                          color: selected ? colors.textPrimary : colors.textSecondary,
                          fontSize: 12)),
                  selected: selected,
                  onSelected: (v) {
                    setDialogState(() {
                      if (v) {
                        selectedEvents.add(e);
                      } else {
                        selectedEvents.remove(e);
                      }
                    });
                  },
                    backgroundColor: colors.fieldFill,
                  selectedColor: _accent.withValues(alpha: 0.3),
                  checkmarkColor: Colors.white,
                  side: BorderSide(
                      color: selected
                          ? _accent
                        : colors.fieldBorder),
                  shape: RoundedRectangleBorder(
                      borderRadius: BorderRadius.circular(6)),
                );
              }).toList(),
            ),
          ],
        ),
      ),
      actions: [
        const AppDialogCancel(),
        AppDialogAction(
          label: 'Create',
          onTap: () async {
            if (nameCtrl.text.trim().isEmpty || urlCtrl.text.trim().isEmpty) {
              return;
            }
            final api = ref.read(apiClientProvider);
            await api.post('/webhooks', data: {
              'name': nameCtrl.text.trim(),
              'url': urlCtrl.text.trim(),
              'events': selectedEvents.toList(),
              'enabled': true,
            });
            if (mounted) {
              Navigator.of(context, rootNavigator: true).pop();
            }
            ref.invalidate(_webhooksProvider(projectId));
          },
        ),
      ],
    );
  }

  Future<void> _deleteWebhook(String webhookId) async {
    final colors = consoleColors(context);
    final confirmed = await showAppDialog<bool>(
      context: context,
      title: 'Delete webhook',
      content: Text('This webhook will stop receiving notifications.',
          style: TextStyle(color: colors.textSecondary)),
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
      await ref.read(apiClientProvider).delete('/webhooks/$webhookId');
      if (!mounted) return;
      final routerState = GoRouterState.of(context);
      final projectId = routerState.pathParameters['projectId'];
      if (projectId != null) {
        ref.invalidate(_webhooksProvider(projectId));
      }
    }
  }

  // ===========================================================================
  // Audit Log Tab
  // ===========================================================================

  Widget _buildAuditLogTab(String projectId) =>
      _AuditLogTab(projectId: projectId);
}

// =============================================================================
// Shared Components
// =============================================================================

class _SettingsCard extends StatelessWidget {
  final String title;
  final String? subtitle;
  final List<Widget> children;
  final Widget? trailing;
  final bool danger;

  const _SettingsCard({
    required this.title,
    this.subtitle,
    required this.children,
    this.trailing,
    this.danger = false,
  });

  @override
  Widget build(BuildContext context) {
    final colors = consoleColors(context);
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: colors.surface,
        borderRadius: BorderRadius.circular(8),
        border: Border.all(
            color: danger
                ? _red.withValues(alpha: 0.3)
                : colors.border),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(title,
                        style: TextStyle(
                        color: danger ? _red : colors.textPrimary,
                            fontSize: 15,
                            fontWeight: FontWeight.w600)),
                    if (subtitle != null) ...[
                      const SizedBox(height: 4),
                      Text(subtitle!,
                          style: TextStyle(
                            color: colors.textSecondary,
                              fontSize: 13)),
                    ],
                  ],
                ),
              ),
              if (trailing != null) trailing!,
            ],
          ),
          const SizedBox(height: 16),
            Container(height: 1, color: colors.border),
          const SizedBox(height: 16),
          ...children,
        ],
      ),
    );
  }
}

class _SettingsField extends StatelessWidget {
  final String label;
  final TextEditingController? controller;
  final String? value;
  final String hint;
  final bool readOnly;
  final bool copyable;
  final ValueChanged<String>? onChanged;

  const _SettingsField({
    required this.label,
    this.controller,
    this.value,
    this.hint = '',
    this.readOnly = false,
    this.copyable = false,
    this.onChanged,
  });

  @override
  Widget build(BuildContext context) {
    final colors = consoleColors(context);
    return Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        SizedBox(
          width: 160,
          child: Padding(
            padding: const EdgeInsets.only(top: 10),
            child: Text(label,
                style: TextStyle(
                color: colors.textSecondary,
                    fontSize: 13,
                    fontWeight: FontWeight.w500)),
          ),
        ),
        const SizedBox(width: 16),
        Expanded(
          child: readOnly
              ? Container(
                  padding: const EdgeInsets.symmetric(
                      horizontal: 12, vertical: 10),
                  decoration: BoxDecoration(
                    color: colors.fieldFill,
                    borderRadius: BorderRadius.circular(8),
                    border: Border.all(color: colors.fieldBorder),
                  ),
                  child: Row(
                    children: [
                      Expanded(
                        child: SelectableText(
                          value ?? controller?.text ?? '',
                          style: TextStyle(
                            color: colors.textSecondary,
                            fontSize: 13,
                            fontFamily: 'monospace',
                          ),
                        ),
                      ),
                      if (copyable)
                        GestureDetector(
                          onTap: () {
                            Clipboard.setData(ClipboardData(
                                text: value ?? controller?.text ?? ''));
                            ScaffoldMessenger.of(context).showSnackBar(
                              const SnackBar(
                                  content: Text('Copied to clipboard')),
                            );
                          },
                          child: Icon(LucideIcons.copy,
                              size: 14, color: colors.textSecondary),
                        ),
                    ],
                  ),
                )
              : TextField(
                  controller: controller,
                  onChanged: onChanged,
                  style: TextStyle(
                    color: colors.textPrimary, fontSize: 13),
                  decoration: InputDecoration(
                    hintText: hint,
                    hintStyle: TextStyle(
                    color: colors.textSubtle,
                        fontSize: 13),
                    filled: true,
                  fillColor: colors.fieldFill,
                    isDense: true,
                    contentPadding: const EdgeInsets.symmetric(
                        horizontal: 12, vertical: 10),
                    border: OutlineInputBorder(
                        borderRadius: BorderRadius.circular(8),
                    borderSide: BorderSide(color: colors.fieldBorder)),
                    enabledBorder: OutlineInputBorder(
                        borderRadius: BorderRadius.circular(8),
                    borderSide: BorderSide(color: colors.fieldBorder)),
                    focusedBorder: OutlineInputBorder(
                        borderRadius: BorderRadius.circular(8),
                        borderSide: const BorderSide(color: _accent)),
                  ),
                ),
        ),
      ],
    );
  }
}

class _ServiceToggle extends StatelessWidget {
  final String label;
  final String description;
  final IconData icon;
  final bool enabled;
  final ValueChanged<bool> onChanged;

  const _ServiceToggle({
    required this.label,
    required this.description,
    required this.icon,
    required this.enabled,
    required this.onChanged,
  });

  @override
  Widget build(BuildContext context) {
    final colors = consoleColors(context);
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 6),
      child: Row(
        children: [
          Icon(icon, size: 16, color: colors.textSecondary),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(label,
                  style: TextStyle(
                    color: colors.textPrimary,
                        fontSize: 13,
                        fontWeight: FontWeight.w500)),
                Text(description,
                    style: TextStyle(
                    color: colors.textSecondary,
                        fontSize: 12)),
              ],
            ),
          ),
          Switch(
            value: enabled,
            onChanged: onChanged,
            activeThumbColor: _accent,
          ),
        ],
      ),
    );
  }
}


class _PlatformCard extends StatelessWidget {
  final Map<String, dynamic> platform;
  final VoidCallback onDelete;

  const _PlatformCard(
      {super.key, required this.platform, required this.onDelete});

  IconData _typeIcon(String type) {
    switch (type) {
      case 'web':
        return LucideIcons.globe;
      case 'flutter-ios':
      case 'apple-ios':
        return LucideIcons.smartphone;
      case 'flutter-android':
      case 'android':
        return LucideIcons.smartphone;
      default:
        return LucideIcons.monitor;
    }
  }

  String _typeLabel(String type) {
    switch (type) {
      case 'web':
        return 'Web';
      case 'flutter-ios':
        return 'Flutter iOS';
      case 'flutter-android':
        return 'Flutter Android';
      case 'apple-ios':
        return 'Apple iOS';
      case 'android':
        return 'Android';
      default:
        return type;
    }
  }

  @override
  Widget build(BuildContext context) {
    final colors = consoleColors(context);
    final name = platform['name'] as String? ?? 'Unnamed';
    final type = platform['type'] as String? ?? '';
    final hostname = platform['hostname'] as String? ?? '';

    return Container(
      margin: const EdgeInsets.only(bottom: 8),
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: colors.surface,
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: colors.border),
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
            child: Icon(_typeIcon(type), size: 18, color: _accent),
          ),
          const SizedBox(width: 14),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(name,
                  style: TextStyle(
                    color: colors.textPrimary,
                        fontSize: 14,
                        fontWeight: FontWeight.w500)),
                const SizedBox(height: 2),
                Row(
                  children: [
                    Container(
                      padding: const EdgeInsets.symmetric(
                          horizontal: 6, vertical: 2),
                      decoration: BoxDecoration(
                        color: _accent.withValues(alpha: 0.15),
                        borderRadius: BorderRadius.circular(4),
                      ),
                      child: Text(_typeLabel(type),
                          style: const TextStyle(
                              color: _accent,
                              fontSize: 11,
                              fontWeight: FontWeight.w500)),
                    ),
                    if (hostname.isNotEmpty) ...[
                      const SizedBox(width: 8),
                      Text(hostname,
                          style: TextStyle(
                            color: colors.textSecondary,
                              fontSize: 12,
                              fontFamily: 'monospace')),
                    ],
                  ],
                ),
              ],
            ),
          ),
          GestureDetector(
            onTap: onDelete,
            child: Icon(LucideIcons.trash2,
                size: 14, color: colors.textSubtle),
          ),
        ],
      ),
    );
  }
}

class _WebhookCard extends StatelessWidget {
  final Map<String, dynamic> webhook;
  final VoidCallback onDelete;

  const _WebhookCard(
      {super.key, required this.webhook, required this.onDelete});

  @override
  Widget build(BuildContext context) {
    final colors = consoleColors(context);
    final name = webhook['name'] as String? ?? 'Unnamed';
    final url = webhook['url'] as String? ?? '';
    final enabled = webhook['enabled'] as bool? ?? true;
    final events = List<String>.from(webhook['events'] ?? []);

    return Container(
      margin: const EdgeInsets.only(bottom: 8),
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: colors.surface,
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: colors.border),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Container(
                width: 36,
                height: 36,
                decoration: BoxDecoration(
                  color: _accent.withValues(alpha: 0.1),
                  borderRadius: BorderRadius.circular(8),
                ),
                child:
                    const Icon(LucideIcons.webhook, size: 18, color: _accent),
              ),
              const SizedBox(width: 14),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Row(
                      children: [
                        Text(name,
                          style: TextStyle(
                            color: colors.textPrimary,
                                fontSize: 14,
                                fontWeight: FontWeight.w500)),
                        const SizedBox(width: 8),
                        Container(
                          padding: const EdgeInsets.symmetric(
                              horizontal: 6, vertical: 2),
                          decoration: BoxDecoration(
                            color: (enabled ? _green : colors.textSubtle)
                                .withValues(alpha: 0.15),
                            borderRadius: BorderRadius.circular(4),
                          ),
                          child: Text(
                            enabled ? 'Active' : 'Disabled',
                            style: TextStyle(
                              color: enabled ? _green : colors.textSubtle,
                                fontSize: 11,
                                fontWeight: FontWeight.w500),
                          ),
                        ),
                      ],
                    ),
                    const SizedBox(height: 2),
                    Text(url,
                      style: TextStyle(
                        color: colors.textSecondary,
                            fontSize: 12,
                            fontFamily: 'monospace'),
                        overflow: TextOverflow.ellipsis),
                  ],
                ),
              ),
              GestureDetector(
                onTap: onDelete,
                child: Icon(LucideIcons.trash2,
                  size: 14, color: colors.textSubtle),
              ),
            ],
          ),
          if (events.isNotEmpty) ...[
            const SizedBox(height: 10),
            Wrap(
              spacing: 6,
              runSpacing: 4,
              children: events
                  .map((e) => Container(
                        padding: const EdgeInsets.symmetric(
                            horizontal: 6, vertical: 2),
                        decoration: BoxDecoration(
                          color: colors.fill,
                          borderRadius: BorderRadius.circular(4),
                          border: Border.all(color: colors.border),
                        ),
                        child: Text(e,
                            style: TextStyle(
                                color: colors.textSecondary,
                                fontSize: 11,
                                fontFamily: 'monospace')),
                      ))
                  .toList(),
            ),
          ],
        ],
      ),
    );
  }
}

class _EmptyState extends StatelessWidget {
  final IconData icon;
  final String title;
  final String subtitle;
  final String actionLabel;
  final VoidCallback onAction;

  const _EmptyState({
    required this.icon,
    required this.title,
    required this.subtitle,
    required this.actionLabel,
    required this.onAction,
  });

  @override
  Widget build(BuildContext context) {
    final colors = consoleColors(context);
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.symmetric(vertical: 60),
      child: Column(
        children: [
          Container(
            width: 48,
            height: 48,
            decoration: BoxDecoration(
              color: colors.fill,
              borderRadius: BorderRadius.circular(12),
            ),
            child: Icon(icon, size: 22, color: colors.textSubtle),
          ),
          const SizedBox(height: 16),
          Text(title,
              style: TextStyle(
                  color: colors.textPrimary,
                  fontSize: 15,
                  fontWeight: FontWeight.w500)),
          const SizedBox(height: 6),
          Text(subtitle,
              style: TextStyle(color: colors.textSecondary, fontSize: 13),
              textAlign: TextAlign.center),
          const SizedBox(height: 16),
          FilledButton(
            style: FilledButton.styleFrom(
              backgroundColor: _accent,
              padding:
                  const EdgeInsets.symmetric(horizontal: 20, vertical: 10),
              shape: RoundedRectangleBorder(
                  borderRadius: BorderRadius.circular(8)),
            ),
            onPressed: onAction,
            child: Text(actionLabel,
                style: const TextStyle(fontSize: 13)),
          ),
        ],
      ),
    );
  }
}

class _PlatformTypeChip extends StatelessWidget {
  final String label;
  final IconData icon;
  final bool selected;
  final VoidCallback onTap;

  const _PlatformTypeChip({
    required this.label,
    required this.icon,
    required this.selected,
    required this.onTap,
  });

  @override
  Widget build(BuildContext context) {
    final colors = consoleColors(context);
    return GestureDetector(
      onTap: onTap,
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
        decoration: BoxDecoration(
          color: selected ? _accent.withValues(alpha: 0.15) : colors.fieldFill,
          borderRadius: BorderRadius.circular(8),
          border: Border.all(color: selected ? _accent : colors.fieldBorder),
        ),
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(icon,
                size: 14,
              color: selected ? _accent : colors.textSecondary),
            const SizedBox(width: 6),
            Text(label,
                style: TextStyle(
                color: selected ? colors.textPrimary : colors.textSecondary,
                    fontSize: 12)),
          ],
        ),
      ),
    );
  }
}

class _DeleteConfirmField extends StatefulWidget {
  final String projectId;
  const _DeleteConfirmField({required this.projectId});

  @override
  State<_DeleteConfirmField> createState() => _DeleteConfirmFieldState();
}

class _DeleteConfirmFieldState extends State<_DeleteConfirmField> {
  final _ctrl = TextEditingController();

  @override
  void dispose() {
    _ctrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final colors = consoleColors(context);
    return TextField(
      controller: _ctrl,
      style: TextStyle(color: colors.textPrimary, fontSize: 13),
      decoration: InputDecoration(
        hintText: widget.projectId,
      hintStyle: TextStyle(color: colors.textSubtle, fontSize: 13),
        filled: true,
      fillColor: colors.fieldFill,
        isDense: true,
        contentPadding:
            const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
        border: OutlineInputBorder(
            borderRadius: BorderRadius.circular(8),
        borderSide: BorderSide(color: colors.fieldBorder)),
        enabledBorder: OutlineInputBorder(
            borderRadius: BorderRadius.circular(8),
        borderSide: BorderSide(color: colors.fieldBorder)),
        focusedBorder: OutlineInputBorder(
            borderRadius: BorderRadius.circular(8),
            borderSide: const BorderSide(color: _red)),
      ),
    );
  }
}

// =============================================================================
// Create API Key dialog
// =============================================================================

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
  'Functions': ['functions.read', 'functions.write', 'functions.execute'],
  'Messaging': ['messaging.read', 'messaging.write'],
  'Deploy': ['deploy.read', 'deploy.write'],
  'Workflows': ['workflows.read', 'workflows.write', 'workflows.execute'],
};

class _CreateKeyDialog extends StatefulWidget {
  final String projectId;
  final dynamic api;
  final void Function(String secret) onCreated;

  const _CreateKeyDialog({
    required this.projectId,
    required this.api,
    required this.onCreated,
  });

  @override
  State<_CreateKeyDialog> createState() => _CreateKeyDialogState();
}

class _CreateKeyDialogState extends State<_CreateKeyDialog> {
  final _nameCtrl = TextEditingController();
  String _expiry = 'never';
  final Set<String> _scopes = {};
  bool _loading = false;

  @override
  void dispose() {
    _nameCtrl.dispose();
    super.dispose();
  }

  String? _expiresAtIso() {
    if (_expiry == 'never') return null;
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
        return null;
    }
    return t.toIso8601String();
  }

  String? _expiryPreview() {
    final iso = _expiresAtIso();
    if (iso == null) return null;
    final t = DateTime.parse(iso).toLocal();
    final months = [
      'Jan','Feb','Mar','Apr','May','Jun',
      'Jul','Aug','Sep','Oct','Nov','Dec',
    ];
    return 'Your key will expire on ${months[t.month - 1]} ${t.day}, ${t.year}';
  }

  Future<void> _submit() async {
    if (_nameCtrl.text.trim().isEmpty) return;
    setState(() => _loading = true);
    try {
      final body = <String, dynamic>{
        'name': _nameCtrl.text.trim(),
        'scopes': _scopes.toList(),
      };
      final iso = _expiresAtIso();
      if (iso != null) body['expiresAt'] = iso;

      final res = await widget.api.post(
        '/projects/${widget.projectId}/keys',
        data: body,
      );
      final secret = (res.data as Map<String, dynamic>)['secret'] as String? ?? '';
      if (mounted) Navigator.of(context).pop();
      widget.onCreated(secret);
    } catch (_) {
      if (mounted) setState(() => _loading = false);
    }
  }

  void _toggleScope(String scope) {
    setState(() {
      if (_scopes.contains(scope)) {
        _scopes.remove(scope);
      } else {
        _scopes.add(scope);
      }
    });
  }

  void _toggleGroup(String group) {
    final groupScopes = _kScopeGroups[group]!;
    final allSelected = groupScopes.every(_scopes.contains);
    setState(() {
      if (allSelected) {
        _scopes.removeAll(groupScopes);
      } else {
        _scopes.addAll(groupScopes);
      }
    });
  }

  void _selectAll() => setState(() {
        for (final g in _kScopeGroups.values) {
          _scopes.addAll(g);
        }
      });

  void _deselectAll() => setState(() => _scopes.clear());

  @override
  Widget build(BuildContext context) {
    final colors = consoleColors(context);
    final preview = _expiryPreview();

    return Center(
      child: Material(
        color: Colors.transparent,
        child: Container(
          width: 520,
          constraints: const BoxConstraints(maxHeight: 680),
          decoration: BoxDecoration(
            color: colors.surface,
            borderRadius: BorderRadius.circular(12),
            border: Border.all(color: colors.border),
            boxShadow: [
              BoxShadow(
                color: colors.shadow,
                blurRadius: 32,
                offset: const Offset(0, 8),
              ),
            ],
          ),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              // Header
              Padding(
                padding: const EdgeInsets.fromLTRB(20, 20, 20, 0),
                child: Row(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Expanded(
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text('Create API key',
                              style: TextStyle(
                                  color: colors.textPrimary,
                                  fontSize: 16,
                                  fontWeight: FontWeight.w600)),
                          const SizedBox(height: 4),
                          Text(
                              'API keys authenticate server-side SDK requests',
                              style: TextStyle(
                                  color: colors.textMuted, fontSize: 13)),
                        ],
                      ),
                    ),
                    GestureDetector(
                      onTap: () => Navigator.of(context).pop(),
                      child: Icon(LucideIcons.x,
                          size: 16, color: colors.textMuted),
                    ),
                  ],
                ),
              ),
              const SizedBox(height: 16),
              Padding(
                padding: const EdgeInsets.symmetric(horizontal: 20),
                child: Container(height: 1, color: colors.border),
              ),
              const SizedBox(height: 16),

              // Scrollable body
              Flexible(
                child: SingleChildScrollView(
                  padding: const EdgeInsets.symmetric(horizontal: 20),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      // Name
                      AppDialogField(
                        controller: _nameCtrl,
                        label: 'Name',
                        hint: 'e.g. Production key',
                        autofocus: true,
                      ),
                      const SizedBox(height: 16),

                      // Expiry dropdown
                      AppSelectField<String>(
                        label: 'Expiration',
                        value: _expiry,
                        items: _kExpiryOptions.entries
                            .map((e) => DropdownMenuItem(
                                  value: e.key,
                                  child: Text(e.value),
                                ))
                            .toList(),
                        onChanged: (v) =>
                            setState(() => _expiry = v ?? 'never'),
                      ),
                      if (preview != null) ...[
                        const SizedBox(height: 6),
                        Row(
                          children: [
                            Icon(LucideIcons.info,
                                size: 12, color: colors.textSubtle),
                            const SizedBox(width: 5),
                            Text(preview,
                                style: TextStyle(
                                    color: colors.textSubtle, fontSize: 12)),
                          ],
                        ),
                      ],
                      const SizedBox(height: 20),

                      // Scopes header
                      Row(
                        children: [
                          Text('Scopes',
                              style: TextStyle(
                                  color: colors.textSecondary,
                                  fontSize: 12,
                                  fontWeight: FontWeight.w500)),
                          const Spacer(),
                          GestureDetector(
                            onTap: _selectAll,
                            child: const Text('Select all',
                                style: TextStyle(
                                    color: _accent,
                                    fontSize: 12)),
                          ),
                          const SizedBox(width: 12),
                          GestureDetector(
                            onTap: _deselectAll,
                            child: Text('Deselect all',
                                style: TextStyle(
                                    color: colors.textSubtle,
                                    fontSize: 12),
                            ),
                          ),
                        ],
                      ),
                      const SizedBox(height: 6),
                      Text(
                          'Grant only the permissions your application needs.',
                          style: TextStyle(
                              color: colors.textSubtle, fontSize: 12)),
                      const SizedBox(height: 12),

                      // Scope groups
                      ..._kScopeGroups.entries.map((entry) =>
                          _ScopeGroupRow(
                            group: entry.key,
                            scopes: entry.value,
                            selected: _scopes,
                            onToggleGroup: () => _toggleGroup(entry.key),
                            onToggleScope: _toggleScope,
                          )),
                      const SizedBox(height: 8),
                    ],
                  ),
                ),
              ),

              // Actions
              Padding(
                padding: const EdgeInsets.fromLTRB(20, 8, 20, 20),
                child: Row(
                  mainAxisAlignment: MainAxisAlignment.end,
                  children: [
                    TextButton(
                      onPressed: () => Navigator.of(context).pop(),
                      style: TextButton.styleFrom(
                        foregroundColor: colors.textMuted,
                        padding: const EdgeInsets.symmetric(
                            horizontal: 16, vertical: 10),
                      ),
                      child: const Text('Cancel',
                          style: TextStyle(fontSize: 13)),
                    ),
                    const SizedBox(width: 8),
                    FilledButton(
                      style: FilledButton.styleFrom(
                        backgroundColor: _accent,
                        padding: const EdgeInsets.symmetric(
                            horizontal: 20, vertical: 10),
                        shape: RoundedRectangleBorder(
                            borderRadius: BorderRadius.circular(8)),
                      ),
                      onPressed: _loading ? null : _submit,
                      child: _loading
                          ? const SizedBox(
                              width: 14,
                              height: 14,
                              child: CircularProgressIndicator(
                                  strokeWidth: 2, color: Colors.white),
                            )
                          : const Text('Create',
                              style: TextStyle(fontSize: 13)),
                    ),
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

class _ScopeGroupRow extends StatefulWidget {
  final String group;
  final List<String> scopes;
  final Set<String> selected;
  final VoidCallback onToggleGroup;
  final void Function(String) onToggleScope;

  const _ScopeGroupRow({
    required this.group,
    required this.scopes,
    required this.selected,
    required this.onToggleGroup,
    required this.onToggleScope,
  });

  @override
  State<_ScopeGroupRow> createState() => _ScopeGroupRowState();
}

class _ScopeGroupRowState extends State<_ScopeGroupRow> {
  bool _expanded = false;

  @override
  Widget build(BuildContext context) {
    final colors = consoleColors(context);
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
              padding:
                  const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
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
                    materialTapTargetSize: MaterialTapTargetSize.shrinkWrap,
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
                    style: TextStyle(
                        color: colors.textSubtle, fontSize: 12),
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
            ...widget.scopes.map((scope) => _ScopeCheckTile(
                  scope: scope,
                  checked: widget.selected.contains(scope),
                  onChanged: () => widget.onToggleScope(scope),
                )),
          ],
        ],
      ),
    );
  }
}

class _ScopeCheckTile extends StatelessWidget {
  final String scope;
  final bool checked;
  final VoidCallback onChanged;

  const _ScopeCheckTile({
    required this.scope,
    required this.checked,
    required this.onChanged,
  });

  @override
  Widget build(BuildContext context) {
    final colors = consoleColors(context);
    return InkWell(
      onTap: onChanged,
      child: Padding(
        padding:
            const EdgeInsets.only(left: 40, right: 12, top: 8, bottom: 8),
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

// =============================================================================
// Secret reveal field (shown once after creation)
// =============================================================================

class _SecretRevealField extends StatefulWidget {
  final String secret;
  const _SecretRevealField({required this.secret});

  @override
  State<_SecretRevealField> createState() => _SecretRevealFieldState();
}

class _SecretRevealFieldState extends State<_SecretRevealField> {
  bool _visible = true;

  @override
  Widget build(BuildContext context) {
    final colors = consoleColors(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Container(
          padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
          decoration: BoxDecoration(
            color: colors.fill,
            borderRadius: BorderRadius.circular(8),
            border: Border.all(color: colors.border),
          ),
          child: Row(
            children: [
              Expanded(
                child: SelectableText(
                  _visible ? widget.secret : '•' * 48,
                  style: TextStyle(
                    fontFamily: 'monospace',
                    fontSize: 12,
                    color: colors.textPrimary,
                  ),
                ),
              ),
              GestureDetector(
                onTap: () => setState(() => _visible = !_visible),
                child: Icon(
                    _visible ? LucideIcons.eyeOff : LucideIcons.eye,
                    size: 14,
                    color: colors.textSubtle),
              ),
              const SizedBox(width: 10),
              _CopyButton(text: widget.secret),
            ],
          ),
        ),
      ],
    );
  }
}

// =============================================================================
// Audit Log Tab
// =============================================================================

class _AuditLogTab extends ConsumerStatefulWidget {
  final String projectId;
  const _AuditLogTab({required this.projectId});

  @override
  ConsumerState<_AuditLogTab> createState() => _AuditLogTabState();
}

class _AuditLogTabState extends ConsumerState<_AuditLogTab> {
  final _searchCtrl = TextEditingController();
  int _page = 1;
  int _perPage = 25;
  String? _filterMethod;
  String? _filterResourceType;
  bool _loading = false;
  bool _hasError = false;
  Object? _error;
  List<Map<String, dynamic>> _rows = [];
  int _total = 0;

  @override
  void initState() {
    super.initState();
    _fetch();
  }

  @override
  void dispose() {
    _searchCtrl.dispose();
    super.dispose();
  }

  Future<void> _fetch() async {
    setState(() { _loading = true; _hasError = false; });
    try {
      final api = ref.read(apiClientProvider);
      final params = <String, dynamic>{
        'limit': _perPage,
        'offset': (_page - 1) * _perPage,
      };
      if (_filterMethod != null) params['method'] = _filterMethod;
      if (_filterResourceType != null) params['resourceType'] = _filterResourceType;
      final res = await api.get('/audit', params: params);
      final data = res.data as Map<String, dynamic>;
      if (mounted) {
        setState(() {
          _rows = List<Map<String, dynamic>>.from(data['logs'] ?? []);
          _total = (data['total'] as num?)?.toInt() ?? _rows.length;
          _loading = false;
        });
      }
    } catch (e) {
      if (mounted) setState(() { _loading = false; _hasError = true; _error = e; });
    }
  }

  void _applyFilters(Map<String, String?> filters) {
    setState(() {
      _filterMethod = filters['method'];
      _filterResourceType = filters['resourceType'];
      _page = 1;
    });
    _fetch();
  }

  @override
  Widget build(BuildContext context) {
    if (_hasError) return AppErrorState(error: _error!);
    if (_loading && _rows.isEmpty) {
      return const Center(child: CircularProgressIndicator());
    }

    return AppDataTable(
      columns: const [
        AppTableColumn(key: 'method',       label: 'Method',   flex: 1, sortable: false),
        AppTableColumn(key: 'path',         label: 'Path',     flex: 3, sortable: false),
        AppTableColumn(key: 'statusCode',   label: 'Status',   flex: 1, sortable: false),
        AppTableColumn(key: 'resourceType', label: 'Resource', flex: 2, sortable: false),
        AppTableColumn(key: 'action',       label: 'Action',   flex: 2, sortable: false, defaultVisible: false),
        AppTableColumn(key: 'userId',       label: 'User',     flex: 2, sortable: false, defaultVisible: false),
        AppTableColumn(key: 'ipAddress',    label: 'IP',       flex: 2, sortable: false, defaultVisible: false),
        AppTableColumn(key: r'$createdAt',  label: 'Time',     flex: 2, sortable: false),
      ],
      rows: _rows,
      getCellValue: (row, key) => switch (key) {
        'method'       => row['method'] as String? ?? '',
        'path'         => row['path'] as String? ?? '',
        'statusCode'   => '${(row['statusCode'] as num?)?.toInt() ?? 0}',
        'resourceType' => row['resourceType'] as String? ?? '',
        'action'       => row['action'] as String? ?? '',
        'userId'       => row['userId'] as String? ?? '',
        'ipAddress'    => row['ipAddress'] as String? ?? '',
        r'$createdAt'  => _fmtTs(row[r'$createdAt'] as String? ?? ''),
        _              => '',
      },
      cellBuilder: (row, key) {
        if (key == 'method') {
          return _MethodChip(method: row['method'] as String? ?? '');
        }
        if (key == 'statusCode') {
          return _StatusChip(status: (row['statusCode'] as num?)?.toInt() ?? 0);
        }
        return null;
      },
      filters: const [
        AppTableFilter(
          key: 'method',
          label: 'Method',
          options: ['GET', 'POST', 'PUT', 'PATCH', 'DELETE'],
        ),
        AppTableFilter(
          key: 'resourceType',
          label: 'Resource type',
          options: [
            'user', 'session', 'team', 'database', 'table', 'row',
            'bucket', 'file', 'function', 'workflow', 'deployment',
            'project', 'api_key',
          ],
        ),
      ],
      onFiltersChanged: _applyFilters,
      total: _total,
      perPage: _perPage,
      currentPage: _page,
      onPrev: () { setState(() => _page--); _fetch(); },
      onNext: () { setState(() => _page++); _fetch(); },
      onPerPageChanged: (v) { setState(() { _perPage = v; _page = 1; }); _fetch(); },
      itemLabel: 'entries',
      searchController: _searchCtrl,
      onSearch: () {},
      searchHint: 'Search not available server-side',
      createLabel: 'Audit Log',
      emptyIcon: LucideIcons.clipboardList,
      emptyTitle: 'No audit log entries yet',
      emptySubtitle: 'API activity on this project will appear here',
    );
  }
}

// ── Audit cell widgets ─────────────────────────────────────────────────────

class _MethodChip extends StatelessWidget {
  final String method;
  const _MethodChip({required this.method});

  @override
  Widget build(BuildContext context) {
    final color = switch (method.toUpperCase()) {
      'GET'           => const Color(0xFF3472A4),
      'POST'          => const Color(0xFF10B981),
      'PUT' || 'PATCH'=> const Color(0xFFF59E0B),
      'DELETE'        => const Color(0xFFEF4444),
      _               => const Color(0xFF6B7280),
    };
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
      decoration: BoxDecoration(
        color: color.withAlpha(38),
        borderRadius: BorderRadius.circular(4),
      ),
      child: Text(method.toUpperCase(),
          style: TextStyle(
              color: color, fontSize: 11, fontWeight: FontWeight.w600)),
    );
  }
}

class _StatusChip extends StatelessWidget {
  final int status;
  const _StatusChip({required this.status});

  @override
  Widget build(BuildContext context) {
    final color = status >= 500
        ? const Color(0xFFEF4444)
        : status >= 400
            ? const Color(0xFFF59E0B)
            : const Color(0xFF10B981);
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
      decoration: BoxDecoration(
        color: color.withAlpha(38),
        borderRadius: BorderRadius.circular(4),
      ),
      child: Text('$status',
          style: TextStyle(
              color: color, fontSize: 11, fontWeight: FontWeight.w600)),
    );
  }
}

String _fmtTs(String iso) {
  if (iso.isEmpty) return '';
  try {
    final dt = DateTime.parse(iso).toLocal();
    return '${dt.year}-${dt.month.toString().padLeft(2, '0')}-${dt.day.toString().padLeft(2, '0')} '
        '${dt.hour.toString().padLeft(2, '0')}:${dt.minute.toString().padLeft(2, '0')}';
  } catch (_) {
    return iso;
  }
}

// =============================================================================
// Shared copy button with green-tick feedback
// =============================================================================

class _CopyButton extends StatefulWidget {
  final String text;
  final double size;
  const _CopyButton({required this.text, this.size = 14});

  @override
  State<_CopyButton> createState() => _CopyButtonState();
}

class _CopyButtonState extends State<_CopyButton> {
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
        size: widget.size,
        color: _copied ? const Color(0xFF10B981) : consoleColors(context).textSubtle,
      ),
    );
  }
}

// =============================================================================
// Secret prefix hint cell (table)
// =============================================================================

class _PrefixCell extends StatelessWidget {
  final String prefix;
  const _PrefixCell({required this.prefix});

  @override
  Widget build(BuildContext context) {
    final colors = consoleColors(context);
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        Text('$prefix···',
            style: TextStyle(
                color: colors.textSubtle,
                fontSize: 12,
                fontFamily: 'monospace')),
        const SizedBox(width: 6),
        _CopyButton(text: prefix, size: 12),
      ],
    );
  }
}

