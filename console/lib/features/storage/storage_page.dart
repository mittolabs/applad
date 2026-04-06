import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../core/api/client.dart';
import '../../core/widgets/app_dialog.dart';
import '../../core/widgets/page_tabs.dart';
import '../../core/widgets/search_list.dart';

// --- Providers -----------------------------------------------------------

final _storageTabProvider = StateProvider<int>((ref) => 0);
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

final selectedBucketProvider = StateProvider<String?>((ref) => null);

final filesProvider =
    FutureProvider.family<Map<String, dynamic>, String>((ref, bucketId) async {
  final api = ref.read(apiClientProvider);
  final res =
      await api.get('/storage/buckets/$bucketId/files', params: {'limit': 50});
  return res.data as Map<String, dynamic>;
});

// --- Page ----------------------------------------------------------------

class StoragePage extends ConsumerStatefulWidget {
  const StoragePage({super.key});

  @override
  ConsumerState<StoragePage> createState() => _StoragePageState();
}

class _StoragePageState extends ConsumerState<StoragePage> {
  final _searchCtrl = TextEditingController();

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
    final tab = ref.watch(_storageTabProvider);

    return Scaffold(
      body: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(24, 20, 24, 0),
            child: Text('Storage',
                style: Theme.of(context)
                    .textTheme
                    .headlineSmall
                    ?.copyWith(color: Colors.white)),
          ),
          PageTabs(
            tabs: const ['Buckets', 'Usage'],
            selected: tab,
            onChanged: (i) =>
                ref.read(_storageTabProvider.notifier).state = i,
          ),
          const Divider(height: 1, color: Color(0xFF2A2B30)),
          Expanded(child: tab == 0 ? _bucketsTab() : const _UsageTab()),
        ],
      ),
    );
  }

  Widget _bucketsTab() {
    final bucketsAsync = ref.watch(bucketsProvider);
    final selectedBucket = ref.watch(selectedBucketProvider);
    final perPage = ref.watch(_bucketPerPageProvider);
    final currentPage = ref.watch(_bucketPageProvider);

    return Column(
      children: [
        SearchListHeader(
          searchController: _searchCtrl,
          total: bucketsAsync.whenOrNull(
                  data: (d) => d['total'] as int? ?? 0) ??
              0,
          perPage: perPage,
          currentPage: currentPage,
          onPerPageChanged: (v) {
            ref.read(_bucketPerPageProvider.notifier).state = v;
            ref.read(_bucketPageProvider.notifier).state = 1;
          },
          onPrev: () =>
              ref.read(_bucketPageProvider.notifier).update((s) => s - 1),
          onNext: () =>
              ref.read(_bucketPageProvider.notifier).update((s) => s + 1),
          onSearch: _doSearch,
          trailing: FilledButton.icon(
            onPressed: () => _showCreateBucketDialog(context, ref),
            icon: const Icon(Icons.add, size: 16),
            label: const Text('Create Bucket'),
          ),
        ),
        Expanded(
          child: Row(
            children: [
              // Buckets list
              SizedBox(
                width: 260,
                child: bucketsAsync.when(
                  loading: () =>
                      const Center(child: CircularProgressIndicator()),
                  error: (e, _) => Center(child: Text('Error: $e')),
                  data: (data) {
                    final buckets = List<Map<String, dynamic>>.from(
                        data['buckets'] ?? []);
                    if (buckets.isEmpty) {
                      return const Center(child: Text('No buckets'));
                    }
                    return ListView.builder(
                      itemCount: buckets.length,
                      itemBuilder: (context, i) {
                        final b = buckets[i];
                        final id = b['\$id'] as String;
                        return ListTile(
                          leading: const Icon(Icons.folder),
                          title: Text(b['name'] ?? id),
                          subtitle:
                              Text(id.length > 8 ? id.substring(0, 8) : id),
                          selected: selectedBucket == id,
                          onTap: () => ref
                              .read(selectedBucketProvider.notifier)
                              .state = id,
                          trailing: IconButton(
                            icon:
                                const Icon(Icons.delete_outline, size: 18),
                            onPressed: () => _deleteBucket(ref, id),
                          ),
                        );
                      },
                    );
                  },
                ),
              ),
              const VerticalDivider(width: 1),
              // Files panel
              if (selectedBucket != null)
                Expanded(child: _FilesPanel(bucketId: selectedBucket)),
              if (selectedBucket == null)
                const Expanded(
                    child: Center(child: Text('Select a bucket'))),
            ],
          ),
        ),
        SearchListFooter(
          total: bucketsAsync.whenOrNull(
                  data: (d) => d['total'] as int? ?? 0) ??
              0,
          perPage: perPage,
          currentPage: currentPage,
          onPrev: () =>
              ref.read(_bucketPageProvider.notifier).update((s) => s - 1),
          onNext: () =>
              ref.read(_bucketPageProvider.notifier).update((s) => s + 1),
          onPerPageChanged: (v) {
            ref.read(_bucketPerPageProvider.notifier).state = v;
            ref.read(_bucketPageProvider.notifier).state = 1;
          },
        ),
      ],
    );
  }

  void _showCreateBucketDialog(BuildContext context, WidgetRef ref) {
    final nameCtrl = TextEditingController();
    showAppDialog(
      context: context,
      title: 'Create Bucket',
      content: AppDialogField(
        controller: nameCtrl,
        label: 'Name',
        hint: 'Bucket name',
        autofocus: true,
      ),
      actions: [
        const AppDialogCancel(),
        AppDialogAction(
          label: 'Create',
          onTap: () async {
            final api = ref.read(apiClientProvider);
            await api.post('/storage/buckets', data: {
              'bucketId': 'unique()',
              'name': nameCtrl.text,
              'permissions': <String>[],
              'allowedFileExtensions': <String>[],
            });
            if (context.mounted) Navigator.pop(context);
            ref.invalidate(bucketsProvider);
          },
        ),
      ],
    );
  }

  Future<void> _deleteBucket(WidgetRef ref, String id) async {
    final api = ref.read(apiClientProvider);
    await api.delete('/storage/buckets/$id');
    ref.read(selectedBucketProvider.notifier).state = null;
    ref.invalidate(bucketsProvider);
  }
}

// --- Usage tab -----------------------------------------------------------

class _UsageTab extends StatelessWidget {
  const _UsageTab();

  @override
  Widget build(BuildContext context) {
    return const Center(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(Icons.bar_chart, size: 48, color: Color(0x40FFFFFF)),
          SizedBox(height: 16),
          Text('Storage usage analytics coming soon.',
              style: TextStyle(color: Color(0x40FFFFFF))),
        ],
      ),
    );
  }
}

// --- Files panel ---------------------------------------------------------

class _FilesPanel extends ConsumerWidget {
  final String bucketId;
  const _FilesPanel({required this.bucketId});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final filesAsync = ref.watch(filesProvider(bucketId));

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Padding(
          padding: const EdgeInsets.all(16),
          child: Row(
            children: [
              Text('Files',
                  style: Theme.of(context).textTheme.titleMedium),
              const Spacer(),
            ],
          ),
        ),
        Expanded(
          child: filesAsync.when(
            loading: () =>
                const Center(child: CircularProgressIndicator()),
            error: (e, _) => Center(child: Text('Error: $e')),
            data: (data) {
              final files = List<Map<String, dynamic>>.from(
                  data['files'] ?? []);
              if (files.isEmpty) {
                return const Center(child: Text('No files'));
              }
              return ListView.builder(
                padding:
                    const EdgeInsets.symmetric(horizontal: 16),
                itemCount: files.length,
                itemBuilder: (context, i) {
                  final f = files[i];
                  return ListTile(
                    leading: _iconForMime(
                        f['mimeType'] as String? ?? ''),
                    title: Text(f['name'] ?? 'Untitled'),
                    subtitle: Text(
                        '${_formatSize(f['sizeOriginal'] ?? 0)}  •  ${f['mimeType'] ?? ''}'),
                    trailing: Row(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        IconButton(
                          icon: const Icon(Icons.download),
                          tooltip: 'Download',
                          onPressed: () {
                            final fileId = f['\$id'];
                            final api = ref.read(apiClientProvider);
                            final url =
                                '${api.dio.options.baseUrl}/storage/buckets/$bucketId/files/$fileId/download';
                            ScaffoldMessenger.of(context).showSnackBar(
                              SnackBar(content: Text('Download: $url')),
                            );
                          },
                        ),
                        IconButton(
                          icon: const Icon(Icons.delete_outline),
                          onPressed: () =>
                              _deleteFile(ref, f['\$id']),
                        ),
                      ],
                    ),
                  );
                },
              );
            },
          ),
        ),
      ],
    );
  }

  Widget _iconForMime(String mime) {
    if (mime.startsWith('image/')) return const Icon(Icons.image);
    if (mime.startsWith('video/')) return const Icon(Icons.videocam);
    if (mime.contains('pdf')) return const Icon(Icons.picture_as_pdf);
    return const Icon(Icons.insert_drive_file);
  }

  String _formatSize(dynamic bytes) {
    final b = (bytes is int) ? bytes : 0;
    if (b < 1024) return '$b B';
    if (b < 1024 * 1024) return '${(b / 1024).toStringAsFixed(1)} KB';
    return '${(b / (1024 * 1024)).toStringAsFixed(1)} MB';
  }

  Future<void> _deleteFile(WidgetRef ref, String fileId) async {
    final api = ref.read(apiClientProvider);
    await api.delete('/storage/buckets/$bucketId/files/$fileId');
    ref.invalidate(filesProvider(bucketId));
  }
}
