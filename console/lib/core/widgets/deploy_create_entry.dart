import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:lucide_icons/lucide_icons.dart';
import '../api/client.dart';
import 'app_dialog.dart';

const _bg = Color(0xFF0B0B0F);
const _surface = Color(0xFF16171B);
const _accent = Color(0xFF3472A4);
const _border = Color(0x14FFFFFF);
const _dimText = Color(0x80FFFFFF);
const _subtleText = Color(0x40FFFFFF);
const _fieldFill = Color(0x0AFFFFFF);
const _fieldBorder = Color(0x1AFFFFFF);

/// Result of the 3-option entry point dialog.
enum CreateEntryChoice { template, repository, upload }

class CreateEntryResult {
  final CreateEntryChoice choice;
  final Map<String, dynamic>? templateConfig;
  final Map<String, dynamic>? repoConfig;

  const CreateEntryResult({required this.choice, this.templateConfig, this.repoConfig});
}

/// Shows the 3-option entry point dialog used by Sites, Containers, Mobile, and Desktop.
///
/// [category] is one of: sites, containers, mobile, desktop
/// [title] is the dialog title, e.g. "Create site"
/// Returns a [CreateEntryResult] or null if cancelled.
Future<CreateEntryResult?> showCreateEntryDialog({
  required BuildContext context,
  required WidgetRef ref,
  required String category,
  required String title,
  required String subtitle,
}) {
  return showDialog<CreateEntryResult>(
    context: context,
    barrierColor: Colors.black.withOpacity(0.6),
    builder: (ctx) => _CreateEntryDialog(
      category: category,
      title: title,
      subtitle: subtitle,
      ref: ref,
    ),
  );
}

class _CreateEntryDialog extends StatefulWidget {
  final String category;
  final String title;
  final String subtitle;
  final WidgetRef ref;

  const _CreateEntryDialog({
    required this.category,
    required this.title,
    required this.subtitle,
    required this.ref,
  });

  @override
  State<_CreateEntryDialog> createState() => _CreateEntryDialogState();
}

class _CreateEntryDialogState extends State<_CreateEntryDialog> {
  String _view = 'entry'; // entry | templates | repo | upload
  List<Map<String, dynamic>> _templates = [];
  List<Map<String, dynamic>> _connections = [];
  List<Map<String, dynamic>> _repos = [];
  bool _loadingTemplates = false;
  bool _loadingConnections = false;
  bool _loadingRepos = false;
  String _templateSearch = '';
  String _templateFilter = '';
  String _repoSearch = '';
  String? _selectedConnectionId;
  final _repoSearchCtrl = TextEditingController();
  final _templateSearchCtrl = TextEditingController();

  @override
  void dispose() {
    _repoSearchCtrl.dispose();
    _templateSearchCtrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Material(
        color: Colors.transparent,
        child: Container(
          width: _view == 'entry' ? 540 : 640,
          constraints: const BoxConstraints(maxHeight: 600),
          decoration: BoxDecoration(
            color: _surface,
            borderRadius: BorderRadius.circular(12),
            border: Border.all(color: Colors.white.withOpacity(0.08)),
            boxShadow: [
              BoxShadow(color: Colors.black.withOpacity(0.5), blurRadius: 32, offset: const Offset(0, 8)),
            ],
          ),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              // Header
              Padding(
                padding: const EdgeInsets.fromLTRB(20, 20, 20, 0),
                child: Row(children: [
                  if (_view != 'entry')
                    GestureDetector(
                      onTap: () => setState(() => _view = 'entry'),
                      child: Padding(
                        padding: const EdgeInsets.only(right: 10),
                        child: Icon(LucideIcons.arrowLeft, size: 16, color: Colors.white.withOpacity(0.4)),
                      ),
                    ),
                  Expanded(
                    child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                      Text(
                        _view == 'entry' ? widget.title
                          : _view == 'templates' ? 'Clone a template'
                          : _view == 'repo' ? 'Connect a repository'
                          : 'Manual upload',
                        style: const TextStyle(color: Colors.white, fontSize: 16, fontWeight: FontWeight.w600),
                      ),
                      const SizedBox(height: 4),
                      Text(
                        _view == 'entry' ? widget.subtitle
                          : _view == 'templates' ? 'Choose a template to get started quickly'
                          : _view == 'repo' ? 'Import from a Git repository'
                          : 'Upload your project files directly',
                        style: TextStyle(color: Colors.white.withOpacity(0.45), fontSize: 13),
                      ),
                    ]),
                  ),
                  GestureDetector(
                    onTap: () => Navigator.of(context).pop(),
                    child: Icon(LucideIcons.x, size: 16, color: Colors.white.withOpacity(0.3)),
                  ),
                ]),
              ),
              const SizedBox(height: 16),
              Padding(
                padding: const EdgeInsets.symmetric(horizontal: 20),
                child: Container(height: 1, color: Colors.white.withOpacity(0.06)),
              ),
              const SizedBox(height: 16),
              // Content
              Flexible(
                child: SingleChildScrollView(
                  padding: const EdgeInsets.fromLTRB(20, 0, 20, 20),
                  child: _view == 'entry' ? _entryView()
                    : _view == 'templates' ? _templatesView()
                    : _view == 'repo' ? _repoView()
                    : _uploadView(),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }

  Widget _entryView() {
    return Column(mainAxisSize: MainAxisSize.min, children: [
      _entryOption(
        icon: LucideIcons.layoutTemplate,
        title: 'Clone a template',
        description: 'Start with a pre-configured template for your framework',
        onTap: () {
          setState(() => _view = 'templates');
          _loadTemplates();
        },
      ),
      const SizedBox(height: 10),
      _entryOption(
        icon: LucideIcons.gitBranch,
        title: 'Connect a repository',
        description: 'Import an existing project from GitHub or another Git provider',
        onTap: () {
          setState(() => _view = 'repo');
          _loadConnections();
        },
      ),
      const SizedBox(height: 10),
      _entryOption(
        icon: LucideIcons.upload,
        title: 'Manual upload',
        description: 'Upload your project files or build output directly',
        onTap: () {
          Navigator.of(context).pop(
            const CreateEntryResult(choice: CreateEntryChoice.upload),
          );
        },
      ),
    ]);
  }

  Widget _entryOption({
    required IconData icon,
    required String title,
    required String description,
    required VoidCallback onTap,
  }) {
    return GestureDetector(
      onTap: onTap,
      child: MouseRegion(
        cursor: SystemMouseCursors.click,
        child: Container(
          width: double.infinity,
          padding: const EdgeInsets.all(16),
          decoration: BoxDecoration(
            color: _bg,
            borderRadius: BorderRadius.circular(8),
            border: Border.all(color: _border),
          ),
          child: Row(children: [
            Container(
              width: 44, height: 44,
              decoration: BoxDecoration(color: _accent.withOpacity(0.1), borderRadius: BorderRadius.circular(10)),
              child: Icon(icon, size: 20, color: _accent),
            ),
            const SizedBox(width: 14),
            Expanded(
              child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                Text(title, style: const TextStyle(color: Colors.white, fontSize: 14, fontWeight: FontWeight.w500)),
                const SizedBox(height: 3),
                Text(description, style: TextStyle(color: Colors.white.withOpacity(0.4), fontSize: 12)),
              ]),
            ),
            Icon(LucideIcons.chevronRight, size: 16, color: Colors.white.withOpacity(0.2)),
          ]),
        ),
      ),
    );
  }

  // ── Templates view ──────────────────────────────────────────────────────────

  void _loadTemplates() async {
    if (_loadingTemplates) return;
    setState(() => _loadingTemplates = true);
    try {
      final api = widget.ref.read(apiClientProvider);
      final res = await api.get('/deploy/templates?category=${widget.category}');
      final data = res.data as Map<String, dynamic>;
      setState(() {
        _templates = List<Map<String, dynamic>>.from(data['templates'] ?? []);
        _loadingTemplates = false;
      });
    } catch (_) {
      setState(() => _loadingTemplates = false);
    }
  }

  Widget _templatesView() {
    if (_loadingTemplates) {
      return const Padding(
        padding: EdgeInsets.symmetric(vertical: 40),
        child: Center(child: CircularProgressIndicator(color: _accent)),
      );
    }

    // Collect unique frameworks for filter
    final frameworks = <String>{};
    for (final t in _templates) {
      final fw = t['framework'] as String?;
      if (fw != null && fw.isNotEmpty) frameworks.add(fw);
    }

    // Apply filters
    var filtered = _templates;
    if (_templateSearch.isNotEmpty) {
      final q = _templateSearch.toLowerCase();
      filtered = filtered.where((t) {
        final name = (t['name'] ?? '').toString().toLowerCase();
        final desc = (t['description'] ?? '').toString().toLowerCase();
        return name.contains(q) || desc.contains(q);
      }).toList();
    }
    if (_templateFilter.isNotEmpty) {
      filtered = filtered.where((t) => t['framework'] == _templateFilter).toList();
    }

    return Column(mainAxisSize: MainAxisSize.min, children: [
      // Search bar
      TextField(
        controller: _templateSearchCtrl,
        style: const TextStyle(color: Colors.white, fontSize: 13),
        onChanged: (v) => setState(() => _templateSearch = v),
        decoration: InputDecoration(
          hintText: 'Search templates...',
          hintStyle: TextStyle(color: Colors.white.withOpacity(0.22), fontSize: 13),
          prefixIcon: Icon(LucideIcons.search, size: 16, color: Colors.white.withOpacity(0.3)),
          filled: true,
          fillColor: _fieldFill,
          contentPadding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
          border: OutlineInputBorder(borderRadius: BorderRadius.circular(8), borderSide: const BorderSide(color: _fieldBorder)),
          enabledBorder: OutlineInputBorder(borderRadius: BorderRadius.circular(8), borderSide: const BorderSide(color: _fieldBorder)),
          focusedBorder: OutlineInputBorder(borderRadius: BorderRadius.circular(8), borderSide: const BorderSide(color: _accent)),
        ),
      ),
      const SizedBox(height: 12),
      // Framework filter chips
      if (frameworks.isNotEmpty)
        Padding(
          padding: const EdgeInsets.only(bottom: 12),
          child: SingleChildScrollView(
            scrollDirection: Axis.horizontal,
            child: Row(children: [
              _filterChip('All', _templateFilter.isEmpty, () => setState(() => _templateFilter = '')),
              ...frameworks.map((fw) => Padding(
                padding: const EdgeInsets.only(left: 6),
                child: _filterChip(fw, _templateFilter == fw, () => setState(() => _templateFilter = fw)),
              )),
            ]),
          ),
        ),
      // Template grid
      if (filtered.isEmpty)
        Padding(
          padding: const EdgeInsets.symmetric(vertical: 32),
          child: Text(
            _templates.isEmpty ? 'No templates available' : 'No templates match your search',
            style: const TextStyle(color: _dimText, fontSize: 13),
          ),
        )
      else
        Wrap(
          spacing: 10,
          runSpacing: 10,
          children: filtered.map((t) => _templateCard(t)).toList(),
        ),
    ]);
  }

  Widget _filterChip(String label, bool active, VoidCallback onTap) {
    return GestureDetector(
      onTap: onTap,
      child: MouseRegion(
        cursor: SystemMouseCursors.click,
        child: Container(
          padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 5),
          decoration: BoxDecoration(
            color: active ? _accent.withOpacity(0.15) : _bg,
            borderRadius: BorderRadius.circular(6),
            border: Border.all(color: active ? _accent.withOpacity(0.4) : _border),
          ),
          child: Text(label, style: TextStyle(color: active ? _accent : _dimText, fontSize: 12, fontWeight: active ? FontWeight.w500 : FontWeight.w400)),
        ),
      ),
    );
  }

  Widget _templateCard(Map<String, dynamic> t) {
    return GestureDetector(
      onTap: () {
        Navigator.of(context).pop(
          CreateEntryResult(choice: CreateEntryChoice.template, templateConfig: t),
        );
      },
      child: MouseRegion(
        cursor: SystemMouseCursors.click,
        child: Container(
          width: 185,
          padding: const EdgeInsets.all(14),
          decoration: BoxDecoration(
            color: _bg,
            borderRadius: BorderRadius.circular(8),
            border: Border.all(color: _border),
          ),
          child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
            Container(
              width: 36, height: 36,
              decoration: BoxDecoration(color: _accent.withOpacity(0.1), borderRadius: BorderRadius.circular(8)),
              child: Icon(LucideIcons.layoutTemplate, size: 18, color: _accent),
            ),
            const SizedBox(height: 10),
            Text(t['name'] ?? '', style: const TextStyle(color: Colors.white, fontSize: 13, fontWeight: FontWeight.w500)),
            const SizedBox(height: 4),
            Text(
              t['description'] ?? '',
              maxLines: 2,
              overflow: TextOverflow.ellipsis,
              style: TextStyle(color: Colors.white.withOpacity(0.35), fontSize: 11),
            ),
            if (t['framework'] != null) ...[
              const SizedBox(height: 8),
              Container(
                padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
                decoration: BoxDecoration(color: _accent.withOpacity(0.1), borderRadius: BorderRadius.circular(4)),
                child: Text(t['framework'], style: const TextStyle(color: _accent, fontSize: 10)),
              ),
            ],
          ]),
        ),
      ),
    );
  }

  // ── Repository view ─────────────────────────────────────────────────────────

  void _loadConnections() async {
    if (_loadingConnections) return;
    setState(() => _loadingConnections = true);
    try {
      final api = widget.ref.read(apiClientProvider);
      final res = await api.get('/deploy/git/connections');
      final data = res.data as Map<String, dynamic>;
      final list = List<Map<String, dynamic>>.from(data['connections'] ?? []);
      setState(() {
        _connections = list;
        _loadingConnections = false;
        if (list.isNotEmpty) {
          _selectedConnectionId = list.first['\$id'] as String?;
          _loadRepos();
        }
      });
    } catch (_) {
      setState(() => _loadingConnections = false);
    }
  }

  void _loadRepos() async {
    if (_selectedConnectionId == null) return;
    setState(() => _loadingRepos = true);
    try {
      final api = widget.ref.read(apiClientProvider);
      final res = await api.get('/deploy/git/connections/$_selectedConnectionId/repos');
      final data = res.data as Map<String, dynamic>;
      setState(() {
        _repos = List<Map<String, dynamic>>.from(data['repos'] ?? []);
        _loadingRepos = false;
      });
    } catch (_) {
      setState(() => _loadingRepos = false);
    }
  }

  Widget _repoView() {
    if (_loadingConnections) {
      return const Padding(
        padding: EdgeInsets.symmetric(vertical: 40),
        child: Center(child: CircularProgressIndicator(color: _accent)),
      );
    }

    if (_connections.isEmpty) {
      return Column(mainAxisSize: MainAxisSize.min, children: [
        const SizedBox(height: 20),
        Container(
          width: 64, height: 64,
          decoration: BoxDecoration(color: _accent.withOpacity(0.1), shape: BoxShape.circle),
          child: const Icon(LucideIcons.gitBranch, size: 28, color: _accent),
        ),
        const SizedBox(height: 16),
        const Text('No Git connections', style: TextStyle(color: Colors.white, fontSize: 15, fontWeight: FontWeight.w600)),
        const SizedBox(height: 6),
        Text('Connect your GitHub account to import repositories', style: TextStyle(color: Colors.white.withOpacity(0.4), fontSize: 13)),
        const SizedBox(height: 20),
        FilledButton.icon(
          style: FilledButton.styleFrom(backgroundColor: _accent, shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(8))),
          icon: const Icon(LucideIcons.github, size: 16),
          label: const Text('Connect to GitHub', style: TextStyle(fontSize: 13)),
          onPressed: () {
            // TODO: open GitHub OAuth flow
          },
        ),
        const SizedBox(height: 20),
      ]);
    }

    return Column(mainAxisSize: MainAxisSize.min, children: [
      // Connection dropdown
      Container(
        width: double.infinity,
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 4),
        decoration: BoxDecoration(
          color: _fieldFill,
          borderRadius: BorderRadius.circular(8),
          border: Border.all(color: _fieldBorder),
        ),
        child: DropdownButtonHideUnderline(
          child: DropdownButton<String>(
            isExpanded: true,
            dropdownColor: _surface,
            value: _selectedConnectionId,
            style: const TextStyle(color: Colors.white, fontSize: 13),
            icon: Icon(LucideIcons.chevronDown, size: 14, color: Colors.white.withOpacity(0.3)),
            items: _connections.map((c) {
              return DropdownMenuItem<String>(
                value: c['\$id'] as String,
                child: Row(children: [
                  const Icon(LucideIcons.github, size: 14, color: _dimText),
                  const SizedBox(width: 8),
                  Text(c['name'] ?? c['login'] ?? c['\$id'] ?? '', style: const TextStyle(color: Colors.white, fontSize: 13)),
                ]),
              );
            }).toList(),
            onChanged: (v) {
              setState(() {
                _selectedConnectionId = v;
                _repos = [];
              });
              _loadRepos();
            },
          ),
        ),
      ),
      const SizedBox(height: 12),
      // Search repos
      TextField(
        controller: _repoSearchCtrl,
        style: const TextStyle(color: Colors.white, fontSize: 13),
        onChanged: (v) => setState(() => _repoSearch = v),
        decoration: InputDecoration(
          hintText: 'Search repositories...',
          hintStyle: TextStyle(color: Colors.white.withOpacity(0.22), fontSize: 13),
          prefixIcon: Icon(LucideIcons.search, size: 16, color: Colors.white.withOpacity(0.3)),
          filled: true,
          fillColor: _fieldFill,
          contentPadding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
          border: OutlineInputBorder(borderRadius: BorderRadius.circular(8), borderSide: const BorderSide(color: _fieldBorder)),
          enabledBorder: OutlineInputBorder(borderRadius: BorderRadius.circular(8), borderSide: const BorderSide(color: _fieldBorder)),
          focusedBorder: OutlineInputBorder(borderRadius: BorderRadius.circular(8), borderSide: const BorderSide(color: _accent)),
        ),
      ),
      const SizedBox(height: 12),
      // Repo list
      if (_loadingRepos)
        const Padding(
          padding: EdgeInsets.symmetric(vertical: 24),
          child: Center(child: CircularProgressIndicator(color: _accent)),
        )
      else ...[
        ..._filteredRepos().map((repo) => Padding(
          padding: const EdgeInsets.only(bottom: 6),
          child: _repoCard(repo),
        )),
        if (_filteredRepos().isEmpty)
          Padding(
            padding: const EdgeInsets.symmetric(vertical: 20),
            child: Text(
              _repos.isEmpty ? 'No repositories found' : 'No repositories match your search',
              style: const TextStyle(color: _dimText, fontSize: 13),
            ),
          ),
      ],
    ]);
  }

  List<Map<String, dynamic>> _filteredRepos() {
    if (_repoSearch.isEmpty) return _repos;
    final q = _repoSearch.toLowerCase();
    return _repos.where((r) {
      final name = (r['name'] ?? '').toString().toLowerCase();
      final full = (r['fullName'] ?? '').toString().toLowerCase();
      return name.contains(q) || full.contains(q);
    }).toList();
  }

  Widget _repoCard(Map<String, dynamic> repo) {
    return GestureDetector(
      onTap: () {
        Navigator.of(context).pop(
          CreateEntryResult(choice: CreateEntryChoice.repository, repoConfig: repo),
        );
      },
      child: MouseRegion(
        cursor: SystemMouseCursors.click,
        child: Container(
          width: double.infinity,
          padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 12),
          decoration: BoxDecoration(
            color: _bg,
            borderRadius: BorderRadius.circular(8),
            border: Border.all(color: _border),
          ),
          child: Row(children: [
            const Icon(LucideIcons.gitBranch, size: 16, color: _accent),
            const SizedBox(width: 12),
            Expanded(
              child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                Text(repo['name'] ?? '', style: const TextStyle(color: Colors.white, fontSize: 13, fontWeight: FontWeight.w500)),
                if (repo['description'] != null && (repo['description'] as String).isNotEmpty)
                  Text(repo['description'], maxLines: 1, overflow: TextOverflow.ellipsis,
                    style: TextStyle(color: Colors.white.withOpacity(0.35), fontSize: 11)),
              ]),
            ),
            Container(
              padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 5),
              decoration: BoxDecoration(color: _accent.withOpacity(0.15), borderRadius: BorderRadius.circular(6)),
              child: const Text('Connect', style: TextStyle(color: _accent, fontSize: 11, fontWeight: FontWeight.w500)),
            ),
          ]),
        ),
      ),
    );
  }

  // ── Upload view ─────────────────────────────────────────────────────────────

  Widget _uploadView() {
    return Column(mainAxisSize: MainAxisSize.min, children: [
      const SizedBox(height: 8),
      Container(
        width: double.infinity,
        height: 160,
        decoration: BoxDecoration(
          color: _bg,
          borderRadius: BorderRadius.circular(8),
          border: Border.all(color: _border, style: BorderStyle.solid),
        ),
        child: Column(mainAxisAlignment: MainAxisAlignment.center, children: [
          Icon(LucideIcons.uploadCloud, size: 36, color: Colors.white.withOpacity(0.25)),
          const SizedBox(height: 12),
          Text('Drag & drop your project files\nor click to browse',
            textAlign: TextAlign.center,
            style: TextStyle(color: Colors.white.withOpacity(0.4), fontSize: 13)),
          const SizedBox(height: 14),
          OutlinedButton.icon(
            style: OutlinedButton.styleFrom(
              foregroundColor: Colors.white70,
              side: BorderSide(color: Colors.white.withOpacity(0.12)),
              padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
            ),
            icon: const Icon(LucideIcons.folderOpen, size: 14),
            label: const Text('Browse files', style: TextStyle(fontSize: 12)),
            onPressed: () {
              Navigator.of(context).pop(
                const CreateEntryResult(choice: CreateEntryChoice.upload),
              );
            },
          ),
        ]),
      ),
      const SizedBox(height: 8),
    ]);
  }
}
