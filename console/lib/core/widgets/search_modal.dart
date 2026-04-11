import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:lucide_icons/lucide_icons.dart';
import '../theme/console_colors.dart';

// ── Public search item model ──────────────────────────────────────────────────

class SearchItem {
  final String category;
  final String label;
  final bool isCreate;
  final String? shortcut;
  final VoidCallback action;

  const SearchItem({
    required this.category,
    required this.label,
    required this.isCreate,
    required this.action,
    this.shortcut,
  });
}

// ── Search modal ──────────────────────────────────────────────────────────────

class SearchModal extends StatefulWidget {
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
  State<SearchModal> createState() => _SearchModalState();
}

class _SearchModalState extends State<SearchModal> {
  final _ctrl = TextEditingController();
  final _focusNode = FocusNode();
  int _selectedIndex = 0;

  @override
  void initState() {
    super.initState();
    _ctrl.addListener(() => setState(() => _selectedIndex = 0));
  }

  @override
  void dispose() {
    _ctrl.dispose();
    _focusNode.dispose();
    super.dispose();
  }

  String _projectPath(String section) {
    final pid = widget.projectId;
    if (pid != null) return '/project/$pid/$section';
    return '/$section';
  }

  List<SearchItem> get _allItems => [
        // PROJECTS
        SearchItem(
          category: 'PROJECTS',
          label: 'Create project',
          isCreate: true,
          shortcut: 'C',
          action: widget.onCreateProject,
        ),
        SearchItem(
          category: 'PROJECTS',
          label: 'Find projects',
          isCreate: false,
          action: () => widget.onNavigate('/projects'),
        ),
        for (final p in widget.projects)
          SearchItem(
            category: 'PROJECTS',
            label: p['name'] ?? 'Unnamed project',
            isCreate: false,
            action: () {
              final id = p['\$id'] as String? ?? '';
              widget.onNavigate('/project/$id/overview');
            },
          ),

        // ORGANIZATIONS
        SearchItem(
          category: 'ORGANIZATIONS',
          label: 'Create new organization',
          isCreate: true,
          shortcut: 'C then O',
          action: widget.onCreateOrg,
        ),
        SearchItem(
          category: 'ORGANIZATIONS',
          label: 'Find organizations',
          isCreate: false,
          action: () => widget.onNavigate('/projects'),
        ),
        for (final o in widget.orgs)
          SearchItem(
            category: 'ORGANIZATIONS',
            label: o['name'] ?? 'Unnamed org',
            isCreate: false,
            action: () => widget.onNavigate('/projects'),
          ),

        // NAVIGATION
        SearchItem(
          category: 'NAVIGATION',
          label: 'Go to overview',
          isCreate: false,
          shortcut: 'G then V',
          action: () => widget.onNavigate(_projectPath('overview')),
        ),
        SearchItem(
          category: 'NAVIGATION',
          label: 'Go to auth',
          isCreate: false,
          action: () => widget.onNavigate(_projectPath('auth')),
        ),
        SearchItem(
          category: 'NAVIGATION',
          label: 'Go to databases',
          isCreate: false,
          action: () => widget.onNavigate(_projectPath('databases')),
        ),
        SearchItem(
          category: 'NAVIGATION',
          label: 'Go to functions',
          isCreate: false,
          action: () => widget.onNavigate(_projectPath('functions')),
        ),
        SearchItem(
          category: 'NAVIGATION',
          label: 'Go to messaging',
          isCreate: false,
          action: () => widget.onNavigate(_projectPath('messaging')),
        ),
        SearchItem(
          category: 'NAVIGATION',
          label: 'Go to storage',
          isCreate: false,
          action: () => widget.onNavigate(_projectPath('storage')),
        ),
        SearchItem(
          category: 'NAVIGATION',
          label: 'Go to deploy',
          isCreate: false,
          action: () => widget.onNavigate(_projectPath('deploy')),
        ),
        SearchItem(
          category: 'NAVIGATION',
          label: 'Go to health',
          isCreate: false,
          action: () => widget.onNavigate(_projectPath('health')),
        ),
        SearchItem(
          category: 'NAVIGATION',
          label: 'Go to workflows',
          isCreate: false,
          action: () => widget.onNavigate(_projectPath('workflows')),
        ),
        SearchItem(
          category: 'NAVIGATION',
          label: 'Go to settings',
          isCreate: false,
          shortcut: 'G then S',
          action: () => widget.onNavigate(_projectPath('settings')),
        ),
      ];

  List<SearchItem> get _filtered {
    final q = _ctrl.text.trim().toLowerCase();
    if (q.isEmpty) return _allItems;
    return _allItems
        .where((item) => item.label.toLowerCase().contains(q))
        .toList();
  }

  List<Object> get _displayList {
    final filtered = _filtered;
    if (filtered.isEmpty) return [];

    final grouped = <String, List<SearchItem>>{};
    for (final item in filtered) {
      grouped.putIfAbsent(item.category, () => []).add(item);
    }

    final result = <Object>[];
    for (final cat in ['PROJECTS', 'ORGANIZATIONS', 'NAVIGATION']) {
      final items = grouped[cat];
      if (items == null || items.isEmpty) continue;
      result.add(cat);
      result.addAll(items);
    }
    return result;
  }

  List<SearchItem> get _selectable =>
      _displayList.whereType<SearchItem>().toList();

  void _onKey(KeyEvent event) {
    if (event is! KeyDownEvent) return;
    final items = _selectable;
    if (event.logicalKey == LogicalKeyboardKey.arrowDown) {
      setState(
          () => _selectedIndex = (_selectedIndex + 1).clamp(0, items.length - 1));
    } else if (event.logicalKey == LogicalKeyboardKey.arrowUp) {
      setState(
          () => _selectedIndex = (_selectedIndex - 1).clamp(0, items.length - 1));
    } else if (event.logicalKey == LogicalKeyboardKey.enter) {
      if (items.isNotEmpty && _selectedIndex < items.length) {
        items[_selectedIndex].action();
      }
    } else if (event.logicalKey == LogicalKeyboardKey.escape) {
      Navigator.of(context).pop();
    }
  }

  @override
  Widget build(BuildContext context) {
    final colors = consoleColors(context);
    final display = _displayList;
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
            constraints: const BoxConstraints(maxHeight: 480),
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
                  padding: const EdgeInsets.symmetric(horizontal: 16),
                  child: Row(
                    children: [
                      Icon(LucideIcons.search,
                          size: 16,
                          color: colors.textMuted),
                      const SizedBox(width: 12),
                      Expanded(
                        child: TextField(
                          controller: _ctrl,
                          autofocus: true,
                          style: TextStyle(
                              color: colors.textPrimary, fontSize: 14),
                          decoration: InputDecoration(
                            hintText: 'Search...',
                            hintStyle: TextStyle(
                                color: colors.textSubtle,
                                fontSize: 14),
                            border: InputBorder.none,
                            enabledBorder: InputBorder.none,
                            focusedBorder: InputBorder.none,
                            filled: false,
                            contentPadding:
                                const EdgeInsets.symmetric(vertical: 10),
                          ),
                          onSubmitted: (_) {
                            final items = _selectable;
                            if (items.isNotEmpty &&
                                _selectedIndex < items.length) {
                              items[_selectedIndex].action();
                            }
                          },
                        ),
                      ),
                      // Esc badge
                      const SearchHintBadge(label: 'Esc'),
                    ],
                  ),
                ),

                Container(height: 1, color: colors.border),

                // Results
                if (display.isEmpty)
                  Padding(
                    padding: const EdgeInsets.all(40),
                    child: Column(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        Icon(LucideIcons.search,
                            size: 28,
                          color: colors.textSubtle),
                        const SizedBox(height: 12),
                        Text(
                          'No results found',
                          style: TextStyle(
                              color: colors.textMuted,
                              fontSize: 13),
                        ),
                      ],
                    ),
                  )
                else
                  Flexible(
                    child: ListView.builder(
                      padding: const EdgeInsets.symmetric(vertical: 6),
                      shrinkWrap: true,
                      itemCount: display.length,
                      itemBuilder: (ctx, i) {
                        final entry = display[i];
                        if (entry is String) {
                          return Padding(
                            padding: const EdgeInsets.only(
                                left: 16, top: 12, bottom: 4, right: 16),
                            child: Text(
                              entry,
                              style: TextStyle(
                                color: colors.textMuted,
                                fontSize: 11,
                                fontWeight: FontWeight.w600,
                                letterSpacing: 1.0,
                              ),
                            ),
                          );
                        }

                        selectableCounter++;
                        final itemIndex = selectableCounter;
                        final item = entry as SearchItem;
                        final isSelected = itemIndex == _selectedIndex;

                        return SearchItemTile(
                          item: item,
                          isSelected: isSelected,
                          onTap: item.action,
                          onHover: () =>
                              setState(() => _selectedIndex = itemIndex),
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
                            color: colors.textSubtle,
                              fontSize: 11)),
                      const SizedBox(width: 16),
                      const SearchHintBadge(label: '↑↓'),
                      const SizedBox(width: 6),
                      Text('navigate',
                          style: TextStyle(
                            color: colors.textSubtle,
                              fontSize: 11)),
                      const Spacer(),
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
    return MouseRegion(
      onEnter: (_) => onHover(),
      cursor: SystemMouseCursors.click,
      child: GestureDetector(
        onTap: onTap,
        child: Container(
          margin: const EdgeInsets.symmetric(horizontal: 6, vertical: 1),
          padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
          decoration: BoxDecoration(
            color: isSelected
                ? colors.fillActive
                : Colors.transparent,
            borderRadius: BorderRadius.circular(8),
          ),
          child: Row(
            children: [
              Container(
                width: 28,
                height: 28,
                decoration: BoxDecoration(
                  color: item.isCreate
                      ? const Color(0xFF3472A4).withValues(alpha: 0.12)
                      : colors.fill,
                  borderRadius: BorderRadius.circular(6),
                ),
                child: Center(
                  child: Icon(
                    item.isCreate ? LucideIcons.plus : LucideIcons.arrowRight,
                    size: 13,
                    color: item.isCreate
                        ? const Color(0xFF3472A4)
                        : colors.textMuted,
                  ),
                ),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: Text(
                  item.label,
                  style: TextStyle(
                    color: isSelected ? colors.textPrimary : colors.textSecondary,
                    fontSize: 13,
                    fontWeight: isSelected ? FontWeight.w500 : FontWeight.w400,
                  ),
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
                  color: consoleColors(context).textSubtle, fontSize: 11)),
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
