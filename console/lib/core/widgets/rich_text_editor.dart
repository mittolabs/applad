import 'package:flutter/material.dart';
import 'package:flutter_markdown/flutter_markdown.dart';
import 'package:lucide_icons/lucide_icons.dart';
import '../theme/console_colors.dart';

/// A self-contained markdown rich text editor with Write / Preview tabs.
///
/// Usage:
/// ```dart
/// final ctrl = TextEditingController();
/// RichTextEditor(controller: ctrl, hintText: 'Write something…')
/// ```
class RichTextEditor extends StatefulWidget {
  final TextEditingController controller;
  final String hintText;
  /// Minimum visible lines in edit mode (default 12).
  final int minLines;

  const RichTextEditor({
    super.key,
    required this.controller,
    this.hintText = 'Write in Markdown…',
    this.minLines = 12,
  });

  @override
  State<RichTextEditor> createState() => _RichTextEditorState();
}

class _RichTextEditorState extends State<RichTextEditor>
    with SingleTickerProviderStateMixin {
  late TabController _tabCtrl;
  final _focusNode = FocusNode();
  bool _preview = false;

  @override
  void initState() {
    super.initState();
    _tabCtrl = TabController(length: 2, vsync: this);
    _tabCtrl.addListener(() {
      if (mounted) setState(() => _preview = _tabCtrl.index == 1);
    });
    // Rebuild on text changes so preview stays in sync.
    widget.controller.addListener(_onTextChange);
  }

  @override
  void dispose() {
    _tabCtrl.dispose();
    _focusNode.dispose();
    widget.controller.removeListener(_onTextChange);
    super.dispose();
  }

  void _onTextChange() {
    if (_preview && mounted) setState(() {});
  }

  // ── Markdown insertion helpers ─────────────────────────────────────────────

  void _wrap(String before, String after, {String placeholder = ''}) {
    final ctrl = widget.controller;
    final text = ctrl.text;
    final sel = ctrl.selection;
    final selected = sel.isValid && !sel.isCollapsed
        ? text.substring(sel.start, sel.end)
        : placeholder;
    final start = sel.isValid ? sel.start : text.length;
    final end = sel.isValid ? sel.end : text.length;
    final newText = text.replaceRange(start, end, '$before$selected$after');
    final newOffset = start + before.length + selected.length;
    ctrl.value = TextEditingValue(
      text: newText,
      selection: TextSelection.collapsed(offset: newOffset),
    );
    _focusNode.requestFocus();
  }

  void _linePrefix(String prefix) {
    final ctrl = widget.controller;
    final text = ctrl.text;
    final sel = ctrl.selection;
    final cursorPos = sel.isValid ? sel.start : text.length;
    final lineStart = text.lastIndexOf('\n', cursorPos - 1) + 1;
    final newText = text.replaceRange(lineStart, lineStart, '$prefix ');
    ctrl.value = TextEditingValue(
      text: newText,
      selection:
          TextSelection.collapsed(offset: cursorPos + prefix.length + 1),
    );
    _focusNode.requestFocus();
  }

  // ── Build ──────────────────────────────────────────────────────────────────

  @override
  Widget build(BuildContext context) {
    final cs = consoleColors(context);

    return Container(
      decoration: BoxDecoration(
        color: cs.background,
        borderRadius: BorderRadius.circular(10),
        border: Border.all(color: cs.border),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          // ── Tab bar + toolbar ──────────────────────────────────────────
          Container(
            decoration: BoxDecoration(
              color: cs.surface,
              borderRadius:
                  const BorderRadius.vertical(top: Radius.circular(10)),
              border: Border(bottom: BorderSide(color: cs.border)),
            ),
            child: Row(
              children: [
                _Tab('Write', 0, _tabCtrl, cs),
                _Tab('Preview', 1, _tabCtrl, cs),
                const SizedBox(width: 4),
                if (!_preview) ...[
                  _divider(cs),
                  ..._toolbar(cs),
                ],
              ],
            ),
          ),

          // ── Content ────────────────────────────────────────────────────
          AnimatedSwitcher(
            duration: const Duration(milliseconds: 120),
            child: _preview
                ? _PreviewPane(
                    key: const ValueKey('preview'),
                    text: widget.controller.text,
                    cs: cs,
                    minLines: widget.minLines,
                  )
                : _EditPane(
                    key: const ValueKey('edit'),
                    controller: widget.controller,
                    focusNode: _focusNode,
                    hintText: widget.hintText,
                    minLines: widget.minLines,
                    cs: cs,
                  ),
          ),
        ],
      ),
    );
  }

  Widget _divider(ConsoleColors cs) => Container(
        width: 1,
        height: 20,
        margin: const EdgeInsets.symmetric(horizontal: 4),
        color: cs.border,
      );

  List<Widget> _toolbar(ConsoleColors cs) {
    return [
      _Btn(LucideIcons.bold,           'Bold',           cs, () => _wrap('**', '**', placeholder: 'bold')),
      _Btn(LucideIcons.italic,         'Italic',         cs, () => _wrap('*', '*', placeholder: 'italic')),
      _Btn(LucideIcons.strikethrough,  'Strikethrough',  cs, () => _wrap('~~', '~~', placeholder: 'text')),
      _divider(cs),
      _Btn(LucideIcons.heading1,       'Heading 1',      cs, () => _linePrefix('#')),
      _Btn(LucideIcons.heading2,       'Heading 2',      cs, () => _linePrefix('##')),
      _Btn(LucideIcons.heading3,       'Heading 3',      cs, () => _linePrefix('###')),
      _divider(cs),
      _Btn(LucideIcons.list,           'Bullet list',    cs, () => _linePrefix('-')),
      _Btn(LucideIcons.listOrdered,    'Numbered list',  cs, () => _linePrefix('1.')),
      _Btn(LucideIcons.listTodo,       'Task list',      cs, () => _linePrefix('- [ ]')),
      _divider(cs),
      _Btn(LucideIcons.code,           'Inline code',    cs, () => _wrap('`', '`', placeholder: 'code')),
      _Btn(LucideIcons.terminalSquare, 'Code block',     cs, () => _wrap('\n```\n', '\n```\n', placeholder: 'code')),
      _divider(cs),
      _Btn(LucideIcons.quote,          'Blockquote',     cs, () => _linePrefix('>')),
      _Btn(LucideIcons.link,           'Link',           cs, () => _wrap('[', '](url)', placeholder: 'text')),
      _Btn(LucideIcons.image,          'Image',          cs, () => _wrap('![', '](url)', placeholder: 'alt')),
      _Btn(LucideIcons.minus,          'Horizontal rule',cs, () {
        final ctrl = widget.controller;
        final pos = ctrl.selection.isValid ? ctrl.selection.end : ctrl.text.length;
        final newText = ctrl.text.replaceRange(pos, pos, '\n\n---\n\n');
        ctrl.value = TextEditingValue(
          text: newText,
          selection: TextSelection.collapsed(offset: pos + 7),
        );
        _focusNode.requestFocus();
      }),
      const SizedBox(width: 4),
    ];
  }
}

// ── Sub-widgets ───────────────────────────────────────────────────────────────

class _Tab extends StatelessWidget {
  final String label;
  final int index;
  final TabController ctrl;
  final ConsoleColors cs;

  const _Tab(this.label, this.index, this.ctrl, this.cs);

  @override
  Widget build(BuildContext context) {
    const accent = Color(0xFF3472A4);
    return AnimatedBuilder(
      animation: ctrl,
      builder: (_, __) {
        final active = ctrl.index == index;
        return GestureDetector(
          onTap: () => ctrl.animateTo(index),
          child: Container(
            padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 10),
            decoration: BoxDecoration(
              border: Border(
                bottom: BorderSide(
                  color: active ? accent : Colors.transparent,
                  width: 2,
                ),
              ),
            ),
            child: Text(
              label,
              style: TextStyle(
                fontSize: 13,
                fontWeight: active ? FontWeight.w600 : FontWeight.w400,
                color: active ? cs.textPrimary : cs.textSubtle,
              ),
            ),
          ),
        );
      },
    );
  }
}

class _Btn extends StatelessWidget {
  final IconData icon;
  final String tooltip;
  final ConsoleColors cs;
  final VoidCallback onTap;

  const _Btn(this.icon, this.tooltip, this.cs, this.onTap);

  @override
  Widget build(BuildContext context) {
    return Tooltip(
      message: tooltip,
      waitDuration: const Duration(milliseconds: 500),
      child: InkWell(
        onTap: onTap,
        borderRadius: BorderRadius.circular(4),
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 5, vertical: 8),
          child: Icon(icon, size: 15, color: cs.textSubtle),
        ),
      ),
    );
  }
}

class _EditPane extends StatelessWidget {
  final TextEditingController controller;
  final FocusNode focusNode;
  final String hintText;
  final int minLines;
  final ConsoleColors cs;

  const _EditPane({
    super.key,
    required this.controller,
    required this.focusNode,
    required this.hintText,
    required this.minLines,
    required this.cs,
  });

  @override
  Widget build(BuildContext context) {
    return TextField(
      controller: controller,
      focusNode: focusNode,
      maxLines: null,
      minLines: minLines,
      style: TextStyle(
        color: cs.textPrimary,
        fontSize: 14,
        fontFamily: 'monospace',
        height: 1.65,
      ),
      decoration: InputDecoration(
        hintText: hintText,
        hintStyle: TextStyle(color: cs.textSubtle, fontSize: 13, height: 1.6),
        contentPadding: const EdgeInsets.all(16),
        border: InputBorder.none,
      ),
    );
  }
}

class _PreviewPane extends StatelessWidget {
  final String text;
  final int minLines;
  final ConsoleColors cs;

  const _PreviewPane({
    super.key,
    required this.text,
    required this.minLines,
    required this.cs,
  });

  @override
  Widget build(BuildContext context) {
    // Approximate minLines height so the pane doesn't collapse.
    final minHeight = minLines * 22.0;

    return Container(
      constraints: BoxConstraints(minHeight: minHeight),
      padding: const EdgeInsets.all(16),
      child: text.trim().isEmpty
          ? Text(
              'Nothing to preview yet.',
              style: TextStyle(color: cs.textSubtle, fontSize: 13),
            )
          : MarkdownBody(
              data: text,
              selectable: true,
              styleSheet: MarkdownStyleSheet(
                p: TextStyle(
                    color: cs.textPrimary, fontSize: 14, height: 1.65),
                h1: TextStyle(
                    color: cs.textPrimary,
                    fontSize: 22,
                    fontWeight: FontWeight.w700,
                    height: 1.4),
                h2: TextStyle(
                    color: cs.textPrimary,
                    fontSize: 18,
                    fontWeight: FontWeight.w700,
                    height: 1.4),
                h3: TextStyle(
                    color: cs.textPrimary,
                    fontSize: 15,
                    fontWeight: FontWeight.w600,
                    height: 1.4),
                code: TextStyle(
                  color: const Color(0xFF7DD3FC),
                  backgroundColor: cs.fillHover,
                  fontFamily: 'monospace',
                  fontSize: 13,
                ),
                codeblockDecoration: BoxDecoration(
                  color: cs.fillHover,
                  borderRadius: BorderRadius.circular(8),
                ),
                blockquote: TextStyle(
                    color: cs.textSecondary,
                    fontStyle: FontStyle.italic),
                blockquoteDecoration: BoxDecoration(
                  border: Border(
                    left: BorderSide(
                        color: cs.border.withValues(alpha: 0.8), width: 3),
                  ),
                ),
                blockquotePadding: const EdgeInsets.only(left: 12),
                listBullet:
                    TextStyle(color: cs.textPrimary),
                strong: TextStyle(
                    color: cs.textPrimary,
                    fontWeight: FontWeight.w700),
                em: TextStyle(
                    color: cs.textSecondary,
                    fontStyle: FontStyle.italic),
                a: const TextStyle(color: Color(0xFF3472A4)),
                tableHead: TextStyle(
                    color: cs.textPrimary,
                    fontWeight: FontWeight.w600),
                tableBody:
                    TextStyle(color: cs.textSecondary),
                tableBorder: TableBorder.all(color: cs.border),
                tableHeadAlign: TextAlign.left,
                horizontalRuleDecoration: BoxDecoration(
                  border:
                      Border(top: BorderSide(color: cs.border)),
                ),
                checkbox: TextStyle(color: cs.textPrimary),
              ),
            ),
    );
  }
}
