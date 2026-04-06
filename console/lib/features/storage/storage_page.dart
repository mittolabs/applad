import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../core/api/client.dart';

final bucketsProvider = FutureProvider<Map<String, dynamic>>((ref) async {
  final api = ref.read(apiClientProvider);
  final res = await api.get('/storage/buckets');
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

class StoragePage extends ConsumerWidget {
  const StoragePage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final bucketsAsync = ref.watch(bucketsProvider);
    final selectedBucket = ref.watch(selectedBucketProvider);

    return Scaffold(
      appBar: AppBar(
        title: const Text('Storage'),
        actions: [
          FilledButton.icon(
            onPressed: () => _showCreateBucketDialog(context, ref),
            icon: const Icon(Icons.add),
            label: const Text('Create Bucket'),
          ),
          const SizedBox(width: 16),
        ],
      ),
      body: Row(
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
                      subtitle: Text(id.length > 8 ? id.substring(0, 8) : id),
                      selected: selectedBucket == id,
                      onTap: () => ref
                          .read(selectedBucketProvider.notifier)
                          .state = id,
                      trailing: IconButton(
                        icon: const Icon(Icons.delete_outline,
                            size: 18),
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
            Expanded(
                child: _FilesPanel(bucketId: selectedBucket)),
          if (selectedBucket == null)
            const Expanded(
                child: Center(child: Text('Select a bucket'))),
        ],
      ),
    );
  }

  void _showCreateBucketDialog(BuildContext context, WidgetRef ref) {
    final nameCtrl = TextEditingController();
    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('Create Bucket'),
        content: TextField(
          controller: nameCtrl,
          decoration: const InputDecoration(labelText: 'Name'),
        ),
        actions: [
          TextButton(
              onPressed: () => Navigator.pop(ctx),
              child: const Text('Cancel')),
          FilledButton(
            onPressed: () async {
              final api = ref.read(apiClientProvider);
              await api.post('/storage/buckets', data: {
                'bucketId': 'unique()',
                'name': nameCtrl.text,
                'permissions': <String>[],
                'allowedFileExtensions': <String>[],
              });
              if (ctx.mounted) Navigator.pop(ctx);
              ref.invalidate(bucketsProvider);
            },
            child: const Text('Create'),
          ),
        ],
      ),
    );
  }

  Future<void> _deleteBucket(WidgetRef ref, String id) async {
    final api = ref.read(apiClientProvider);
    await api.delete('/storage/buckets/$id');
    ref.read(selectedBucketProvider.notifier).state = null;
    ref.invalidate(bucketsProvider);
  }
}

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
                            // Trigger browser download via API URL
                            final fileId = f['\$id'];
                            final api = ref.read(apiClientProvider);
                            final url =
                                '${api.dio.options.baseUrl}/storage/buckets/$bucketId/files/$fileId/download';
                            // On web, open download URL in new tab
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
    if (mime.startsWith('image/')) {
      return const Icon(Icons.image);
    }
    if (mime.startsWith('video/')) {
      return const Icon(Icons.videocam);
    }
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
