import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:lucide_icons/lucide_icons.dart';
import '../../core/api/client.dart';
import '../../core/providers/project_provider.dart';
import '../../core/widgets/app_dialog.dart';
import '../../core/widgets/page_tabs.dart';

// --- Constants ---------------------------------------------------------------

const _bgColor = Color(0xFF0B0B0F);
const _cardColor = Color(0xFF16171B);
const _accent = Color(0xFF3472A4);
const _dimText = Color(0x80FFFFFF);
const _subtleText = Color(0x40FFFFFF);
const _border = Color(0x0DFFFFFF);
const _fieldFill = Color(0x0AFFFFFF);
const _fieldBorder = Color(0x1AFFFFFF);
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
  int _tabIndex = 0;

  // General tab
  final _nameCtrl = TextEditingController();
  final _descCtrl = TextEditingController();
  bool _generalDirty = false;
  bool _generalSaving = false;

  // Platform search
  final _platformSearchCtrl = TextEditingController();

  // Webhook search
  final _webhookSearchCtrl = TextEditingController();

  @override
  void dispose() {
    _nameCtrl.dispose();
    _descCtrl.dispose();
    _platformSearchCtrl.dispose();
    _webhookSearchCtrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final routerState = GoRouterState.of(context);
    final projectId = routerState.pathParameters['projectId'];
    if (projectId == null) {
      return const Center(
          child: Text('No project selected',
              style: TextStyle(color: _dimText)));
    }

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
            // Title
            const Text('Settings',
                style: TextStyle(
                    color: Colors.white,
                    fontSize: 22,
                    fontWeight: FontWeight.w600)),
            const SizedBox(height: 4),
            Text('Manage your project configuration',
                style: TextStyle(
                    color: Colors.white.withOpacity(0.4), fontSize: 14)),
            const SizedBox(height: 24),

            // Tabs
            PageTabs(
              tabs: const [
                'General',
                'API Keys',
                'Platforms',
                'Webhooks',
              ],
              selected: _tabIndex,
              onChanged: (i) => setState(() => _tabIndex = i),
            ),
            const SizedBox(height: 24),

            // Tab body
            Expanded(
              child: SingleChildScrollView(
                child: _buildTabBody(projectId),
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildTabBody(String projectId) {
    switch (_tabIndex) {
      case 0:
        return _buildGeneralTab(projectId);
      case 1:
        return _buildApiKeysTab(projectId);
      case 2:
        return _buildPlatformsTab(projectId);
      case 3:
        return _buildWebhooksTab(projectId);
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
      error: (e, _) => Center(
          child: Text('Error loading project: $e',
              style: const TextStyle(color: _red))),
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
                          color: Colors.white.withOpacity(0.3),
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
                                  color: Colors.white.withOpacity(0.4),
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
    final confirmed = await showAppDialog<bool>(
      context: context,
      title: 'Delete project',
      content: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            'This will permanently delete the project and all its data including databases, storage, functions, and deployments.',
            style: TextStyle(color: Colors.white.withOpacity(0.6)),
          ),
          const SizedBox(height: 16),
          Text('Type the project ID to confirm:',
              style: TextStyle(
                  color: Colors.white.withOpacity(0.5), fontSize: 13)),
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

    return keysAsync.when(
      loading: () => const Center(child: CircularProgressIndicator()),
      error: (e, _) => Center(
          child: Text('Error: $e', style: const TextStyle(color: _red))),
      data: (keys) {
        return Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            // Header
            Row(
              children: [
                Text('${keys.length} API key${keys.length == 1 ? '' : 's'}',
                    style: const TextStyle(color: _dimText, fontSize: 13)),
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
                  label: const Text('Create API key',
                      style: TextStyle(fontSize: 12)),
                  onPressed: () => _showCreateKeyDialog(projectId),
                ),
              ],
            ),
            const SizedBox(height: 16),

            // Keys list
            if (keys.isEmpty)
              _EmptyState(
                icon: LucideIcons.key,
                title: 'No API keys',
                subtitle:
                    'Create an API key to authenticate server-side requests',
                actionLabel: 'Create API key',
                onAction: () => _showCreateKeyDialog(projectId),
              )
            else
              ...keys.map((k) => _ApiKeyCard(
                    key: ValueKey(k['\$id']),
                    keyData: k,
                    onDelete: () => _deleteKey(projectId, k['\$id']),
                  )),
            const SizedBox(height: 40),
          ],
        );
      },
    );
  }

  void _showCreateKeyDialog(String projectId) {
    final nameCtrl = TextEditingController();
    showAppDialog(
      context: context,
      title: 'Create API key',
      subtitle: 'API keys authenticate server-side SDK requests',
      content: AppDialogField(
        controller: nameCtrl,
        label: 'Key name',
        hint: 'e.g. Production key',
        autofocus: true,
      ),
      actions: [
        const AppDialogCancel(),
        AppDialogAction(
          label: 'Create',
          onTap: () async {
            if (nameCtrl.text.trim().isEmpty) return;
            final api = ref.read(apiClientProvider);
            await api.post('/projects/$projectId/keys', data: {
              'name': nameCtrl.text.trim(),
              'scopes': <String>[],
            });
            if (mounted) {
              Navigator.of(context, rootNavigator: true).pop();
            }
            ref.invalidate(apiKeysProvider(projectId));
          },
        ),
      ],
    );
  }

  Future<void> _deleteKey(String projectId, String keyId) async {
    final confirmed = await showAppDialog<bool>(
      context: context,
      title: 'Delete API key',
      content: Text(
          'Any applications using this key will lose access immediately.',
          style: TextStyle(color: Colors.white.withOpacity(0.6))),
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

  Widget _buildPlatformsTab(String projectId) {
    final platformsAsync = ref.watch(_platformsProvider(projectId));

    return platformsAsync.when(
      loading: () => const Center(child: CircularProgressIndicator()),
      error: (e, _) => Center(
          child: Text('Error: $e', style: const TextStyle(color: _red))),
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
                    style:
                        const TextStyle(fontSize: 13, color: Colors.white),
                    decoration: InputDecoration(
                      hintText: 'Search platforms...',
                      hintStyle:
                          const TextStyle(color: _subtleText, fontSize: 13),
                      prefixIcon: const Padding(
                        padding: EdgeInsets.only(left: 10, right: 6),
                        child: Icon(Icons.search,
                            size: 16, color: _subtleText),
                      ),
                      prefixIconConstraints:
                          const BoxConstraints(minWidth: 32, minHeight: 0),
                      filled: true,
                      fillColor: _fieldFill,
                      isDense: true,
                      contentPadding: const EdgeInsets.symmetric(
                          vertical: 10, horizontal: 12),
                      border: OutlineInputBorder(
                          borderRadius: BorderRadius.circular(8),
                          borderSide: const BorderSide(color: _fieldBorder)),
                      enabledBorder: OutlineInputBorder(
                          borderRadius: BorderRadius.circular(8),
                          borderSide: const BorderSide(color: _fieldBorder)),
                      focusedBorder: OutlineInputBorder(
                          borderRadius: BorderRadius.circular(8),
                          borderSide: const BorderSide(color: _accent)),
                    ),
                  ),
                ),
                const SizedBox(width: 12),
                Text('${platforms.length} platform${platforms.length == 1 ? '' : 's'}',
                    style: const TextStyle(color: _dimText, fontSize: 13)),
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
                        color: Colors.white.withOpacity(0.5),
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
    final confirmed = await showAppDialog<bool>(
      context: context,
      title: 'Remove platform',
      content: Text('This platform will no longer have API access.',
          style: TextStyle(color: Colors.white.withOpacity(0.6))),
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
    final webhooksAsync = ref.watch(_webhooksProvider(projectId));

    return webhooksAsync.when(
      loading: () => const Center(child: CircularProgressIndicator()),
      error: (e, _) => Center(
          child: Text('Error: $e', style: const TextStyle(color: _red))),
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
                    style:
                        const TextStyle(fontSize: 13, color: Colors.white),
                    decoration: InputDecoration(
                      hintText: 'Search webhooks...',
                      hintStyle:
                          const TextStyle(color: _subtleText, fontSize: 13),
                      prefixIcon: const Padding(
                        padding: EdgeInsets.only(left: 10, right: 6),
                        child: Icon(Icons.search,
                            size: 16, color: _subtleText),
                      ),
                      prefixIconConstraints:
                          const BoxConstraints(minWidth: 32, minHeight: 0),
                      filled: true,
                      fillColor: _fieldFill,
                      isDense: true,
                      contentPadding: const EdgeInsets.symmetric(
                          vertical: 10, horizontal: 12),
                      border: OutlineInputBorder(
                          borderRadius: BorderRadius.circular(8),
                          borderSide: const BorderSide(color: _fieldBorder)),
                      enabledBorder: OutlineInputBorder(
                          borderRadius: BorderRadius.circular(8),
                          borderSide: const BorderSide(color: _fieldBorder)),
                      focusedBorder: OutlineInputBorder(
                          borderRadius: BorderRadius.circular(8),
                          borderSide: const BorderSide(color: _accent)),
                    ),
                  ),
                ),
                const SizedBox(width: 12),
                Text(
                    '${webhooks.length} webhook${webhooks.length == 1 ? '' : 's'}',
                    style: const TextStyle(color: _dimText, fontSize: 13)),
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
    final nameCtrl = TextEditingController();
    final urlCtrl = TextEditingController();
    final Set<String> selectedEvents = {};

    const events = [
      'databases.*',
      'collections.*',
      'documents.*',
      'storage.*',
      'users.*',
      'teams.*',
      'functions.*',
      'deployments.*',
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
                    color: Colors.white.withOpacity(0.5),
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
                          color: selected ? Colors.white : _dimText,
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
                  backgroundColor: _fieldFill,
                  selectedColor: _accent.withOpacity(0.3),
                  checkmarkColor: Colors.white,
                  side: BorderSide(
                      color: selected
                          ? _accent
                          : Colors.white.withOpacity(0.1)),
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
    final confirmed = await showAppDialog<bool>(
      context: context,
      title: 'Delete webhook',
      content: Text('This webhook will stop receiving notifications.',
          style: TextStyle(color: Colors.white.withOpacity(0.6))),
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
      final routerState = GoRouterState.of(context);
      final projectId = routerState.pathParameters['projectId'];
      if (projectId != null) {
        ref.invalidate(_webhooksProvider(projectId));
      }
    }
  }
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
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: _cardColor,
        borderRadius: BorderRadius.circular(8),
        border: Border.all(
            color: danger
                ? _red.withOpacity(0.3)
                : Colors.white.withOpacity(0.06)),
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
                            color: danger ? _red : Colors.white,
                            fontSize: 15,
                            fontWeight: FontWeight.w600)),
                    if (subtitle != null) ...[
                      const SizedBox(height: 4),
                      Text(subtitle!,
                          style: TextStyle(
                              color: Colors.white.withOpacity(0.4),
                              fontSize: 13)),
                    ],
                  ],
                ),
              ),
              if (trailing != null) trailing!,
            ],
          ),
          const SizedBox(height: 16),
          Container(
              height: 1, color: Colors.white.withOpacity(0.06)),
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
    return Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        SizedBox(
          width: 160,
          child: Padding(
            padding: const EdgeInsets.only(top: 10),
            child: Text(label,
                style: TextStyle(
                    color: Colors.white.withOpacity(0.5),
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
                    color: _fieldFill,
                    borderRadius: BorderRadius.circular(8),
                    border: Border.all(color: _fieldBorder),
                  ),
                  child: Row(
                    children: [
                      Expanded(
                        child: SelectableText(
                          value ?? controller?.text ?? '',
                          style: TextStyle(
                            color: Colors.white.withOpacity(0.6),
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
                              size: 14, color: Colors.white.withOpacity(0.4)),
                        ),
                    ],
                  ),
                )
              : TextField(
                  controller: controller,
                  onChanged: onChanged,
                  style: const TextStyle(
                      color: Colors.white, fontSize: 13),
                  decoration: InputDecoration(
                    hintText: hint,
                    hintStyle: TextStyle(
                        color: Colors.white.withOpacity(0.22),
                        fontSize: 13),
                    filled: true,
                    fillColor: _fieldFill,
                    isDense: true,
                    contentPadding: const EdgeInsets.symmetric(
                        horizontal: 12, vertical: 10),
                    border: OutlineInputBorder(
                        borderRadius: BorderRadius.circular(8),
                        borderSide: const BorderSide(color: _fieldBorder)),
                    enabledBorder: OutlineInputBorder(
                        borderRadius: BorderRadius.circular(8),
                        borderSide: const BorderSide(color: _fieldBorder)),
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
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 6),
      child: Row(
        children: [
          Icon(icon, size: 16, color: _dimText),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(label,
                    style: const TextStyle(
                        color: Colors.white,
                        fontSize: 13,
                        fontWeight: FontWeight.w500)),
                Text(description,
                    style: TextStyle(
                        color: Colors.white.withOpacity(0.35),
                        fontSize: 12)),
              ],
            ),
          ),
          Switch(
            value: enabled,
            onChanged: onChanged,
            activeColor: _accent,
          ),
        ],
      ),
    );
  }
}

class _ApiKeyCard extends StatefulWidget {
  final Map<String, dynamic> keyData;
  final VoidCallback onDelete;

  const _ApiKeyCard({super.key, required this.keyData, required this.onDelete});

  @override
  State<_ApiKeyCard> createState() => _ApiKeyCardState();
}

class _ApiKeyCardState extends State<_ApiKeyCard> {
  bool _showSecret = false;

  @override
  Widget build(BuildContext context) {
    final name = widget.keyData['name'] as String? ?? 'Unnamed';
    final id = widget.keyData['\$id'] as String? ?? '';
    final secret = widget.keyData['secret'] as String?;
    final createdAt = widget.keyData['createdAt'] as String?;

    return Container(
      margin: const EdgeInsets.only(bottom: 8),
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: _cardColor,
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: Colors.white.withOpacity(0.06)),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Icon(LucideIcons.key, size: 16, color: _accent),
              const SizedBox(width: 10),
              Expanded(
                child: Text(name,
                    style: const TextStyle(
                        color: Colors.white,
                        fontSize: 14,
                        fontWeight: FontWeight.w500)),
              ),
              if (createdAt != null)
                Text(createdAt,
                    style: const TextStyle(color: _subtleText, fontSize: 11)),
              const SizedBox(width: 12),
              GestureDetector(
                onTap: widget.onDelete,
                child: Icon(LucideIcons.trash2,
                    size: 14, color: Colors.white.withOpacity(0.3)),
              ),
            ],
          ),
          if (secret != null) ...[
            const SizedBox(height: 12),
            Container(
              padding:
                  const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
              decoration: BoxDecoration(
                color: Colors.white.withOpacity(0.03),
                borderRadius: BorderRadius.circular(6),
                border: Border.all(color: Colors.white.withOpacity(0.06)),
              ),
              child: Row(
                children: [
                  Expanded(
                    child: SelectableText(
                      _showSecret ? secret : '•' * 40,
                      style: TextStyle(
                        fontFamily: 'monospace',
                        fontSize: 12,
                        color: Colors.white.withOpacity(0.5),
                      ),
                    ),
                  ),
                  GestureDetector(
                    onTap: () => setState(() => _showSecret = !_showSecret),
                    child: Icon(
                      _showSecret ? LucideIcons.eyeOff : LucideIcons.eye,
                      size: 14,
                      color: Colors.white.withOpacity(0.3),
                    ),
                  ),
                  const SizedBox(width: 8),
                  GestureDetector(
                    onTap: () {
                      Clipboard.setData(ClipboardData(text: secret));
                      ScaffoldMessenger.of(context).showSnackBar(
                        const SnackBar(content: Text('Copied to clipboard')),
                      );
                    },
                    child: Icon(LucideIcons.copy,
                        size: 14, color: Colors.white.withOpacity(0.3)),
                  ),
                ],
              ),
            ),
          ],
          const SizedBox(height: 8),
          Text('ID: $id',
              style: const TextStyle(color: _subtleText, fontSize: 11)),
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
    final name = platform['name'] as String? ?? 'Unnamed';
    final type = platform['type'] as String? ?? '';
    final hostname = platform['hostname'] as String? ?? '';

    return Container(
      margin: const EdgeInsets.only(bottom: 8),
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: _cardColor,
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: Colors.white.withOpacity(0.06)),
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
            child: Icon(_typeIcon(type), size: 18, color: _accent),
          ),
          const SizedBox(width: 14),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(name,
                    style: const TextStyle(
                        color: Colors.white,
                        fontSize: 14,
                        fontWeight: FontWeight.w500)),
                const SizedBox(height: 2),
                Row(
                  children: [
                    Container(
                      padding: const EdgeInsets.symmetric(
                          horizontal: 6, vertical: 2),
                      decoration: BoxDecoration(
                        color: _accent.withOpacity(0.15),
                        borderRadius: BorderRadius.circular(4),
                      ),
                      child: Text(_typeLabel(type),
                          style: TextStyle(
                              color: _accent,
                              fontSize: 11,
                              fontWeight: FontWeight.w500)),
                    ),
                    if (hostname.isNotEmpty) ...[
                      const SizedBox(width: 8),
                      Text(hostname,
                          style: const TextStyle(
                              color: _dimText,
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
                size: 14, color: Colors.white.withOpacity(0.3)),
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
    final name = webhook['name'] as String? ?? 'Unnamed';
    final url = webhook['url'] as String? ?? '';
    final enabled = webhook['enabled'] as bool? ?? true;
    final events = List<String>.from(webhook['events'] ?? []);

    return Container(
      margin: const EdgeInsets.only(bottom: 8),
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: _cardColor,
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: Colors.white.withOpacity(0.06)),
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
                  color: _accent.withOpacity(0.1),
                  borderRadius: BorderRadius.circular(8),
                ),
                child:
                    Icon(LucideIcons.webhook, size: 18, color: _accent),
              ),
              const SizedBox(width: 14),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Row(
                      children: [
                        Text(name,
                            style: const TextStyle(
                                color: Colors.white,
                                fontSize: 14,
                                fontWeight: FontWeight.w500)),
                        const SizedBox(width: 8),
                        Container(
                          padding: const EdgeInsets.symmetric(
                              horizontal: 6, vertical: 2),
                          decoration: BoxDecoration(
                            color: (enabled ? _green : _subtleText)
                                .withOpacity(0.15),
                            borderRadius: BorderRadius.circular(4),
                          ),
                          child: Text(
                            enabled ? 'Active' : 'Disabled',
                            style: TextStyle(
                                color: enabled ? _green : _subtleText,
                                fontSize: 11,
                                fontWeight: FontWeight.w500),
                          ),
                        ),
                      ],
                    ),
                    const SizedBox(height: 2),
                    Text(url,
                        style: const TextStyle(
                            color: _dimText,
                            fontSize: 12,
                            fontFamily: 'monospace'),
                        overflow: TextOverflow.ellipsis),
                  ],
                ),
              ),
              GestureDetector(
                onTap: onDelete,
                child: Icon(LucideIcons.trash2,
                    size: 14, color: Colors.white.withOpacity(0.3)),
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
                          color: Colors.white.withOpacity(0.04),
                          borderRadius: BorderRadius.circular(4),
                          border: Border.all(
                              color: Colors.white.withOpacity(0.08)),
                        ),
                        child: Text(e,
                            style: const TextStyle(
                                color: _dimText,
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
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.symmetric(vertical: 60),
      child: Column(
        children: [
          Container(
            width: 48,
            height: 48,
            decoration: BoxDecoration(
              color: Colors.white.withOpacity(0.04),
              borderRadius: BorderRadius.circular(12),
            ),
            child: Icon(icon, size: 22, color: _subtleText),
          ),
          const SizedBox(height: 16),
          Text(title,
              style: const TextStyle(
                  color: Colors.white,
                  fontSize: 15,
                  fontWeight: FontWeight.w500)),
          const SizedBox(height: 6),
          Text(subtitle,
              style: const TextStyle(color: _dimText, fontSize: 13),
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
    return GestureDetector(
      onTap: onTap,
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
        decoration: BoxDecoration(
          color: selected ? _accent.withOpacity(0.15) : _fieldFill,
          borderRadius: BorderRadius.circular(8),
          border: Border.all(
              color: selected ? _accent : Colors.white.withOpacity(0.1)),
        ),
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(icon,
                size: 14,
                color: selected ? _accent : _dimText),
            const SizedBox(width: 6),
            Text(label,
                style: TextStyle(
                    color: selected ? Colors.white : _dimText,
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
    return TextField(
      controller: _ctrl,
      style: const TextStyle(color: Colors.white, fontSize: 13),
      decoration: InputDecoration(
        hintText: widget.projectId,
        hintStyle:
            TextStyle(color: Colors.white.withOpacity(0.22), fontSize: 13),
        filled: true,
        fillColor: _fieldFill,
        isDense: true,
        contentPadding:
            const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
        border: OutlineInputBorder(
            borderRadius: BorderRadius.circular(8),
            borderSide: const BorderSide(color: _fieldBorder)),
        enabledBorder: OutlineInputBorder(
            borderRadius: BorderRadius.circular(8),
            borderSide: const BorderSide(color: _fieldBorder)),
        focusedBorder: OutlineInputBorder(
            borderRadius: BorderRadius.circular(8),
            borderSide: const BorderSide(color: _red)),
      ),
    );
  }
}
