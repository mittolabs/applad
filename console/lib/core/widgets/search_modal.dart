import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:lucide_icons/lucide_icons.dart';
import '../api/client.dart';
import '../theme/console_colors.dart';

// ── Public search item model ──────────────────────────────────────────────────

class SearchItem {
  final String category;
  final String label;
  final String? subtitle;
  final bool isCreate;
  final String? shortcut;
  final IconData? icon;
  final VoidCallback action;

  const SearchItem({
    required this.category,
    required this.label,
    required this.isCreate,
    required this.action,
    this.subtitle,
    this.shortcut,
    this.icon,
  });
}

// ── Search modal ──────────────────────────────────────────────────────────────

class SearchModal extends ConsumerStatefulWidget {
  final List<Map<String, dynamic>> projects;
  final List<Map<String, dynamic>> orgs;
  final String? projectId;
  final void Function(String path) onNavigate;
  final VoidCallback onCreateProject;
  final VoidCallback onCreateOrg;

  const SearchModal({
    super.key,
    required this.projects,
    required this.orgs,
    this.projectId,
    required this.onNavigate,
    required this.onCreateProject,
    required this.onCreateOrg,
  });

  @override
  ConsumerState<SearchModal> createState() => _SearchModalState();
}

class _SearchModalState extends ConsumerState<SearchModal> {
  final _ctrl = TextEditingController();
  final _focusNode = FocusNode();
  int _selectedIndex = 0;

  List<SearchItem> _serverResults = [];
  bool _searching = false;
  Timer? _debounce;

  @override
  void initState() {
    super.initState();
    _ctrl.addListener(_onQueryChanged);
  }

  @override
  void dispose() {
    _debounce?.cancel();
    _ctrl.dispose();
    _focusNode.dispose();
    super.dispose();
  }

  // ── Server search ─────────────────────────────────────────────────────────

  void _onQueryChanged() {
    setState(() {
      _selectedIndex = 0;
    });
    _debounce?.cancel();

    final q = _ctrl.text.trim();
    if (q.length < 2 || widget.projectId == null) {
      setState(() {
        _serverResults = [];
        _searching = false;
      });
      return;
    }

    setState(() => _searching = true);
    _debounce =
        Timer(const Duration(milliseconds: 250), () => _runSearch(q));
  }

  Future<void> _runSearch(String q) async {
    final pid = widget.projectId;
    if (pid == null) return;

    try {
      final api = ref.read(apiClientProvider);
      final res = await api.get(
        '/projects/$pid/search',
        params: {'q': q, 'limit': '20'},
      );
      final raw = List<Map<String, dynamic>>.from(
          (res.data as Map?)?['results'] as List? ?? []);

      if (!mounted) return;
      setState(() {
        _serverResults = raw.map((r) => _toItem(r, pid)).toList();
        _searching = false;
      });
    } catch (_) {
      if (mounted) setState(() => _searching = false);
    }
  }

  SearchItem _toItem(Map<String, dynamic> r, String pid) {
    final type = r['type'] as String? ?? '';
    final label = r['label'] as String? ?? '';
    final subtitle = r['subtitle'] as String?;

    final (category, icon, path) = switch (type) {
      'function' => (
          'FUNCTIONS',
          LucideIcons.code2,
          '/project/$pid/functions'
        ),
      'database' => (
          'DATABASES',
          LucideIcons.database,
          '/project/$pid/databases'
        ),
      'bucket' =>
        ('BUCKETS', LucideIcons.hardDrive, '/project/$pid/storage'),
      'workflow' => (
          'WORKFLOWS',
          LucideIcons.gitBranch,
          '/project/$pid/workflows'
        ),
      'deployment' => (
          'DEPLOYMENTS',
          LucideIcons.rocket,
          '/project/$pid/deploy'
        ),
      'user' => ('USERS', LucideIcons.user, '/project/$pid/auth'),
      _ => ('RESULTS', LucideIcons.layers, '/project/$pid/overview'),
    };

    return SearchItem(
      category: category,
      label: label,
      subtitle: (subtitle?.isNotEmpty ?? false) ? subtitle : null,
      isCreate: false,
      icon: icon,
      action: () => widget.onNavigate(path),
    );
  }

  // ── Static items (client-side, instant) ───────────────────────────────────

  String _projectPath(String section) {
    final pid = widget.projectId;
    if (pid != null) return '/project/$pid/$section';
    return '/$section';
  }

  List<SearchItem> get _staticItems => [
        SearchItem(
          category: 'NAVIGATION',
          label: 'Go to overview',
          shortcut: 'G then V',
          isCreate: false,
          icon: LucideIcons.layoutDashboard,
          action: () => widget.onNavigate(_projectPath('overview')),
        ),
        SearchItem(
          category: 'NAVIGATION',
          label: 'Go to auth',
          isCreate: false,
          icon: LucideIcons.shieldCheck,
          action: () => widget.onNavigate(_projectPath('auth')),
        ),
        SearchItem(
          category: 'NAVIGATION',
          label: 'Go to databases',
          isCreate: false,
          icon: LucideIcons.database,
          action: () => widget.onNavigate(_projectPath('databases')),
        ),
        SearchItem(
          category: 'NAVIGATION',
          label: 'Go to functions',
          isCreate: false,
          icon: LucideIcons.code2,
          action: () => widget.onNavigate(_projectPath('functions')),
        ),
        SearchItem(
          category: 'NAVIGATION',
          label: 'Go to messaging',
          isCreate: false,
          icon: LucideIcons.messageSquare,
          action: () => widget.onNavigate(_projectPath('messaging')),
        ),
        SearchItem(
          category: 'NAVIGATION',
          label: 'Go to storage',
          isCreate: false,
          icon: LucideIcons.hardDrive,
          action: () => widget.onNavigate(_projectPath('storage')),
        ),
        SearchItem(
          category: 'NAVIGATION',
          label: 'Go to deploy',
          isCreate: false,
          icon: LucideIcons.rocket,
          action: () => widget.onNavigate(_projectPath('deploy')),
        ),
        SearchItem(
          category: 'NAVIGATION',
          label: 'Go to workflows',
          isCreate: false,
          icon: LucideIcons.gitBranch,
          action: () => widget.onNavigate(_projectPath('workflows')),
        ),
        SearchItem(
          category: 'NAVIGATION',
          label: 'Go to settings',
          shortcut: 'G then S',
          isCreate: false,
          icon: LucideIcons.settings,
          action: () => widget.onNavigate(_projectPath('settings')),
        ),
        SearchItem(
          category: 'PROJECTS',
          label: 'Create project',
          isCreate: true,
          shortcut: 'C',
          icon: LucideIcons.folderPlus,
          action: widget.onCreateProject,
        ),
        for (final p in widget.projects)
          SearchItem(
            category: 'PROJECTS',
            label: p['name'] as String? ?? 'Unnamed project',
            isCreate: false,
            icon: LucideIcons.folder,
            action: () {
              final id = p[r'$id'] as String? ?? '';
              widget.onNavigate('/project/$id/overview');
            },
          ),
        SearchItem(
          category: 'ORGANIZATIONS',
          label: 'Create new organization',
          isCreate: true,
          icon: LucideIcons.building,
          action: widget.onCreateOrg,
        ),
        for (final o in widget.orgs)
          SearchItem(
            category: 'ORGANIZATIONS',
            label: o['name'] as String? ?? 'Unnamed org',
            isCreate: false,
            icon: LucideIcons.building2,
            action: () => widget.onNavigate('/projects'),
          ),
      ];

  // ── Display list ──────────────────────────────────────────────────────────

  List<Object> get _displayList {
    final q = _ctrl.text.trim().toLowerCase();

    // Static items filtered client-side instantly
    final filteredStatic = q.isEmpty
        ? _staticItems
        : _staticItems
            .where((i) => i.label.toLowerCase().contains(q))
            .toList();

    // Group static items
    final grouped = <String, List<SearchItem>>{};
    for (final item in filteredStatic) {
      grouped.putIfAbsent(item.category, () => []).add(item);
    }

    // Group server results (already filtered/ranked by DB)
    for (final item in _serverResults) {
      grouped.putIfAbsent(item.category, () => []).add(item);
    }

    const order = [
      'NAVIGATION',
      'USERS',
      'DATABASES',
      'FUNCTIONS',
      'BUCKETS',
      'WORKFLOWS',
      'DEPLOYMENTS',
      'PROJECTS',
      'ORGANIZATIONS',
    ];

    final result = <Object>[];
    for (final cat in order) {
      final items = grouped[cat];
      if (items == null || items.isEmpty) continue;
      result.add(cat);
      result.addAll(items);
    }
    return result;
  }

  List<SearchItem> get _selectable =>
      _displayList.whereType<SearchItem>().toList();

  // ── Keyboard navigation ───────────────────────────────────────────────────

  void _onKey(KeyEvent event) {
    if (event is! KeyDownEvent) return;
    final items = _selectable;
    if (event.logicalKey == LogicalKeyboardKey.arrowDown) {
      setState(() =>
          _selectedIndex = (_selectedIndex + 1).clamp(0, items.length - 1));
    } else if (event.logicalKey == LogicalKeyboardKey.arrowUp) {
      setState(() =>
          _selectedIndex = (_selectedIndex - 1).clamp(0, items.length - 1));
    } else if (event.logicalKey == LogicalKeyboardKey.enter) {
      if (items.isNotEmpty && _selectedIndex < items.length) {
        _activateItem(items[_selectedIndex]);
      }
    } else if (event.logicalKey == LogicalKeyboardKey.escape) {
      Navigator.of(context).pop();
    }
  }

  void _activateItem(SearchItem item) {
    Navigator.of(context).pop();
    item.action();
  }

  // ── Build ─────────────────────────────────────────────────────────────────

  @override
  Widget build(BuildContext context) {
    final colors = consoleColors(context);
    final display = _displayList;
    final q = _ctrl.text.trim();
    int selectableCounter = -1;

    return Align(
      alignment: const Alignment(0, -0.3),
      child: Material(
        type: MaterialType.transparency,
        child: KeyboardListener(
          focusNode: _focusNode,
          autofocus: true,
          onKeyEvent: _onKey,
          child: Container(
            width: 640,
            constraints: const BoxConstraints(maxHeight: 520),
            decoration: BoxDecoration(
              color: colors.surface,
              borderRadius: BorderRadius.circular(12),
              border: Border.all(color: colors.border),
              boxShadow: [
                BoxShadow(
                  color: Colors.black.withValues(alpha: 0.6),
                  blurRadius: 48,
                  offset: const Offset(0, 20),
                ),
              ],
            ),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                // Search input
                Container(
                  padding:
                      const EdgeInsets.symmetric(horizontal: 16),
                  child: Row(
                    children: [
                      Icon(LucideIcons.search,
                          size: 16, color: colors.textMuted),
                      const SizedBox(width: 12),
                      Expanded(
                        child: TextField(
                          controller: _ctrl,
                          autofocus: true,
                          style: TextStyle(
                              color: colors.textPrimary, fontSize: 14),
                          decoration: InputDecoration(
                            hintText: widget.projectId != null
                                ? 'Search pages, resources, users…'
                                : 'Search projects, organizations…',
                            hintStyle: TextStyle(
                                color: colors.textSubtle, fontSize: 14),
                            border: InputBorder.none,
                            enabledBorder: InputBorder.none,
                            focusedBorder: InputBorder.none,
                            filled: false,
                            contentPadding: const EdgeInsets.symmetric(
                                vertical: 10),
                          ),
                          onSubmitted: (_) {
                            final items = _selectable;
                            if (items.isNotEmpty &&
                                _selectedIndex < items.length) {
                              _activateItem(items[_selectedIndex]);
                            }
                          },
                        ),
                      ),
                      if (_searching)
                        const SizedBox(
                          width: 14,
                          height: 14,
                          child: CircularProgressIndicator(
                              strokeWidth: 1.5),
                        )
                      else
                        const SearchHintBadge(label: 'Esc'),
                    ],
                  ),
                ),

                Container(height: 1, color: colors.border),

                // Results
                if (display.isEmpty && !_searching)
                  Padding(
                    padding: const EdgeInsets.all(40),
                    child: Column(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        Icon(LucideIcons.search,
                            size: 28, color: colors.textSubtle),
                        const SizedBox(height: 12),
                        Text(
                          q.isEmpty
                              ? 'Start typing to search…'
                              : 'No results for "$q"',
                          style: TextStyle(
                              color: colors.textMuted, fontSize: 13),
                        ),
                      ],
                    ),
                  )
                else
                  Flexible(
                    child: ListView.builder(
                      padding:
                          const EdgeInsets.symmetric(vertical: 6),
                      shrinkWrap: true,
                      itemCount: display.length,
                      itemBuilder: (ctx, i) {
                        final entry = display[i];
                        if (entry is String) {
                          return _CategoryHeader(
                            label: entry,
                            colors: colors,
                          );
                        }

                        selectableCounter++;
                        final itemIndex = selectableCounter;
                        final item = entry as SearchItem;
                        final isSelected = itemIndex == _selectedIndex;

                        return SearchItemTile(
                          item: item,
                          isSelected: isSelected,
                          onTap: () {
                            Navigator.of(context).pop();
                            item.action();
                          },
                          onHover: () => setState(
                              () => _selectedIndex = itemIndex),
                        );
                      },
                    ),
                  ),

                Container(height: 1, color: colors.border),

                // Bottom hints
                Padding(
                  padding: const EdgeInsets.symmetric(
                      horizontal: 16, vertical: 10),
                  child: Row(
                    children: [
                      const SearchHintBadge(label: '↵'),
                      const SizedBox(width: 6),
                      Text('select',
                          style: TextStyle(
                              color: colors.textSubtle, fontSize: 11)),
                      const SizedBox(width: 16),
                      const SearchHintBadge(label: '↑↓'),
                      const SizedBox(width: 6),
                      Text('navigate',
                          style: TextStyle(
                              color: colors.textSubtle, fontSize: 11)),
                      const Spacer(),
                      if (widget.projectId != null &&
                          q.isNotEmpty &&
                          q.length < 2)
                        Text(
                          'type one more character to search data',
                          style: TextStyle(
                              color: colors.textSubtle, fontSize: 11),
                        ),
                    ],
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

// ── Category header ───────────────────────────────────────────────────────────

class _CategoryHeader extends StatelessWidget {
  final String label;
  final ConsoleColors colors;

  const _CategoryHeader({required this.label, required this.colors});

  static const _icons = <String, IconData>{
    'NAVIGATION': LucideIcons.compass,
    'PROJECTS': LucideIcons.folder,
    'ORGANIZATIONS': LucideIcons.building2,
    'USERS': LucideIcons.users,
    'DATABASES': LucideIcons.database,
    'FUNCTIONS': LucideIcons.code2,
    'BUCKETS': LucideIcons.hardDrive,
    'WORKFLOWS': LucideIcons.gitBranch,
    'DEPLOYMENTS': LucideIcons.rocket,
  };

  @override
  Widget build(BuildContext context) {
    final icon = _icons[label];
    return Padding(
      padding:
          const EdgeInsets.only(left: 16, top: 12, bottom: 4, right: 16),
      child: Row(children: [
        if (icon != null) ...[
          Icon(icon, size: 10, color: colors.textMuted),
          const SizedBox(width: 6),
        ],
        Text(
          label,
          style: TextStyle(
            color: colors.textMuted,
            fontSize: 11,
            fontWeight: FontWeight.w600,
            letterSpacing: 0.8,
          ),
        ),
      ]),
    );
  }
}

// ── Search item tile ──────────────────────────────────────────────────────────

class SearchItemTile extends StatelessWidget {
  final SearchItem item;
  final bool isSelected;
  final VoidCallback onTap;
  final VoidCallback onHover;

  const SearchItemTile({
    super.key,
    required this.item,
    required this.isSelected,
    required this.onTap,
    required this.onHover,
  });

  @override
  Widget build(BuildContext context) {
    final colors = consoleColors(context);
    final IconData iconData = item.icon ??
        (item.isCreate ? LucideIcons.plus : LucideIcons.arrowRight);
    final Color iconColor = item.isCreate
        ? const Color(0xFF3472A4)
        : (item.icon != null ? colors.textSecondary : colors.textMuted);
    final Color iconBg = item.isCreate
        ? const Color(0xFF3472A4).withValues(alpha: 0.12)
        : colors.fill;

    return MouseRegion(
      onEnter: (_) => onHover(),
      cursor: SystemMouseCursors.click,
      child: GestureDetector(
        onTap: onTap,
        child: Container(
          margin: const EdgeInsets.symmetric(horizontal: 6, vertical: 1),
          padding:
              const EdgeInsets.symmetric(horizontal: 12, vertical: 9),
          decoration: BoxDecoration(
            color:
                isSelected ? colors.fillActive : Colors.transparent,
            borderRadius: BorderRadius.circular(8),
          ),
          child: Row(
            children: [
              Container(
                width: 28,
                height: 28,
                decoration: BoxDecoration(
                  color: iconBg,
                  borderRadius: BorderRadius.circular(6),
                ),
                child: Center(
                  child:
                      Icon(iconData, size: 13, color: iconColor),
                ),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    Text(
                      item.label,
                      style: TextStyle(
                        color: isSelected
                            ? colors.textPrimary
                            : colors.textSecondary,
                        fontSize: 13,
                        fontWeight: isSelected
                            ? FontWeight.w500
                            : FontWeight.w400,
                      ),
                    ),
                    if (item.subtitle != null &&
                        item.subtitle!.isNotEmpty) ...[
                      const SizedBox(height: 1),
                      Text(
                        item.subtitle!,
                        style: TextStyle(
                            color: colors.textSubtle, fontSize: 11),
                      ),
                    ],
                  ],
                ),
              ),
              if (item.shortcut != null) ...[
                const SizedBox(width: 8),
                _buildShortcut(context, item.shortcut!),
              ],
            ],
          ),
        ),
      ),
    );
  }

  Widget _buildShortcut(BuildContext context, String shortcut) {
    final parts = shortcut.split(' then ');
    if (parts.length == 1) return SearchHintBadge(label: parts[0]);
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        SearchHintBadge(label: parts[0]),
        Padding(
          padding: const EdgeInsets.symmetric(horizontal: 4),
          child: Text('then',
              style: TextStyle(
                  color: consoleColors(context).textSubtle,
                  fontSize: 11)),
        ),
        SearchHintBadge(label: parts[1]),
      ],
    );
  }
}

// ── Hint badge ────────────────────────────────────────────────────────────────

class SearchHintBadge extends StatelessWidget {
  final String label;
  const SearchHintBadge({super.key, required this.label});

  @override
  Widget build(BuildContext context) {
    final colors = consoleColors(context);
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
      decoration: BoxDecoration(
        color: colors.badgeFill,
        borderRadius: BorderRadius.circular(4),
        border: Border.all(color: colors.border),
      ),
      child: Text(
        label,
        style: TextStyle(
          color: colors.textMuted,
          fontSize: 10,
          fontWeight: FontWeight.w500,
        ),
      ),
    );
  }
}
