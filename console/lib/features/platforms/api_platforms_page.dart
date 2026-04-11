import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:lucide_icons/lucide_icons.dart';
import '../../core/api/client.dart';
import '../../core/providers/project_provider.dart';
import '../../core/theme/console_colors.dart';
import '../../core/utils/url_utils.dart';
import '../../core/widgets/app_dialog.dart';
import '../../core/widgets/app_empty_state.dart';
import '../../core/widgets/app_error_state.dart';

const _accent = Color(0xFF3472A4);

final _platformsProvider =
    FutureProvider.family<List<Map<String, dynamic>>, String>((ref, projectId) async {
  final api = ref.read(apiClientProvider);
  final res = await api.get('/projects/$projectId/platforms');
  final data = res.data as Map<String, dynamic>;
  return List<Map<String, dynamic>>.from(data['platforms'] ?? []);
});

class ApiPlatformsPage extends ConsumerStatefulWidget {
  const ApiPlatformsPage({super.key});

  @override
  ConsumerState<ApiPlatformsPage> createState() => _ApiPlatformsPageState();
}

class _ApiPlatformsPageState extends ConsumerState<ApiPlatformsPage> {
  final _searchCtrl = TextEditingController();

  @override
  void dispose() {
    _searchCtrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final cs = consoleColors(context);
    final projectId = ref.watch(currentProjectProvider) ?? '';
    final platformsAsync = ref.watch(_platformsProvider(projectId));

    return Scaffold(
      backgroundColor: cs.background,
      body: Padding(
        padding: EdgeInsets.symmetric(horizontal: pageHPad(context)),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const SizedBox(height: 32),
            Text('Platforms',
                style: TextStyle(
                    color: cs.textPrimary,
                    fontSize: 22,
                    fontWeight: FontWeight.w600)),
            const SizedBox(height: 4),
            Text('Register platforms to restrict API access to known apps.',
                style: TextStyle(color: cs.textSecondary, fontSize: 13)),
            const SizedBox(height: 24),
            platformsAsync.when(
              loading: () => const Expanded(
                  child: Center(child: CircularProgressIndicator())),
              error: (e, _) => Expanded(
                  child: AppErrorState(
                      error: e,
                      onRetry: () => ref.invalidate(_platformsProvider(projectId)))),
              data: (platforms) {
                final query = _searchCtrl.text.toLowerCase();
                final filtered = query.isEmpty
                    ? platforms
                    : platforms.where((p) {
                        final name = (p['name'] as String? ?? '').toLowerCase();
                        final type = (p['type'] as String? ?? '').toLowerCase();
                        return name.contains(query) || type.contains(query);
                      }).toList();

                return Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Row(children: [
                        SizedBox(
                          width: 280,
                          child: TextField(
                            controller: _searchCtrl,
                            onChanged: (_) => setState(() {}),
                            style: TextStyle(fontSize: 13, color: cs.textPrimary),
                            decoration: InputDecoration(
                              hintText: 'Search platforms...',
                              hintStyle:
                                  TextStyle(color: cs.textSubtle, fontSize: 13),
                              prefixIcon: Padding(
                                padding: const EdgeInsets.only(left: 10, right: 6),
                                child: Icon(LucideIcons.search,
                                    size: 15, color: cs.textSubtle),
                              ),
                              prefixIconConstraints:
                                  const BoxConstraints(minWidth: 32, minHeight: 0),
                              filled: true,
                              fillColor: cs.fieldFill,
                              isDense: true,
                              contentPadding: const EdgeInsets.symmetric(
                                  vertical: 10, horizontal: 12),
                              border: OutlineInputBorder(
                                  borderRadius: BorderRadius.circular(8),
                                  borderSide: BorderSide(color: cs.fieldBorder)),
                              enabledBorder: OutlineInputBorder(
                                  borderRadius: BorderRadius.circular(8),
                                  borderSide: BorderSide(color: cs.fieldBorder)),
                              focusedBorder: OutlineInputBorder(
                                  borderRadius: BorderRadius.circular(8),
                                  borderSide: const BorderSide(color: _accent)),
                            ),
                          ),
                        ),
                        const SizedBox(width: 12),
                        Text(
                          '${platforms.length} platform${platforms.length == 1 ? '' : 's'}',
                          style:
                              TextStyle(color: cs.textSecondary, fontSize: 13),
                        ),
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
                      ]),
                      const SizedBox(height: 16),
                      if (filtered.isEmpty)
                        Expanded(
                          child: AppEmptyState(
                            icon: LucideIcons.smartphone,
                            title: 'No platforms registered',
                            subtitle:
                                'Register platforms to restrict API access to known apps.',
                            actionLabel: 'Add platform',
                            onAction: () => _showAddPlatformDialog(projectId),
                          ),
                        )
                      else
                        Expanded(
                          child: ListView(
                            padding: EdgeInsets.zero,
                            children: [
                              ...filtered.map((p) => _PlatformCard(
                                    key: ValueKey(p[r'$id']),
                                    platform: p,
                                    onDelete: () => _deletePlatform(
                                        projectId, p[r'$id'] as String),
                                  )),
                              const SizedBox(height: 16),
                            ],
                          ),
                        ),
                    ],
                  ),
                );
              },
            ),
          ],
        ),
      ),
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
                  runSpacing: 8,
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
            if (mounted) Navigator.of(context, rootNavigator: true).pop();
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
}

// ── Platform card ─────────────────────────────────────────────────────────────

class _PlatformCard extends StatelessWidget {
  final Map<String, dynamic> platform;
  final VoidCallback onDelete;

  const _PlatformCard({super.key, required this.platform, required this.onDelete});

  IconData _typeIcon(String type) => switch (type) {
    'web' => LucideIcons.globe,
    'flutter-ios' || 'apple-ios' || 'flutter-android' || 'android' =>
        LucideIcons.smartphone,
    _ => LucideIcons.monitor,
  };

  String _typeLabel(String type) => switch (type) {
    'web'              => 'Web',
    'flutter-ios'      => 'Flutter iOS',
    'flutter-android'  => 'Flutter Android',
    'apple-ios'        => 'Apple iOS',
    'android'          => 'Android',
    _                  => type,
  };

  @override
  Widget build(BuildContext context) {
    final colors = consoleColors(context);
    final name     = platform['name']     as String? ?? 'Unnamed';
    final type     = platform['type']     as String? ?? '';
    final hostname = platform['hostname'] as String? ?? '';

    return Container(
      margin: const EdgeInsets.only(bottom: 8),
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: colors.surface,
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: colors.border),
      ),
      child: Row(children: [
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
              Row(children: [
                Container(
                  padding:
                      const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
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
              ]),
            ],
          ),
        ),
        GestureDetector(
          onTap: onDelete,
          child: Icon(LucideIcons.trash2, size: 14, color: colors.textSubtle),
        ),
      ]),
    );
  }
}

// ── Platform type chip ────────────────────────────────────────────────────────

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
          border: Border.all(
              color: selected ? _accent : colors.fieldBorder),
        ),
        child: Row(mainAxisSize: MainAxisSize.min, children: [
          Icon(icon,
              size: 14,
              color: selected ? _accent : colors.textSecondary),
          const SizedBox(width: 6),
          Text(label,
              style: TextStyle(
                  color: selected ? colors.textPrimary : colors.textSecondary,
                  fontSize: 12)),
        ]),
      ),
    );
  }
}
