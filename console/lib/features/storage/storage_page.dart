import 'dart:math' as math;
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:lucide_icons_flutter/lucide_icons.dart';
import '../../core/api/client.dart';
import '../../core/theme/console_colors.dart';
import '../../core/utils/url_utils.dart';
import '../../core/widgets/app_data_table.dart';
import '../../core/widgets/app_dialog.dart';
import '../../core/widgets/id_text.dart';
import '../../core/widgets/page_tabs.dart';
import '../../core/widgets/app_error_state.dart';

// --- Constants ---------------------------------------------------------------

const _accent = Color(0xFF3472A4);
const _red = Color(0xFFEF4444);

// --- Providers ---------------------------------------------------------------

final _bucketSearchProvider = StateProvider<String>((ref) => '');
final _bucketPerPageProvider = StateProvider<int>((ref) => 12);
final _bucketPageProvider = StateProvider<int>((ref) => 1);

final bucketsProvider = FutureProvider<Map<String, dynamic>>((ref) async {
  final api = ref.read(apiClientProvider);
  final search = ref.watch(_bucketSearchProvider);
  final limit = ref.watch(_bucketPerPageProvider);
  final page = ref.watch(_bucketPageProvider);
  final offset = (page - 1) * limit;
  final params = <String, dynamic>{'limit': limit, 'offset': offset};
  if (search.isNotEmpty) params['search'] = search;
  final res = await api.get('/storage/buckets', params: params);
  return res.data as Map<String, dynamic>;
});

final _filesProvider =
    FutureProvider.family<Map<String, dynamic>, String>((ref, bucketId) async {
  final api = ref.read(apiClientProvider);
  final res =
      await api.get('/storage/buckets/$bucketId/files', params: {'limit': 100});
  return res.data as Map<String, dynamic>;
});

final _bucketDetailProvider =
    FutureProvider.family<Map<String, dynamic>, String>((ref, bucketId) async {
  final api = ref.read(apiClientProvider);
  final res = await api.get('/storage/buckets/$bucketId');
  return res.data as Map<String, dynamic>;
});

// --- Page --------------------------------------------------------------------

class StoragePage extends ConsumerStatefulWidget {
  const StoragePage({super.key});

  @override
  ConsumerState<StoragePage> createState() => _StoragePageState();
}

class _StoragePageState extends ConsumerState<StoragePage> {
  final _searchCtrl = TextEditingController();
  int _topTab = 0;
  String? _selectedBucketId;
  String? _selectedFileId;

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    final urlPage = pageFromQuery(context);
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted) return;
      if (ref.read(_bucketPageProvider) != urlPage) {
        ref.read(_bucketPageProvider.notifier).state = urlPage;
      }
    });
  }

  @override
  void dispose() {
    _searchCtrl.dispose();
    super.dispose();
  }

  void _doSearch() {
    ref.read(_bucketSearchProvider.notifier).state = _searchCtrl.text.trim();
    ref.read(_bucketPageProvider.notifier).state = 1;
  }

  @override
  Widget build(BuildContext context) {
    if (_selectedFileId != null && _selectedBucketId != null) {
      return _FileDetailView(
        bucketId: _selectedBucketId!,
        fileId: _selectedFileId!,
        onBack: () => setState(() => _selectedFileId = null),
      );
    }
    if (_selectedBucketId != null) {
      return _BucketDetailView(
        bucketId: _selectedBucketId!,
        onBack: () => setState(() => _selectedBucketId = null),
        onFileSelect: (fileId) =>
            setState(() => _selectedFileId = fileId),
      );
    }
    return _buildBucketsList();
  }

  Widget _buildBucketsList() {
    final cs = consoleColors(context);
    final bucketsAsync = ref.watch(bucketsProvider);
    final perPage = ref.watch(_bucketPerPageProvider);
    final currentPage = ref.watch(_bucketPageProvider);
    final total =
        bucketsAsync.whenOrNull(data: (d) => d['total'] as int? ?? 0) ?? 0;
    final buckets = bucketsAsync.valueOrNull == null
        ? <Map<String, dynamic>>[]
        : List<Map<String, dynamic>>.from(
            bucketsAsync.valueOrNull!['buckets'] ?? []);

    return Scaffold(
      backgroundColor: cs.background,
      body: Padding(
        padding: EdgeInsets.symmetric(
          horizontal: pageHPad(context),
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const SizedBox(height: 32),
            Text('Storage',
                style: TextStyle(
                    color: cs.textPrimary,
                    fontSize: 22,
                    fontWeight: FontWeight.w600)),
            const SizedBox(height: 4),
            Text('Store and serve files with buckets, image transforms and access policies',
                style: TextStyle(color: cs.textSecondary, fontSize: 13)),
            const SizedBox(height: 20),
            PageTabs(
              tabs: const ['Buckets', 'Usage'],
              selected: _topTab,
              onChanged: (i) => setState(() => _topTab = i),
            ),
            const SizedBox(height: 20),
            if (_topTab == 1) ...[
              Expanded(child: _buildUsageTab()),
            ] else if (bucketsAsync.isLoading)
              const Expanded(
                  child: Center(child: CircularProgressIndicator()))
            else if (bucketsAsync.hasError)
              Expanded(child: AppErrorState(error: bucketsAsync.error!))
            else
              Expanded(
                child: AppDataTable(
                  columns: const [
                    AppTableColumn(key: r'$id',      label: 'Bucket ID', flex: 3),
                    AppTableColumn(key: 'name',       label: 'Name',      flex: 3),
                    AppTableColumn(key: 'createdAt',  label: 'Created',   flex: 2),
                    AppTableColumn(key: 'updatedAt',  label: 'Updated',   flex: 2),
                  ],
                  rows: buckets,
                  getCellValue: (row, key) => switch (key) {
                    r'$id'       => row[r'$id'] as String? ?? '',
                    'name'       => row['name'] as String? ?? '',
                    'createdAt'  => _fmtDate(row['createdAt'] ?? row[r'$createdAt']),
                    'updatedAt'  => _fmtDate(row['updatedAt'] ?? row[r'$updatedAt']),
                    _            => '',
                  },
                  getRowIcon: (_) => LucideIcons.folderClosed,
                  onRowTap: (row) =>
                      setState(() => _selectedBucketId = row[r'$id'] as String?),
                  onDeleteRow: (row) => _deleteBucket(row[r'$id'] as String),
                  createLabel: 'Create bucket',
                  onCreateTap: _showCreateBucketDialog,
                  total: total,
                  perPage: perPage,
                  currentPage: currentPage,
                  onPrev: () {
                    final p = currentPage - 1;
                    ref.read(_bucketPageProvider.notifier).state = p;
                    context.go(withQuery(context, {'page': '$p'}));
                  },
                  onNext: () {
                    final p = currentPage + 1;
                    ref.read(_bucketPageProvider.notifier).state = p;
                    context.go(withQuery(context, {'page': '$p'}));
                  },
                  onPerPageChanged: (v) {
                    ref.read(_bucketPerPageProvider.notifier).state = v;
                    ref.read(_bucketPageProvider.notifier).state = 1;
                  },
                  itemLabel: 'Buckets',
                  searchController: _searchCtrl,
                  onSearch: _doSearch,
                  emptyIcon: LucideIcons.folderClosed,
                  emptyTitle: 'No buckets',
                  emptySubtitle: 'Create a bucket to start storing files',
                  gridCardBuilder: (row) => _BucketGridCard(
                    bucket: row,
                    onTap: () => setState(
                        () => _selectedBucketId = row[r'$id'] as String?),
                  ),
                ),
              ),
          ],
        ),
      ),
    );
  }

  Widget _buildUsageTab() {
    final cs = consoleColors(context);
    return SingleChildScrollView(
      padding: const EdgeInsets.only(bottom: 32),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text('Usage',
              style: TextStyle(color: cs.textPrimary, fontSize: 16, fontWeight: FontWeight.w600)),
          const SizedBox(height: 4),
          Text('Storage activity for the past 30 days.',
              style: TextStyle(color: cs.textSecondary, fontSize: 13)),
          const SizedBox(height: 24),
          Row(children: [
            _storageStatCard(cs, 'Total buckets', '—', LucideIcons.folderClosed),
            const SizedBox(width: 12),
            _storageStatCard(cs, 'Total files', '—', LucideIcons.file),
            const SizedBox(width: 12),
            _storageStatCard(cs, 'Storage used', '—', LucideIcons.hardDrive),
          ]),
          const SizedBox(height: 24),
          Container(
            height: 200,
            decoration: BoxDecoration(
              color: cs.surface,
              borderRadius: BorderRadius.circular(8),
              border: Border.all(color: cs.border),
            ),
            child: Center(
              child: Text('Usage charts coming soon',
                  style: TextStyle(color: cs.textSubtle, fontSize: 13)),
            ),
          ),
        ],
      ),
    );
  }

  Widget _storageStatCard(ConsoleColors cs, String label, String value, IconData icon) {
    return Expanded(
      child: Container(
        padding: const EdgeInsets.all(20),
        decoration: BoxDecoration(
          color: cs.surface,
          borderRadius: BorderRadius.circular(8),
          border: Border.all(color: cs.border),
        ),
        child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
          Icon(icon, size: 16, color: cs.textSecondary),
          const SizedBox(height: 12),
          Text(value,
              style: TextStyle(color: cs.textPrimary, fontSize: 24, fontWeight: FontWeight.w700)),
          const SizedBox(height: 4),
          Text(label, style: TextStyle(color: cs.textSecondary, fontSize: 12)),
        ]),
      ),
    );
  }

  void _showCreateBucketDialog() {
    final nameCtrl = TextEditingController();
    showAppDialog(
      context: context,
      title: 'Create bucket',
      subtitle: 'Storage buckets organize your files',
      content: AppDialogField(
        controller: nameCtrl,
        label: 'Bucket name',
        hint: 'e.g. user-avatars',
        autofocus: true,
      ),
      actions: [
        const AppDialogCancel(),
        AppDialogAction(
          label: 'Create',
          onTap: () async {
            if (nameCtrl.text.trim().isEmpty) return;
            final api = ref.read(apiClientProvider);
            await api.post('/storage/buckets', data: {
              'bucketId': 'unique()',
              'name': nameCtrl.text.trim(),
              'permissions': <String>[],
              'allowedFileExtensions': <String>[],
            });
            if (mounted) Navigator.of(context, rootNavigator: true).pop();
            ref.invalidate(bucketsProvider);
          },
        ),
      ],
    );
  }

  Future<void> _deleteBucket(String id) async {
    final confirmed = await showAppDialog<bool>(
      context: context,
      title: 'Delete bucket',
      content: Text('All files in this bucket will be permanently deleted.',
          style: TextStyle(color: consoleColors(context).textSecondary)),
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
      await ref.read(apiClientProvider).delete('/storage/buckets/$id');
      ref.invalidate(bucketsProvider);
    }
  }
}

// =============================================================================
// Shared date formatter
// =============================================================================

String _fmtDate(dynamic raw) {
  if (raw == null) return '—';
  try {
    final dt = raw is DateTime ? raw : DateTime.parse(raw.toString());
    const m = ['', 'Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun',
               'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];
    return '${m[dt.month.clamp(1, 12)]} ${dt.day}, ${dt.year}';
  } catch (_) {
    return '—';
  }
}

// =============================================================================
// Bucket grid card (for grid view)
// =============================================================================

class _BucketGridCard extends StatefulWidget {
  final Map<String, dynamic> bucket;
  final VoidCallback onTap;

  const _BucketGridCard({required this.bucket, required this.onTap});

  @override
  State<_BucketGridCard> createState() => _BucketGridCardState();
}

class _BucketGridCardState extends State<_BucketGridCard> {
  bool _hovered = false;

  @override
  Widget build(BuildContext context) {
    final cs = consoleColors(context);
    final id = widget.bucket[r'$id'] as String? ?? '';
    final name = widget.bucket['name'] as String? ?? '';

    return MouseRegion(
      cursor: SystemMouseCursors.click,
      onEnter: (_) => setState(() => _hovered = true),
      onExit: (_) => setState(() => _hovered = false),
      child: GestureDetector(
        onTap: widget.onTap,
        child: AnimatedContainer(
          duration: const Duration(milliseconds: 120),
          padding: const EdgeInsets.all(16),
          decoration: BoxDecoration(
            color: _hovered ? cs.fillHover : cs.surface,
            borderRadius: BorderRadius.circular(10),
            border: Border.all(
                color: _hovered ? _accent.withValues(alpha: 0.35) : cs.border),
          ),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              const Icon(LucideIcons.folderClosed, size: 20, color: _accent),
              const SizedBox(height: 10),
              Text(name,
                  style: TextStyle(
                      color: cs.textPrimary,
                      fontSize: 14,
                      fontWeight: FontWeight.w500),
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis),
              const Spacer(),
              Container(
                padding:
                    const EdgeInsets.symmetric(horizontal: 7, vertical: 3),
                decoration: BoxDecoration(
                  color: cs.fill,
                  borderRadius: BorderRadius.circular(5),
                  border: Border.all(color: cs.border),
                ),
                child: IdText(id: id, fontSize: 11),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

// =============================================================================
// Bucket Detail View (Files, Usage, Settings)
// =============================================================================

class _BucketDetailView extends ConsumerStatefulWidget {
  final String bucketId;
  final VoidCallback onBack;
  final ValueChanged<String> onFileSelect;

  const _BucketDetailView({
    required this.bucketId,
    required this.onBack,
    required this.onFileSelect,
  });

  @override
  ConsumerState<_BucketDetailView> createState() =>
      _BucketDetailViewState();
}

class _BucketDetailViewState extends ConsumerState<_BucketDetailView> {
  int _tabIndex = 0;
  final _fileSearchCtrl = TextEditingController();

  @override
  void dispose() {
    _fileSearchCtrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final cs = consoleColors(context);
    final bucketAsync = ref.watch(_bucketDetailProvider(widget.bucketId));
    final bucketName = bucketAsync.valueOrNull?['name'] as String? ?? widget.bucketId;

    return Scaffold(
      backgroundColor: cs.background,
      body: Padding(
        padding: EdgeInsets.symmetric(
          horizontal: pageHPad(context),
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const SizedBox(height: 32),
            // Back + title
            Row(
              children: [
                GestureDetector(
                  onTap: widget.onBack,
                  child: MouseRegion(
                    cursor: SystemMouseCursors.click,
                    child: Icon(LucideIcons.arrowLeft,
                        size: 20, color: cs.textMuted),
                  ),
                ),
                const SizedBox(width: 12),
                Text(bucketName,
                    style: TextStyle(
                        color: cs.textPrimary,
                        fontSize: 22,
                        fontWeight: FontWeight.w600)),
                const SizedBox(width: 12),
                Row(
                  children: [
                    Icon(LucideIcons.folderClosed,
                        size: 13,
                        color: cs.textSubtle),
                    const SizedBox(width: 4),
                    IdText(id: widget.bucketId),
                  ],
                ),
              ],
            ),
            const SizedBox(height: 24),
            PageTabs(
              tabs: const ['Files', 'Usage', 'Settings'],
              selected: _tabIndex,
              onChanged: (i) => setState(() => _tabIndex = i),
            ),
            const SizedBox(height: 20),
            Expanded(
              child: _tabIndex == 0
                  ? _buildFilesTab()
                  : _tabIndex == 1
                      ? _buildUsageTab()
                      : _buildSettingsTab(),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildFilesTab() {
    final cs = consoleColors(context);
    final filesAsync = ref.watch(_filesProvider(widget.bucketId));

    return Column(
      children: [
        // Search + upload
        Row(
          children: [
            SizedBox(
              width: 280,
              child: TextField(
                controller: _fileSearchCtrl,
                style: TextStyle(fontSize: 13, color: cs.textPrimary),
                decoration: InputDecoration(
                  hintText: 'Search files',
                  hintStyle:
                      TextStyle(color: cs.textSubtle, fontSize: 13),
                  prefixIcon: Padding(
                    padding: const EdgeInsets.only(left: 10, right: 6),
                    child:
                        Icon(Icons.search, size: 16, color: cs.textSubtle),
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
            const Spacer(),
            FilledButton.icon(
              style: FilledButton.styleFrom(
                backgroundColor: _accent,
                padding:
                    const EdgeInsets.symmetric(horizontal: 14, vertical: 8),
                shape: RoundedRectangleBorder(
                    borderRadius: BorderRadius.circular(8)),
              ),
              icon: const Icon(LucideIcons.plus, size: 14),
              label:
                  const Text('Create file', style: TextStyle(fontSize: 12)),
              onPressed: () {},
            ),
          ],
        ),
        const SizedBox(height: 16),
        // Files table
        Expanded(
          child: filesAsync.when(
            loading: () =>
                const Center(child: CircularProgressIndicator()),
            error: (e, _) => AppErrorState(error: e),
            data: (data) {
              final files = List<Map<String, dynamic>>.from(
                  data['files'] ?? []);
              if (files.isEmpty) {
                return _EmptyState(
                  icon: LucideIcons.file,
                  title: 'No files',
                  subtitle: 'Upload a file to this bucket',
                  actionLabel: 'Upload file',
                  onAction: () {},
                );
              }
              return _FilesTable(
                files: files,
                bucketId: widget.bucketId,
                onSelect: widget.onFileSelect,
                onDelete: (fileId) => _deleteFile(fileId),
              );
            },
          ),
        ),
      ],
    );
  }

  Widget _buildUsageTab() {
    return const Padding(
      padding: EdgeInsets.only(top: 8),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              _UsageStatCard(label: 'Total Files', value: '—'),
              SizedBox(width: 16),
              _UsageStatCard(label: 'Storage Used', value: '—'),
              SizedBox(width: 16),
              _UsageStatCard(label: 'Bandwidth', value: '—'),
            ],
          ),
        ],
      ),
    );
  }

  Widget _buildSettingsTab() {
    final cs = consoleColors(context);
    final bucketAsync = ref.watch(_bucketDetailProvider(widget.bucketId));

    return bucketAsync.when(
      loading: () => const Center(child: CircularProgressIndicator()),
      error: (e, _) => AppErrorState(error: e),
      data: (bucket) {
        final name = bucket['name'] as String? ?? '';
        final enabled = bucket['enabled'] as bool? ?? true;
        final fileSecurity = bucket['fileSecurity'] as bool? ?? false;
        final encryption = bucket['encryption'] as bool? ?? true;
        final antivirus = bucket['antivirus'] as bool? ?? false;
        final compression = bucket['compression'] as String? ?? 'none';
        final maxSize = bucket['maximumFileSize'] as int? ?? 0;
        final extensions = List<String>.from(
            bucket['allowedFileExtensions'] ?? []);
        final created = bucket['createdAt'] ?? bucket['\$createdAt'] ?? '';
        final updated = bucket['updatedAt'] ?? bucket['\$updatedAt'] ?? '';

        return SingleChildScrollView(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              // 1. Bucket status
              _SettingsSection(
                children: [
                  Row(
                    children: [
                      Expanded(
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Text(name,
                                style: TextStyle(
                                    color: cs.textPrimary,
                                    fontSize: 15,
                                    fontWeight: FontWeight.w600)),
                            const SizedBox(height: 4),
                            Text('Created: $created',
                                style: TextStyle(
                                    color: cs.textSubtle, fontSize: 12)),
                            Text('Last updated: $updated',
                                style: TextStyle(
                                    color: cs.textSubtle, fontSize: 12)),
                          ],
                        ),
                      ),
                      _SettingsToggle(
                        label: 'Enabled',
                        value: enabled,
                        onChanged: (_) {},
                      ),
                    ],
                  ),
                ],
                onUpdate: () => _updateBucket({'enabled': !enabled}),
              ),

              // 2. Name
              _SettingsSection(
                title: 'Name',
                children: [
                  _SettingsTextField(
                    label: 'Name',
                    initialValue: name,
                    onSaved: (v) => _updateBucket({'name': v}),
                  ),
                ],
              ),

              // 3. Permissions
              _SettingsSection(
                title: 'Permissions',
                subtitle:
                    'Choose who can access your buckets and files.',
                children: const [
                  _PermissionsTable(),
                ],
                onUpdate: () {},
              ),

              // 4. File security
              _SettingsSection(
                title: 'File security',
                children: [
                  _SettingsToggle(
                    label: 'File security',
                    value: fileSecurity,
                    onChanged: (_) {},
                  ),
                  const SizedBox(height: 8),
                  Text(
                    fileSecurity
                        ? 'When file security is enabled, users will be able to access files for which they have been granted either file or bucket permissions.'
                        : 'If file security is disabled, users can access files only if they have bucket permissions. File permissions will be ignored.',
                    style: TextStyle(
                        color: cs.textSubtle,
                        fontSize: 12),
                  ),
                ],
                onUpdate: () =>
                    _updateBucket({'fileSecurity': !fileSecurity}),
              ),

              // 5. Security settings
              _SettingsSection(
                title: 'Security settings',
                subtitle:
                    'Enable or disable security features for this bucket.',
                children: [
                  _SettingsToggle(
                    label: 'Encryption',
                    value: encryption,
                    onChanged: (_) {},
                    subtitle:
                        'Files inside this bucket will be encrypted. Files larger than 20MB will not be encrypted.',
                  ),
                  const SizedBox(height: 12),
                  _SettingsToggle(
                    label: 'Antivirus',
                    value: antivirus,
                    onChanged: (_) {},
                    subtitle:
                        'Files inside this bucket will be scanned by the antivirus scanner.',
                  ),
                ],
                onUpdate: () => _updateBucket({
                  'encryption': !encryption,
                }),
              ),

              // 6. Compression
              _SettingsSection(
                title: 'Compression',
                subtitle:
                    'Choose an algorithm for compression. For files larger than 20MB, compression will be skipped.',
                children: [
                  _SettingsDropdown(
                    label: 'Algorithm',
                    value: compression,
                    options: const {
                      'none': 'None',
                      'gzip': 'gzip',
                      'zstd': 'zstd',
                    },
                    onChanged: (_) {},
                  ),
                ],
                onUpdate: () {},
              ),

              // 7. Maximum file size
              _SettingsSection(
                title: 'Maximum file size',
                subtitle:
                    'Set the maximum file size allowed in this bucket.',
                children: [
                  Row(
                    children: [
                      SizedBox(
                        width: 120,
                        child: Text(
                            maxSize > 0
                                ? (maxSize / (1024 * 1024)).toStringAsFixed(0)
                                : 'Unlimited',
                            style: TextStyle(
                                color: cs.textPrimary, fontSize: 14)),
                      ),
                      const SizedBox(width: 8),
                      Text('MB',
                          style: TextStyle(
                              color: cs.textMuted, fontSize: 13)),
                    ],
                  ),
                ],
                onUpdate: () {},
              ),

              // 8. Allowed file extensions
              _SettingsSection(
                title: 'Allowed file extensions',
                subtitle:
                    'Allowed file extensions. A maximum of 100 file extensions can be added. Leave empty to allow all file types.',
                children: [
                  if (extensions.isEmpty)
                    Text('All file types allowed',
                        style: TextStyle(
                            color: cs.textSecondary,
                            fontSize: 13))
                  else
                    Wrap(
                      spacing: 6,
                      runSpacing: 6,
                      children: extensions
                          .map((ext) => Container(
                                padding: const EdgeInsets.symmetric(
                                    horizontal: 8, vertical: 4),
                                decoration: BoxDecoration(
                                  color: _accent.withValues(alpha: 0.15),
                                  borderRadius:
                                      BorderRadius.circular(4),
                                ),
                                child: Text(ext,
                                    style: TextStyle(
                                        color: cs.textPrimary,
                                        fontSize: 12,
                                        fontFamily: 'monospace')),
                              ))
                          .toList(),
                    ),
                ],
                onUpdate: () {},
              ),

              // 9. Delete bucket
              Container(
                width: double.infinity,
                padding: const EdgeInsets.all(20),
                margin: const EdgeInsets.only(bottom: 40),
                decoration: BoxDecoration(
                  color: cs.surface,
                  borderRadius: BorderRadius.circular(8),
                  border: Border.all(color: _red.withValues(alpha: 0.3)),
                ),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Row(
                      children: [
                        Expanded(
                          child: Column(
                            crossAxisAlignment:
                                CrossAxisAlignment.start,
                            children: [
                              const Text('Delete bucket',
                                  style: TextStyle(
                                      color: _red,
                                      fontSize: 14,
                                      fontWeight: FontWeight.w500)),
                              const SizedBox(height: 4),
                              Text(
                                  'The bucket will be permanently deleted, including all the files within it. This action is irreversible.',
                                  style: TextStyle(
                                      color: cs.textSecondary,
                                      fontSize: 13)),
                            ],
                          ),
                        ),
                        const SizedBox(width: 16),
                        Container(
                          padding: const EdgeInsets.all(12),
                          decoration: BoxDecoration(
                            color: cs.fill,
                            borderRadius: BorderRadius.circular(8),
                            border: Border.all(color: cs.border),
                          ),
                          child: Column(
                            crossAxisAlignment:
                                CrossAxisAlignment.start,
                            children: [
                              Text(name,
                                  style: TextStyle(
                                      color: cs.textPrimary,
                                      fontSize: 13,
                                      fontWeight: FontWeight.w500)),
                              Text('Last updated: $updated',
                                  style: TextStyle(
                                      color: cs.textSubtle,
                                      fontSize: 11)),
                            ],
                          ),
                        ),
                      ],
                    ),
                    const SizedBox(height: 12),
                    Align(
                      alignment: Alignment.centerRight,
                      child: OutlinedButton(
                        style: OutlinedButton.styleFrom(
                          foregroundColor: _red,
                          side: const BorderSide(color: _red),
                          padding: const EdgeInsets.symmetric(
                              horizontal: 20, vertical: 10),
                          shape: RoundedRectangleBorder(
                              borderRadius: BorderRadius.circular(8)),
                        ),
                        onPressed: () async {
                          final cs2 = consoleColors(context);
                          final confirmed = await showAppDialog<bool>(
                            context: context,
                            title: 'Delete bucket',
                            content: Text(
                              'All files in this bucket will be permanently deleted.',
                              style: TextStyle(color: cs2.textSecondary),
                            ),
                            actions: [
                              const AppDialogCancel(),
                              AppDialogAction(
                                label: 'Delete',
                                destructive: true,
                                onTap: () => Navigator.of(
                                  context,
                                  rootNavigator: true,
                                ).pop(true),
                              ),
                            ],
                          );
                          if (confirmed != true) return;
                          final api = ref.read(apiClientProvider);
                          await api.delete(
                              '/storage/buckets/${widget.bucketId}');
                          ref.invalidate(bucketsProvider);
                          widget.onBack();
                        },
                        child: const Text('Delete',
                            style: TextStyle(fontSize: 13)),
                      ),
                    ),
                  ],
                ),
              ),
            ],
          ),
        );
      },
    );
  }

  Future<void> _updateBucket(Map<String, dynamic> data) async {
    try {
      final api = ref.read(apiClientProvider);
      await api.put('/storage/buckets/${widget.bucketId}', data: data);
      ref.invalidate(_bucketDetailProvider(widget.bucketId));
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context)
            .showSnackBar(SnackBar(content: Text('Error: $e')));
      }
    }
  }

  Future<void> _deleteFile(String fileId) async {
    await ref
        .read(apiClientProvider)
        .delete('/storage/buckets/${widget.bucketId}/files/$fileId');
    ref.invalidate(_filesProvider(widget.bucketId));
  }

}

// =============================================================================
// Files Table
// =============================================================================

class _FilesTable extends StatelessWidget {
  final List<Map<String, dynamic>> files;
  final String bucketId;
  final ValueChanged<String> onSelect;
  final Future<void> Function(String) onDelete;

  const _FilesTable({
    required this.files,
    required this.bucketId,
    required this.onSelect,
    required this.onDelete,
  });

  @override
  Widget build(BuildContext context) {
    final cs = consoleColors(context);
    return Column(
      children: [
        Container(
          padding:
              const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
          decoration: BoxDecoration(
            border: Border(
                bottom:
                    BorderSide(color: cs.border)),
          ),
          child: Row(
            children: [
              const SizedBox(width: 32), // checkbox placeholder
              Expanded(flex: 4, child: Text('Filename', style: TextStyle(color: cs.textMuted, fontSize: 12, fontWeight: FontWeight.w500))),
              Expanded(flex: 2, child: Text('Type', style: TextStyle(color: cs.textMuted, fontSize: 12, fontWeight: FontWeight.w500))),
              Expanded(flex: 1, child: Text('Size', style: TextStyle(color: cs.textMuted, fontSize: 12, fontWeight: FontWeight.w500))),
              Expanded(flex: 2, child: Text('Created', style: TextStyle(color: cs.textMuted, fontSize: 12, fontWeight: FontWeight.w500))),
              const SizedBox(width: 40),
            ],
          ),
        ),
        Expanded(
          child: ListView.builder(
            itemCount: files.length,
            itemBuilder: (context, i) {
              final f = files[i];
              return _FileRow(
                file: f,
                onTap: () => onSelect(f['\$id'] as String),
                onDelete: () => onDelete(f['\$id'] as String),
              );
            },
          ),
        ),
      ],
    );
  }
}

class _FileRow extends StatefulWidget {
  final Map<String, dynamic> file;
  final VoidCallback onTap;
  final VoidCallback onDelete;

  const _FileRow({
    required this.file,
    required this.onTap,
    required this.onDelete,
  });

  @override
  State<_FileRow> createState() => _FileRowState();
}

class _FileRowState extends State<_FileRow> {
  bool _hovered = false;

  @override
  Widget build(BuildContext context) {
    final cs = consoleColors(context);
    final f = widget.file;
    final name = f['name'] as String? ?? 'Untitled';
    final mime = f['mimeType'] as String? ?? '';
    final size = _formatSize(f['sizeOriginal'] ?? 0);
    final created = _timeAgo(f['createdAt'] ?? f['\$createdAt']);

    return MouseRegion(
      onEnter: (_) => setState(() => _hovered = true),
      onExit: (_) => setState(() => _hovered = false),
      cursor: SystemMouseCursors.click,
      child: GestureDetector(
        onTap: widget.onTap,
        child: Container(
          padding:
              const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
          decoration: BoxDecoration(
            color: _hovered ? cs.fillHover : null,
            border: Border(
                bottom:
                    BorderSide(color: cs.fill)),
          ),
          child: Row(
            children: [
              const SizedBox(width: 32),
              Expanded(
                flex: 4,
                child: Row(
                  children: [
                    _mimeIcon(mime),
                    const SizedBox(width: 10),
                    Expanded(
                      child: Text(name,
                          style: TextStyle(
                              color: cs.textPrimary, fontSize: 13),
                          overflow: TextOverflow.ellipsis),
                    ),
                  ],
                ),
              ),
              Expanded(
                flex: 2,
                child: Text(mime,
                    style:
                        TextStyle(color: cs.textMuted, fontSize: 12)),
              ),
              Expanded(
                flex: 1,
                child: Text(size,
                    style:
                        TextStyle(color: cs.textMuted, fontSize: 12)),
              ),
              Expanded(
                flex: 2,
                child: Text(created,
                    style:
                        TextStyle(color: cs.textMuted, fontSize: 12)),
              ),
              SizedBox(
                width: 40,
                child: PopupMenuButton<String>(
                  color: cs.popupSurface,
                  shape: RoundedRectangleBorder(
                      borderRadius: BorderRadius.circular(8)),
                  iconSize: 16,
                  icon: Icon(LucideIcons.moreHorizontal,
                      size: 16,
                      color: _hovered ? cs.textMuted : Colors.transparent),
                  onSelected: (v) async {
                    if (v == 'delete') {
                      final cs2 = consoleColors(context);
                      final confirmed = await showAppDialog<bool>(
                        context: context,
                        title: 'Delete file',
                        content: Text(
                          'Are you sure? This action cannot be undone.',
                          style: TextStyle(color: cs2.textSecondary),
                        ),
                        actions: [
                          const AppDialogCancel(),
                          AppDialogAction(
                            label: 'Delete',
                            destructive: true,
                            onTap: () => Navigator.of(
                              context,
                              rootNavigator: true,
                            ).pop(true),
                          ),
                        ],
                      );
                      if (confirmed == true) widget.onDelete();
                    }
                  },
                  itemBuilder: (_) => [
                    const PopupMenuItem(
                      value: 'delete',
                      child: Text('Delete',
                          style: TextStyle(
                              color: _red, fontSize: 13)),
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

  Widget _mimeIcon(String mime) {
    IconData icon;
    Color color;
    if (mime.startsWith('image/')) {
      icon = LucideIcons.image;
      color = const Color(0xFF10B981);
    } else if (mime.startsWith('video/')) {
      icon = LucideIcons.video;
      color = const Color(0xFF7C3AED);
    } else if (mime.contains('pdf')) {
      icon = LucideIcons.fileText;
      color = _red;
    } else {
      icon = LucideIcons.file;
      color = _accent;
    }
    return Container(
      width: 28,
      height: 28,
      decoration: BoxDecoration(
        color: color.withValues(alpha: 0.1),
        borderRadius: BorderRadius.circular(6),
      ),
      child: Icon(icon, size: 14, color: color),
    );
  }

  String _formatSize(dynamic bytes) {
    final b = (bytes is int) ? bytes : 0;
    if (b < 1024) return '$b B';
    if (b < 1024 * 1024) return '${(b / 1024).toStringAsFixed(0)} KB';
    return '${(b / (1024 * 1024)).toStringAsFixed(1)} MB';
  }

  String _timeAgo(dynamic raw) {
    if (raw == null) return '—';
    try {
      final dt = raw is DateTime ? raw : DateTime.parse(raw.toString());
      final diff = DateTime.now().difference(dt);
      if (diff.inDays == 0) return 'Today';
      if (diff.inDays == 1) return 'Yesterday';
      if (diff.inDays < 30) return '${diff.inDays} days ago';
      return '${_monthName(dt.month)} ${dt.day}, ${dt.year}';
    } catch (_) {
      return '—';
    }
  }

  String _monthName(int m) {
    const names = [
      '', 'Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun',
      'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'
    ];
    return names[m.clamp(1, 12)];
  }
}

// =============================================================================
// File Detail View
// =============================================================================

class _FileDetailView extends ConsumerWidget {
  final String bucketId;
  final String fileId;
  final VoidCallback onBack;

  const _FileDetailView({
    required this.bucketId,
    required this.fileId,
    required this.onBack,
  });

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final cs = consoleColors(context);
    final filesAsync = ref.watch(_filesProvider(bucketId));

    // Find file from cache
    final file = filesAsync.valueOrNull?['files'] != null
        ? (filesAsync.value!['files'] as List)
            .cast<Map<String, dynamic>>()
            .firstWhere((f) => f['\$id'] == fileId,
                orElse: () => <String, dynamic>{})
        : <String, dynamic>{};

    final name = file['name'] as String? ?? fileId;
    final mime = file['mimeType'] as String? ?? '';
    final size = file['sizeOriginal'] as int? ?? 0;
    final created = file['createdAt'] ?? file['\$createdAt'];
    final updated = file['updatedAt'] ?? file['\$updatedAt'];
    final api = ref.read(apiClientProvider);
    final fileUrl =
        '${api.dio.options.baseUrl}/storage/buckets/$bucketId/files/$fileId/view';

    return Scaffold(
      backgroundColor: cs.background,
      body: SingleChildScrollView(
        padding: EdgeInsets.symmetric(
          horizontal: pageHPad(context),
          vertical: 32,
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            // Back + title
            Row(
              children: [
                GestureDetector(
                  onTap: onBack,
                  child: MouseRegion(
                    cursor: SystemMouseCursors.click,
                    child: Icon(LucideIcons.arrowLeft,
                        size: 20,
                        color: cs.textMuted),
                  ),
                ),
                const SizedBox(width: 12),
                Expanded(
                  child: Text(name,
                      style: TextStyle(
                          color: cs.textPrimary,
                          fontSize: 22,
                          fontWeight: FontWeight.w600),
                      overflow: TextOverflow.ellipsis),
                ),
                const SizedBox(width: 12),
                Row(
                  children: [
                    Icon(LucideIcons.file,
                        size: 13,
                        color: cs.textSubtle),
                    const SizedBox(width: 4),
                    IdText(id: fileId),
                  ],
                ),
              ],
            ),
            const SizedBox(height: 24),

            // File info card with preview
            Container(
              width: double.infinity,
              padding: const EdgeInsets.all(24),
              decoration: BoxDecoration(
                color: cs.surface,
                borderRadius: BorderRadius.circular(8),
                border: Border.all(color: cs.border),
              ),
              child: Row(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  // Preview
                  if (mime.startsWith('image/'))
                    Container(
                      width: 200,
                      height: 160,
                      decoration: BoxDecoration(
                        color: cs.fieldFill,
                        borderRadius: BorderRadius.circular(8),
                        border: Border.all(color: cs.border),
                      ),
                      clipBehavior: Clip.antiAlias,
                      child: Image.network(fileUrl,
                          fit: BoxFit.contain,
                          errorBuilder: (_, __, ___) => Center(
                              child: Icon(LucideIcons.image,
                                  size: 32, color: cs.textSubtle))),
                    )
                  else
                    Container(
                      width: 200,
                      height: 160,
                      decoration: BoxDecoration(
                        color: cs.fieldFill,
                        borderRadius: BorderRadius.circular(8),
                        border: Border.all(color: cs.border),
                      ),
                      child: Center(
                          child: Icon(LucideIcons.file,
                              size: 48, color: cs.textSubtle)),
                    ),
                  const SizedBox(width: 24),
                  // Metadata
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        _MetaRow(label: 'Filename', value: name),
                        _MetaRow(label: 'MIME type', value: mime),
                        _MetaRow(label: 'Size', value: _fmtBytes(size)),
                        if (created != null)
                          _MetaRow(label: 'Created', value: '$created'),
                        if (updated != null)
                          _MetaRow(
                              label: 'Last updated', value: '$updated'),
                        const SizedBox(height: 16),
                        // File URL
                        Text('File URL',
                            style: TextStyle(
                                color: cs.textMuted, fontSize: 12)),
                        const SizedBox(height: 4),
                        Container(
                          padding: const EdgeInsets.symmetric(
                              horizontal: 10, vertical: 8),
                          decoration: BoxDecoration(
                            color: cs.fieldFill,
                            borderRadius: BorderRadius.circular(6),
                            border: Border.all(color: cs.fieldBorder),
                          ),
                          child: Row(
                            children: [
                              Expanded(
                                child: Text(fileUrl,
                                    style: TextStyle(
                                        color: cs.textMuted,
                                        fontSize: 12,
                                        fontFamily: 'monospace'),
                                    overflow: TextOverflow.ellipsis),
                              ),
                              GestureDetector(
                                onTap: () {
                                  Clipboard.setData(
                                      ClipboardData(text: fileUrl));
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
                                      color: cs.textSubtle),
                                ),
                              ),
                            ],
                          ),
                        ),
                        const SizedBox(height: 12),
                        // Download button
                        OutlinedButton.icon(
                          style: OutlinedButton.styleFrom(
                            foregroundColor: cs.textSecondary,
                            side: BorderSide(color: cs.border),
                            padding: const EdgeInsets.symmetric(
                                horizontal: 16, vertical: 8),
                            shape: RoundedRectangleBorder(
                                borderRadius: BorderRadius.circular(8)),
                          ),
                          icon: const Icon(LucideIcons.download, size: 14),
                          label: const Text('Download',
                              style: TextStyle(fontSize: 13)),
                          onPressed: () {},
                        ),
                      ],
                    ),
                  ),
                ],
              ),
            ),
            const SizedBox(height: 20),

            // Permissions card
            Container(
              width: double.infinity,
              padding: const EdgeInsets.all(20),
              decoration: BoxDecoration(
                color: cs.surface,
                borderRadius: BorderRadius.circular(8),
                border: Border.all(color: cs.border),
              ),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text('Permissions',
                      style: TextStyle(
                          color: cs.textPrimary,
                          fontSize: 15,
                          fontWeight: FontWeight.w600)),
                  const SizedBox(height: 8),
                  Text(
                    'Assign read or write permissions at the bucket level or file level.',
                    style: TextStyle(
                        color: cs.textSubtle,
                        fontSize: 13),
                  ),
                ],
              ),
            ),
            const SizedBox(height: 20),

            // Delete card
            Container(
              width: double.infinity,
              padding: const EdgeInsets.all(20),
              decoration: BoxDecoration(
                color: cs.surface,
                borderRadius: BorderRadius.circular(8),
                border: Border.all(color: _red.withValues(alpha: 0.3)),
              ),
              child: Row(
                children: [
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        const Text('Delete file',
                            style: TextStyle(
                                color: _red,
                                fontSize: 14,
                                fontWeight: FontWeight.w500)),
                        const SizedBox(height: 4),
                        Text(
                            'The file will be permanently deleted. This action is irreversible.',
                            style: TextStyle(
                                color: cs.textSubtle,
                                fontSize: 13)),
                      ],
                    ),
                  ),
                  OutlinedButton(
                    style: OutlinedButton.styleFrom(
                      foregroundColor: _red,
                      side: const BorderSide(color: _red),
                      padding: const EdgeInsets.symmetric(
                          horizontal: 20, vertical: 10),
                      shape: RoundedRectangleBorder(
                          borderRadius: BorderRadius.circular(8)),
                    ),
                    onPressed: () async {
                      final cs2 = consoleColors(context);
                      final confirmed = await showAppDialog<bool>(
                        context: context,
                        title: 'Delete file',
                        content: Text(
                          'The file will be permanently deleted. This action is irreversible.',
                          style: TextStyle(color: cs2.textSecondary),
                        ),
                        actions: [
                          const AppDialogCancel(),
                          AppDialogAction(
                            label: 'Delete',
                            destructive: true,
                            onTap: () => Navigator.of(
                              context,
                              rootNavigator: true,
                            ).pop(true),
                          ),
                        ],
                      );
                      if (confirmed != true) return;
                      await ref
                          .read(apiClientProvider)
                          .delete(
                              '/storage/buckets/$bucketId/files/$fileId');
                      ref.invalidate(_filesProvider(bucketId));
                      onBack();
                    },
                    child: const Text('Delete',
                        style: TextStyle(fontSize: 13)),
                  ),
                ],
              ),
            ),
            const SizedBox(height: 40),
          ],
        ),
      ),
    );
  }

  String _fmtBytes(int bytes) {
    if (bytes <= 0) return '0 B';
    const units = ['B', 'KB', 'MB', 'GB'];
    final i = (math.log(bytes) / math.log(1024)).floor().clamp(0, 3);
    return '${(bytes / math.pow(1024, i)).toStringAsFixed(1)} ${units[i]}';
  }
}

class _MetaRow extends StatelessWidget {
  final String label;
  final String value;
  const _MetaRow({required this.label, required this.value});

  @override
  Widget build(BuildContext context) {
    final cs = consoleColors(context);
    return Padding(
      padding: const EdgeInsets.only(bottom: 6),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          SizedBox(
            width: 110,
            child: Text(label,
                style: TextStyle(color: cs.textMuted, fontSize: 13)),
          ),
          Expanded(
            child: Text(value,
                style: TextStyle(
                    color: cs.textPrimary, fontSize: 13)),
          ),
        ],
      ),
    );
  }
}

// =============================================================================
// Shared
// =============================================================================

class _UsageStatCard extends StatelessWidget {
  final String label;
  final String value;
  const _UsageStatCard({required this.label, required this.value});

  @override
  Widget build(BuildContext context) {
    final cs = consoleColors(context);
    return Expanded(
      child: Container(
        padding: const EdgeInsets.all(20),
        decoration: BoxDecoration(
          color: cs.surface,
          borderRadius: BorderRadius.circular(8),
          border: Border.all(color: cs.border),
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(value,
                style: TextStyle(
                    color: cs.textPrimary,
                    fontSize: 24,
                    fontWeight: FontWeight.w700)),
            const SizedBox(height: 4),
            Text(label,
                style: TextStyle(
                    color: cs.textMuted,
                    fontSize: 12)),
          ],
        ),
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
    final cs = consoleColors(context);
    return Center(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Container(
            width: 48,
            height: 48,
            decoration: BoxDecoration(
              color: cs.fill,
              borderRadius: BorderRadius.circular(12),
            ),
            child: Icon(icon, size: 22, color: cs.textSubtle),
          ),
          const SizedBox(height: 16),
          Text(title,
              style: TextStyle(
                  color: cs.textPrimary,
                  fontSize: 15,
                  fontWeight: FontWeight.w500)),
          const SizedBox(height: 6),
          Text(subtitle,
              style: TextStyle(color: cs.textMuted, fontSize: 13)),
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
            child:
                Text(actionLabel, style: const TextStyle(fontSize: 13)),
          ),
        ],
      ),
    );
  }
}

// =============================================================================
// Settings Widgets
// =============================================================================

class _SettingsSection extends StatelessWidget {
  final String? title;
  final String? subtitle;
  final List<Widget> children;
  final VoidCallback? onUpdate;

  const _SettingsSection({
    this.title,
    this.subtitle,
    required this.children,
    this.onUpdate,
  });

  @override
  Widget build(BuildContext context) {
    final cs = consoleColors(context);
    return Container(
      width: double.infinity,
      margin: const EdgeInsets.only(bottom: 16),
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: cs.surface,
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: cs.border),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          if (title != null) ...[
            Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(title!,
                          style: TextStyle(
                              color: cs.textPrimary,
                              fontSize: 15,
                              fontWeight: FontWeight.w600)),
                      if (subtitle != null) ...[
                        const SizedBox(height: 4),
                        Text(subtitle!,
                            style: TextStyle(
                                color: cs.textSubtle,
                                fontSize: 13)),
                      ],
                    ],
                  ),
                ),
              ],
            ),
            const SizedBox(height: 16),
          ],
          ...children,
          if (onUpdate != null) ...[
            const SizedBox(height: 16),
            Align(
              alignment: Alignment.centerRight,
              child: FilledButton(
                style: FilledButton.styleFrom(
                  backgroundColor: _accent,
                  padding: const EdgeInsets.symmetric(
                      horizontal: 20, vertical: 10),
                  shape: RoundedRectangleBorder(
                      borderRadius: BorderRadius.circular(8)),
                ),
                onPressed: onUpdate,
                child: const Text('Update',
                    style: TextStyle(fontSize: 13)),
              ),
            ),
          ],
        ],
      ),
    );
  }
}

class _SettingsToggle extends StatelessWidget {
  final String label;
  final bool value;
  final ValueChanged<bool> onChanged;
  final String? subtitle;

  const _SettingsToggle({
    required this.label,
    required this.value,
    required this.onChanged,
    this.subtitle,
  });

  @override
  Widget build(BuildContext context) {
    final cs = consoleColors(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          children: [
            Switch(
              value: value,
              onChanged: onChanged,
              activeThumbColor: _accent,
            ),
            const SizedBox(width: 8),
            Expanded(
              child: Text(label,
                  style: TextStyle(
                      color: cs.textPrimary,
                      fontSize: 13,
                      fontWeight: FontWeight.w500)),
            ),
          ],
        ),
        if (subtitle != null) ...[
          Padding(
            padding: const EdgeInsets.only(left: 52),
            child: Text(subtitle!,
                style: TextStyle(
                    color: cs.textSubtle,
                    fontSize: 12)),
          ),
        ],
      ],
    );
  }
}

class _SettingsTextField extends StatefulWidget {
  final String label;
  final String initialValue;
  final ValueChanged<String> onSaved;

  const _SettingsTextField({
    required this.label,
    required this.initialValue,
    required this.onSaved,
  });

  @override
  State<_SettingsTextField> createState() => _SettingsTextFieldState();
}

class _SettingsTextFieldState extends State<_SettingsTextField> {
  late final TextEditingController _ctrl;

  @override
  void initState() {
    super.initState();
    _ctrl = TextEditingController(text: widget.initialValue);
  }

  @override
  void dispose() {
    _ctrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(widget.label,
            style: TextStyle(
                color: Colors.white.withValues(alpha: 0.5),
                fontSize: 12,
                fontWeight: FontWeight.w500)),
        const SizedBox(height: 6),
        Row(
          children: [
            Expanded(
              child: TextField(
                controller: _ctrl,
                style:
                    const TextStyle(color: Colors.white, fontSize: 13),
                decoration: InputDecoration(
                  filled: true,
                  fillColor: const Color(0x0AFFFFFF),
                  isDense: true,
                  contentPadding: const EdgeInsets.symmetric(
                      horizontal: 12, vertical: 10),
                  border: OutlineInputBorder(
                      borderRadius: BorderRadius.circular(8),
                      borderSide:
                          const BorderSide(color: Color(0x1AFFFFFF))),
                  enabledBorder: OutlineInputBorder(
                      borderRadius: BorderRadius.circular(8),
                      borderSide:
                          const BorderSide(color: Color(0x1AFFFFFF))),
                  focusedBorder: OutlineInputBorder(
                      borderRadius: BorderRadius.circular(8),
                      borderSide: const BorderSide(color: _accent)),
                ),
              ),
            ),
            const SizedBox(width: 12),
            FilledButton(
              style: FilledButton.styleFrom(
                backgroundColor: _accent,
                padding: const EdgeInsets.symmetric(
                    horizontal: 20, vertical: 10),
                shape: RoundedRectangleBorder(
                    borderRadius: BorderRadius.circular(8)),
              ),
              onPressed: () => widget.onSaved(_ctrl.text.trim()),
              child:
                  const Text('Update', style: TextStyle(fontSize: 13)),
            ),
          ],
        ),
      ],
    );
  }
}

class _SettingsDropdown extends StatelessWidget {
  final String label;
  final String value;
  final Map<String, String> options;
  final ValueChanged<String?> onChanged;

  const _SettingsDropdown({
    required this.label,
    required this.value,
    required this.options,
    required this.onChanged,
  });

  @override
  Widget build(BuildContext context) {
    final cs = consoleColors(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(label,
            style: TextStyle(
                color: cs.textMuted,
                fontSize: 12,
                fontWeight: FontWeight.w500)),
        const SizedBox(height: 6),
        Container(
          padding: const EdgeInsets.symmetric(horizontal: 12),
          decoration: BoxDecoration(
            color: cs.fieldFill,
            borderRadius: BorderRadius.circular(8),
            border: Border.all(color: cs.fieldBorder),
          ),
          child: DropdownButtonHideUnderline(
            child: DropdownButton<String>(
              value: options.containsKey(value) ? value : options.keys.first,
              isExpanded: true,
              isDense: true,
              dropdownColor: cs.popupSurface,
              style: TextStyle(
                  color: cs.textPrimary, fontSize: 13),
              icon: Icon(LucideIcons.chevronDown,
                  size: 14, color: cs.textMuted),
              items: options.entries
                  .map((e) => DropdownMenuItem(
                      value: e.key, child: Text(e.value)))
                  .toList(),
              onChanged: onChanged,
            ),
          ),
        ),
      ],
    );
  }
}

class _PermissionsTable extends StatelessWidget {
  const _PermissionsTable();

  @override
  Widget build(BuildContext context) {
    final cs = consoleColors(context);
    const roles = ['Users', 'Guests', 'Any'];
    const perms = ['Create', 'Read', 'Update', 'Delete'];

    return Column(
      children: [
        // Header
        Row(
          children: [
            const SizedBox(width: 80),
            ...perms.map((p) => Expanded(
                  child: Center(
                    child: Text(p,
                        style: TextStyle(
                            color: cs.textMuted,
                            fontSize: 11,
                            fontWeight: FontWeight.w500)),
                  ),
                )),
            const SizedBox(width: 32),
          ],
        ),
        const SizedBox(height: 8),
        // Rows
        ...roles.map((role) => Padding(
              padding: const EdgeInsets.symmetric(vertical: 4),
              child: Row(
                children: [
                  SizedBox(
                    width: 80,
                    child: Text(role,
                        style: const TextStyle(
                            color: Colors.white, fontSize: 13)),
                  ),
                  ...perms.map((_) => Expanded(
                        child: Center(
                          child: SizedBox(
                            width: 18,
                            height: 18,
                            child: Checkbox(
                              value: false,
                              onChanged: (_) {},
                              activeColor: _accent,
                              side: BorderSide(
                                  color: Colors.white.withValues(alpha: 0.2)),
                              shape: RoundedRectangleBorder(
                                  borderRadius:
                                      BorderRadius.circular(3)),
                            ),
                          ),
                        ),
                      )),
                  const SizedBox(width: 32),
                ],
              ),
            )),
        const SizedBox(height: 8),
        Align(
          alignment: Alignment.centerLeft,
          child: TextButton.icon(
            onPressed: () {},
            icon: const Icon(LucideIcons.plus, size: 14),
            label:
                const Text('Add role', style: TextStyle(fontSize: 12)),
            style: TextButton.styleFrom(foregroundColor: _accent),
          ),
        ),
      ],
    );
  }
}
