import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:lucide_icons/lucide_icons.dart';
import '../../core/api/client.dart';
import '../../core/theme/console_colors.dart';
import '../../core/utils/url_utils.dart';
import '../../core/widgets/app_data_table.dart';
import '../../core/widgets/app_dialog.dart';
import '../../core/widgets/id_text.dart';
import '../../core/widgets/page_tabs.dart';
import '../../core/widgets/app_error_state.dart';

// ── Constants ──────────────────────────────────────────────────────────────────
const _accent = Color(0xFF3472A4);

// ── Message model ─────────────────────────────────────────────────────────────
class _Msg {
  final String id;
  final String type; // email | sms | push
  final String subject;
  final String body;
  final String status; // processing | sent | failed | draft
  final String createdAt;
  final List<String> recipients;

  _Msg({
    required this.id,
    required this.type,
    required this.subject,
    required this.body,
    required this.status,
    required this.createdAt,
    required this.recipients,
  });

  factory _Msg.fromJson(Map<String, dynamic> j) => _Msg(
        id: j['id'] as String? ?? '',
        type: j['type'] as String? ?? '',
        subject: j['subject'] as String? ?? '',
        body: j['body'] as String? ?? '',
        status: j['status'] as String? ?? 'processing',
        createdAt: j['createdAt'] as String? ?? '',
        recipients: (j['recipients'] as List<dynamic>?)
                ?.map((e) => e.toString())
                .toList() ??
            [],
      );
}

// ── Providers ──────────────────────────────────────────────────────────────────
final _msgSearchProvider = StateProvider<String>((ref) => '');
final _msgPerPageProvider = StateProvider<int>((ref) => 12);
final _msgPageProvider = StateProvider<int>((ref) => 1);
final _msgRefreshProvider = StateProvider<int>((ref) => 0);
final _tmplRefreshProvider = StateProvider<int>((ref) => 0);

final _messagesApiProvider =
    FutureProvider.autoDispose<Map<String, dynamic>>((ref) async {
  final api = ref.read(apiClientProvider);
  ref.watch(_msgRefreshProvider);
  final search = ref.watch(_msgSearchProvider);
  final limit = ref.watch(_msgPerPageProvider);
  final page = ref.watch(_msgPageProvider);
  final offset = (page - 1) * limit;
  final params = <String, dynamic>{'limit': limit, 'offset': offset};
  if (search.isNotEmpty) params['search'] = search;
  final res = await api.get('/messaging/messages', params: params);
  return res.data as Map<String, dynamic>;
});

final _templatesApiProvider =
    FutureProvider.autoDispose<Map<String, dynamic>>((ref) async {
  final api = ref.read(apiClientProvider);
  ref.watch(_tmplRefreshProvider);
  final res = await api.get('/messaging/templates');
  return res.data as Map<String, dynamic>;
});

// ── Page ──────────────────────────────────────────────────────────────────────
class MessagingPage extends ConsumerStatefulWidget {
  const MessagingPage({super.key});

  @override
  ConsumerState<MessagingPage> createState() => _MessagingPageState();
}

class _MessagingPageState extends ConsumerState<MessagingPage> {
  static const _tabNames = ['messages', 'topics', 'templates', 'providers'];

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    final urlPage = pageFromQuery(context);
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted) return;
      if (ref.read(_msgPageProvider) != urlPage) {
        ref.read(_msgPageProvider.notifier).state = urlPage;
      }
    });
  }

  @override
  Widget build(BuildContext context) {
    final cs = consoleColors(context);
    final tabName = tabFromQuery(context, defaultTab: 'messages');
    final tab = _tabNames.indexOf(tabName).clamp(0, _tabNames.length - 1);

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
            Text(
              'Messaging',
              style: TextStyle(
                color: cs.textPrimary,
                fontSize: 22,
                fontWeight: FontWeight.w600,
              ),
            ),
            const SizedBox(height: 4),
            Text('Send email, SMS and push notifications to your users',
                style: TextStyle(color: cs.textSecondary, fontSize: 13)),
            const SizedBox(height: 20),
            PageTabs(
              tabs: const ['Messages', 'Topics', 'Templates', 'Providers'],
              selected: tab,
              onChanged: (i) => context.go(withQuery(context, {'tab': _tabNames[i], 'page': null})),
            ),
            const SizedBox(height: 20),
            Expanded(
              child: IndexedStack(
                index: tab,
                children: const [
                  _MessagesTab(),
                  _TopicsTab(),
                  _TemplatesTab(),
                  _ProvidersTab(),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}

// ── Messages Tab ──────────────────────────────────────────────────────────────
class _MessagesTab extends ConsumerStatefulWidget {
  const _MessagesTab();

  @override
  ConsumerState<_MessagesTab> createState() => _MessagesTabState();
}

class _MessagesTabState extends ConsumerState<_MessagesTab> {
  final _searchCtrl = TextEditingController();
  _Msg? _selected;

  @override
  void dispose() {
    _searchCtrl.dispose();
    super.dispose();
  }

  void _refresh() => ref.read(_msgRefreshProvider.notifier).state++;

  void _doSearch() {
    ref.read(_msgSearchProvider.notifier).state = _searchCtrl.text.trim();
    ref.read(_msgPageProvider.notifier).state = 1;
    _refresh();
  }

  @override
  Widget build(BuildContext context) {
    if (_selected != null) {
      return _MessageDetail(
        msg: _selected!,
        onBack: () => setState(() => _selected = null),
      );
    }

    final async = ref.watch(_messagesApiProvider);
    final perPage = ref.watch(_msgPerPageProvider);
    final currentPage = ref.watch(_msgPageProvider);

    return async.when(
      loading: () => const Center(child: CircularProgressIndicator(strokeWidth: 2)),
      error: (e, _) => AppErrorState(error: e),
      data: (data) {
        final rows = List<Map<String, dynamic>>.from(data['messages'] ?? []);
        final total = data['total'] as int? ?? rows.length;
        return AppDataTable(
          columns: const [
            AppTableColumn(key: r'$id',       label: 'Message ID', flex: 3),
            AppTableColumn(key: 'type',        label: 'Type',       flex: 2, sortable: false),
            AppTableColumn(key: 'status',      label: 'Status',     flex: 2, sortable: false),
            AppTableColumn(key: 'createdAt',   label: 'Created',    flex: 2),
          ],
          rows: rows,
          getCellValue: (row, key) => switch (key) {
            r'$id'     => row['id'] as String? ?? row[r'$id'] as String? ?? '',
            'type'     => _typeName(row['type'] as String? ?? ''),
            'status'   => row['status'] as String? ?? '',
            'createdAt'=> row['createdAt'] as String? ?? '',
            _          => '',
          },
          cellBuilder: (row, key) {
            if (key == 'status') return _StatusBadge(status: row['status'] as String? ?? '');
            return null;
          },
          getRowIcon: (row) => _typeIcon(row['type'] as String? ?? ''),
          onRowTap: (row) {
            final msg = _Msg.fromJson(row);
            setState(() => _selected = msg);
          },
          createLabel: 'Create message',
          createWidget: _CreateMessageButton(onCreated: _refresh),
          total: total,
          perPage: perPage,
          currentPage: currentPage,
          onPrev: () {
            final p = currentPage - 1;
            ref.read(_msgPageProvider.notifier).state = p;
            context.go(withQuery(context, {'page': '$p'}));
          },
          onNext: () {
            final p = currentPage + 1;
            ref.read(_msgPageProvider.notifier).state = p;
            context.go(withQuery(context, {'page': '$p'}));
          },
          onPerPageChanged: (v) {
            ref.read(_msgPerPageProvider.notifier).state = v;
            ref.read(_msgPageProvider.notifier).state = 1;
          },
          itemLabel: 'messages',
          searchController: _searchCtrl,
          onSearch: _doSearch,
          searchHint: 'Search by type, status, or ID',
          emptyIcon: LucideIcons.messageSquare,
          emptyTitle: 'No messages',
          emptySubtitle: 'Create a message to send email, SMS, or push notifications.',
          filters: const [
            AppTableFilter(key: 'type',   label: 'Type',   options: ['email', 'sms', 'push']),
            AppTableFilter(key: 'status', label: 'Status', options: ['processing', 'sent', 'failed', 'draft']),
          ],
        );
      },
    );
  }

  IconData _typeIcon(String t) {
    if (t == 'email') return LucideIcons.mail;
    if (t == 'sms') return LucideIcons.messageSquare;
    return LucideIcons.bell;
  }

  String _typeName(String t) {
    if (t == 'email') return 'Email';
    if (t == 'sms') return 'SMS';
    return 'Push';
  }

}

// ── Status badge ──────────────────────────────────────────────────────────────
class _StatusBadge extends StatelessWidget {
  final String status;
  const _StatusBadge({required this.status});

  @override
  Widget build(BuildContext context) {
    final Color bg;
    final Color fg;
    switch (status) {
      case 'sent':
        bg = const Color(0xFF16423E);
        fg = const Color(0xFF34D399);
        break;
      case 'failed':
        bg = const Color(0xFF3B1A1A);
        fg = const Color(0xFFEF4444);
        break;
      case 'draft':
        bg = const Color(0xFF252525);
        fg = Colors.white54;
        break;
      default: // processing
        bg = const Color(0xFF2D3B1E);
        fg = const Color(0xFFF59E0B);
        break;
    }
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
      decoration: BoxDecoration(
          color: bg, borderRadius: BorderRadius.circular(4)),
      child: Text(
        status,
        style: TextStyle(
            color: fg, fontSize: 11, fontWeight: FontWeight.w500),
      ),
    );
  }
}

// ── Create message button (popup menu) ────────────────────────────────────────
class _CreateMessageButton extends ConsumerWidget {
  final VoidCallback onCreated;
  const _CreateMessageButton({required this.onCreated});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final cs = consoleColors(context);
    return PopupMenuButton<String>(
      offset: const Offset(0, 40),
      color: cs.popupSurface,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(8),
        side: BorderSide(color: cs.border),
      ),
      itemBuilder: (_) => [
        _menuItem('email', LucideIcons.mail, 'Email'),
        _menuItem('sms', LucideIcons.messageSquare, 'SMS'),
        _menuItem('push', LucideIcons.bell, 'Push notification'),
      ],
      onSelected: (type) => _openDialog(context, ref, type),
      child: Container(
        padding:
            const EdgeInsets.symmetric(horizontal: 14, vertical: 8),
        decoration: BoxDecoration(
          color: _accent,
          borderRadius: BorderRadius.circular(8),
        ),
        child: const Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(LucideIcons.plus, size: 14, color: Colors.white),
            SizedBox(width: 6),
            Text('Create message',
                style: TextStyle(
                    color: Colors.white,
                    fontSize: 12)),
          ],
        ),
      ),
    );
  }

  PopupMenuItem<String> _menuItem(
      String value, IconData icon, String label) {
    return PopupMenuItem(
      value: value,
      height: 36,
      child: Row(
        children: [
          Icon(icon, size: 14, color: Colors.white54),
          const SizedBox(width: 10),
          Text(label,
              style:
                  const TextStyle(color: Colors.white, fontSize: 13)),
        ],
      ),
    );
  }

  void _openDialog(
      BuildContext context, WidgetRef ref, String type) {
    Navigator.of(context).push(
      PageRouteBuilder(
        opaque: false,
        barrierColor: Colors.black54,
        barrierDismissible: true,
        pageBuilder: (_, __, ___) => _CreateMsgDialog(type: type),
        transitionsBuilder: (_, anim, __, child) =>
            FadeTransition(opacity: anim, child: child),
        transitionDuration: const Duration(milliseconds: 150),
      ),
    );
  }
}

// ── Create message dialog ─────────────────────────────────────────────────────
class _CreateMsgDialog extends ConsumerStatefulWidget {
  final String type;
  const _CreateMsgDialog({required this.type});

  @override
  ConsumerState<_CreateMsgDialog> createState() =>
      _CreateMsgDialogState();
}

class _CreateMsgDialogState extends ConsumerState<_CreateMsgDialog> {
  final _subjectCtrl = TextEditingController();
  final _msgCtrl = TextEditingController();
  final _toCtrl = TextEditingController();
  final _titleCtrl = TextEditingController();
  bool _htmlMode = false;
  bool _sending = false;
  bool _saving = false;
  String _schedule = 'Now';

  @override
  void initState() {
    super.initState();
    _msgCtrl.addListener(() => setState(() {}));
    _titleCtrl.addListener(() => setState(() {}));
  }

  @override
  void dispose() {
    _subjectCtrl.dispose();
    _msgCtrl.dispose();
    _toCtrl.dispose();
    _titleCtrl.dispose();
    super.dispose();
  }

  String get _title {
    if (widget.type == 'email') return 'Create email message';
    if (widget.type == 'sms') return 'Create SMS message';
    return 'Create push message';
  }

  @override
  Widget build(BuildContext context) {
    final cs = consoleColors(context);
    return Scaffold(
      backgroundColor: cs.background,
      body: Padding(
        padding: EdgeInsets.symmetric(
          horizontal: pageHPad(context),
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            _dialogHeader(),
            Expanded(
              child: SingleChildScrollView(
                padding: const EdgeInsets.only(top: 24, bottom: 40),
                child: widget.type == 'email'
                    ? _emailBody()
                    : widget.type == 'sms'
                        ? _smsBody()
                        : _pushBody(),
              ),
            ),
            _dialogActions(),
          ],
        ),
      ),
    );
  }

  Widget _dialogHeader() {
    final cs = consoleColors(context);
    return Padding(
      padding: const EdgeInsets.only(top: 32, bottom: 4),
      child: Row(
        children: [
          GestureDetector(
            onTap: () => Navigator.of(context).pop(),
            child: MouseRegion(
              cursor: SystemMouseCursors.click,
              child: Icon(LucideIcons.arrowLeft,
                  size: 20, color: cs.textMuted),
            ),
          ),
          const SizedBox(width: 12),
          Text(
            _title,
            style: TextStyle(
              color: cs.textPrimary,
              fontSize: 22,
              fontWeight: FontWeight.w600,
            ),
          ),
        ],
      ),
    );
  }

  Widget _dialogActions() {
    final cs = consoleColors(context);
    return Container(
      padding: const EdgeInsets.symmetric(vertical: 16),
      decoration: BoxDecoration(
        border: Border(top: BorderSide(color: cs.border)),
      ),
      child: Row(
        mainAxisAlignment: MainAxisAlignment.end,
        children: [
          TextButton(
            onPressed: () => Navigator.of(context).pop(),
            style: TextButton.styleFrom(
              foregroundColor: cs.textMuted,
              padding:
                  const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
            ),
            child: const Text('Cancel',
                style: TextStyle(fontSize: 13)),
          ),
          const SizedBox(width: 8),
          OutlinedButton(
            onPressed: _saving ? null : _saveDraft,
            style: OutlinedButton.styleFrom(
              foregroundColor: cs.textSecondary,
              side: BorderSide(color: cs.border),
              padding:
                  const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
              shape: RoundedRectangleBorder(
                  borderRadius: BorderRadius.circular(8)),
            ),
            child: _saving
                ? const SizedBox(
                    width: 14,
                    height: 14,
                    child: CircularProgressIndicator(
                        strokeWidth: 2, color: Colors.white54))
                : const Text('Save as draft',
                    style: TextStyle(fontSize: 13)),
          ),
          const SizedBox(width: 8),
          FilledButton(
            onPressed: _sending ? null : _create,
            style: FilledButton.styleFrom(
              backgroundColor: _accent,
              padding:
                  const EdgeInsets.symmetric(horizontal: 20, vertical: 10),
              shape: RoundedRectangleBorder(
                  borderRadius: BorderRadius.circular(8)),
            ),
            child: _sending
                ? const SizedBox(
                    width: 14,
                    height: 14,
                    child: CircularProgressIndicator(
                        strokeWidth: 2, color: Colors.white))
                : const Text('Create',
                    style: TextStyle(fontSize: 13)),
          ),
        ],
      ),
    );
  }

  // ── Email form ──
  Widget _emailBody() {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        _sectionLabel('Message'),
        _fieldLabel('Subject'),
        _inputField(_subjectCtrl,
            hint: 'Enter subject', autofocus: true),
        const SizedBox(height: 16),
        _fieldLabel('Message'),
        _inputField(_msgCtrl, hint: 'Type here...', maxLines: 5),
        const SizedBox(height: 10),
        _htmlToggle(),
        const SizedBox(height: 24),
        _divider(),
        _sectionLabel('Targets'),
        _fieldLabel('To'),
        _inputField(_toCtrl,
            hint: 'user@example.com, other@example.com'),
        const SizedBox(height: 24),
        _divider(),
        _sectionLabel('Settings'),
        _fieldLabel('Schedule'),
        _scheduleDropdown(),
        const SizedBox(height: 8),
        _scheduleHint(),
      ],
    );
  }

  // ── SMS form ──
  Widget _smsBody() {
    final cs = consoleColors(context);
    return Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Expanded(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              _sectionLabel('Message'),
              _fieldLabel('Message'),
              _inputField(_msgCtrl, hint: 'Type here...', maxLines: 5),
              const SizedBox(height: 6),
              Align(
                alignment: Alignment.centerRight,
                child: Text(
                  '${_msgCtrl.text.length}/900',
                  style: TextStyle(
                      color: cs.textSubtle,
                      fontSize: 11),
                ),
              ),
              const SizedBox(height: 24),
              _divider(),
              _sectionLabel('Targets'),
              _fieldLabel('To'),
              _inputField(_toCtrl, hint: '+1234567890'),
              const SizedBox(height: 24),
              _divider(),
              _sectionLabel('Settings'),
              _fieldLabel('Schedule'),
              _scheduleDropdown(),
              const SizedBox(height: 8),
              _scheduleHint(),
            ],
          ),
        ),
        const SizedBox(width: 32),
        _SmsPhonePreview(message: _msgCtrl.text),
      ],
    );
  }

  // ── Push form ──
  Widget _pushBody() {
    final cs = consoleColors(context);
    return Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Expanded(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              _sectionLabel('Message'),
              _fieldLabel('Title'),
              _inputField(_titleCtrl,
                  hint: 'Enter title', autofocus: true),
              const SizedBox(height: 16),
              _fieldLabel('Message'),
              _inputField(_msgCtrl, hint: 'Type here...', maxLines: 4),
              const SizedBox(height: 6),
              Align(
                alignment: Alignment.centerRight,
                child: Text(
                  '${_msgCtrl.text.length}/1000',
                  style: TextStyle(
                      color: cs.textSubtle,
                      fontSize: 11),
                ),
              ),
              const SizedBox(height: 16),
              _fieldLabel('Media', optional: true),
              _mediaUpload(),
              const SizedBox(height: 24),
              _divider(),
              _sectionLabel('Targets'),
              _fieldLabel('FCM Token'),
              _inputField(_toCtrl, hint: 'Enter device token'),
              const SizedBox(height: 24),
              _divider(),
              _sectionLabel('Settings'),
              _fieldLabel('Schedule'),
              _scheduleDropdown(),
              const SizedBox(height: 8),
              _scheduleHint(),
            ],
          ),
        ),
        const SizedBox(width: 32),
        _PushPhonePreview(
          title: _titleCtrl.text,
          message: _msgCtrl.text,
        ),
      ],
    );
  }

  // ── Helpers ──
  Widget _sectionLabel(String label) {
    final cs = consoleColors(context);
    return Padding(
      padding: const EdgeInsets.only(bottom: 14),
      child: Text(
        label,
        style: TextStyle(
          color: cs.textMuted,
          fontSize: 11,
          fontWeight: FontWeight.w600,
          letterSpacing: 0.6,
        ),
      ),
    );
  }

  Widget _fieldLabel(String label, {bool optional = false}) {
    final cs = consoleColors(context);
    return Padding(
      padding: const EdgeInsets.only(bottom: 6),
      child: Row(
        children: [
          Text(label,
              style: TextStyle(
                  color: cs.textSecondary, fontSize: 13)),
          if (optional) ...[
            const SizedBox(width: 4),
            Text('optional',
                style: TextStyle(
                    color: cs.textSubtle,
                    fontSize: 11)),
          ],
        ],
      ),
    );
  }

  Widget _inputField(
    TextEditingController ctrl, {
    String hint = '',
    int maxLines = 1,
    bool autofocus = false,
  }) {
    final cs = consoleColors(context);
    final multi = maxLines > 1;
    return TextField(
      controller: ctrl,
      autofocus: autofocus,
      maxLines: multi ? null : 1,
      minLines: multi ? maxLines : 1,
      style: TextStyle(color: cs.textPrimary, fontSize: 13),
      decoration: InputDecoration(
        hintText: hint,
        hintStyle: TextStyle(
            color: cs.textSubtle, fontSize: 13),
        filled: true,
        fillColor: cs.fieldFill,
        isDense: true,
        contentPadding: multi
            ? const EdgeInsets.all(12)
            : const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
        border: OutlineInputBorder(
          borderRadius: BorderRadius.circular(8),
          borderSide: BorderSide(color: cs.fieldBorder),
        ),
        enabledBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(8),
          borderSide: BorderSide(color: cs.fieldBorder),
        ),
        focusedBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(8),
          borderSide: const BorderSide(color: _accent),
        ),
      ),
    );
  }

  Widget _htmlToggle() {
    final cs = consoleColors(context);
    return Row(
      children: [
        Switch(
          value: _htmlMode,
          onChanged: (v) => setState(() => _htmlMode = v),
          activeThumbColor: _accent,
          materialTapTargetSize: MaterialTapTargetSize.shrinkWrap,
        ),
        const SizedBox(width: 10),
        Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text('HTML mode',
                style: TextStyle(color: cs.textSecondary, fontSize: 13)),
            const SizedBox(height: 2),
            Text(
              'Enable the HTML mode if your message contains HTML tags.',
              style: TextStyle(color: cs.textSubtle, fontSize: 11),
            ),
          ],
        ),
      ],
    );
  }

  Widget _mediaUpload() {
    final cs = consoleColors(context);
    return Container(
      height: 90,
      width: double.infinity,
      decoration: BoxDecoration(
        color: cs.fieldFill,
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: cs.fieldBorder),
      ),
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          Icon(LucideIcons.upload,
              size: 20,
              color: cs.textSubtle),
          const SizedBox(height: 6),
          Text('Select a file to upload',
              style: TextStyle(color: cs.textMuted, fontSize: 13)),
          const SizedBox(height: 4),
          Row(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              Text('Max file size: 1MB',
                  style: TextStyle(color: cs.textSubtle, fontSize: 11)),
              const SizedBox(width: 8),
              MouseRegion(
                cursor: SystemMouseCursors.click,
                child: Text(
                  'Browse',
                  style: TextStyle(
                    color: cs.textMuted,
                    fontSize: 11,
                    decoration: TextDecoration.underline,
                  ),
                ),
              ),
            ],
          ),
        ],
      ),
    );
  }

  Widget _scheduleDropdown() {
    final cs = consoleColors(context);
    return Container(
      decoration: BoxDecoration(
        color: cs.fieldFill,
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: cs.fieldBorder),
      ),
      padding:
          const EdgeInsets.symmetric(horizontal: 12, vertical: 0),
      child: DropdownButtonHideUnderline(
        child: DropdownButton<String>(
          value: _schedule,
          isExpanded: true,
          isDense: true,
          dropdownColor: cs.popupSurface,
          style: TextStyle(color: cs.textPrimary, fontSize: 13),
          icon: Icon(Icons.keyboard_arrow_down,
              color: cs.textSubtle, size: 18),
          items: const [
            DropdownMenuItem(value: 'Now', child: Text('Now')),
            DropdownMenuItem(
                value: 'Schedule', child: Text('Schedule')),
          ],
          onChanged: (v) =>
              setState(() => _schedule = v!),
        ),
      ),
    );
  }

  Widget _scheduleHint() {
    final cs = consoleColors(context);
    return Row(
      children: [
        Icon(Icons.info_outline,
            size: 12,
            color: cs.textSubtle),
        const SizedBox(width: 5),
        Text(
          'The message will be sent immediately',
          style: TextStyle(color: cs.textSubtle, fontSize: 12),
        ),
      ],
    );
  }

  Widget _divider() {
    final cs = consoleColors(context);
    return Column(
      children: [
        Divider(color: cs.border, height: 1),
        const SizedBox(height: 20),
      ],
    );
  }

  // ── Actions ──
  Future<void> _saveDraft() async {
    setState(() => _saving = true);
    try {
      await _send(draft: true);
    } finally {
      setState(() => _saving = false);
    }
  }

  Future<void> _create() async {
    setState(() => _sending = true);
    try {
      await _send(draft: false);
    } finally {
      setState(() => _sending = false);
    }
  }

  Future<void> _send({required bool draft}) async {
    final api = ref.read(apiClientProvider);
    try {
      if (widget.type == 'email') {
        final to = _toCtrl.text
            .split(',')
            .map((e) => e.trim())
            .where((e) => e.isNotEmpty)
            .toList();
        await api.post('/messaging/messages/email', data: {
          'to': to,
          'subject': _subjectCtrl.text,
          'html': _msgCtrl.text,
          'draft': draft,
        });
      } else if (widget.type == 'sms') {
        await api.post('/messaging/messages/sms', data: {
          'to': _toCtrl.text.trim(),
          'body': _msgCtrl.text,
          'draft': draft,
        });
      } else {
        await api.post('/messaging/messages/push', data: {
          'token': _toCtrl.text.trim(),
          'title': _titleCtrl.text,
          'body': _msgCtrl.text,
          'draft': draft,
        });
      }
    } catch (_) {
      // still close — backend returns 201 even if send fails asynchronously
    }
    ref.read(_msgRefreshProvider.notifier).state++;
    if (mounted) Navigator.of(context).pop();
  }
}

// ── SMS phone preview ─────────────────────────────────────────────────────────
class _SmsPhonePreview extends StatelessWidget {
  final String message;
  const _SmsPhonePreview({required this.message});

  @override
  Widget build(BuildContext context) {
    return Container(
      width: 155,
      height: 290,
      decoration: BoxDecoration(
        color: const Color(0xFF111113),
        borderRadius: BorderRadius.circular(26),
        border:
            Border.all(color: Colors.white.withValues(alpha: 0.15), width: 2),
      ),
      child: ClipRRect(
        borderRadius: BorderRadius.circular(24),
        child: Column(
          children: [
            _statusBar(),
            Expanded(
              child: Column(
                children: [
                  const SizedBox(height: 8),
                  _contactAvatar(),
                  const SizedBox(height: 4),
                  const Text('Today 4:37 PM',
                      style: TextStyle(
                          color: Colors.white54,
                          fontSize: 9)),
                  const SizedBox(height: 8),
                  if (message.isNotEmpty)
                    Padding(
                      padding:
                          const EdgeInsets.symmetric(horizontal: 10),
                      child: Align(
                        alignment: Alignment.centerRight,
                        child: Container(
                          padding: const EdgeInsets.all(8),
                          decoration: BoxDecoration(
                            color: const Color(0xFF2196F3),
                            borderRadius:
                                BorderRadius.circular(12),
                          ),
                          child: Text(message,
                              style: const TextStyle(
                                  color: Colors.white,
                                  fontSize: 9)),
                        ),
                      ),
                    )
                  else
                    Padding(
                      padding:
                          const EdgeInsets.symmetric(horizontal: 16),
                      child: Text(
                        'Enter your message in the input field on the left to see it here',
                        style: TextStyle(
                            color: Colors.white.withValues(alpha: 0.3),
                            fontSize: 9),
                        textAlign: TextAlign.center,
                      ),
                    ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _statusBar() {
    return Container(
      height: 28,
      padding: const EdgeInsets.symmetric(horizontal: 14),
      child: Row(
        children: [
          Text('9:41',
              style: TextStyle(
                  color: Colors.white.withValues(alpha: 0.8),
                  fontSize: 10,
                  fontWeight: FontWeight.w600)),
          const Spacer(),
          Icon(Icons.signal_cellular_alt,
              size: 10,
              color: Colors.white.withValues(alpha: 0.7)),
          const SizedBox(width: 2),
          Icon(Icons.wifi,
              size: 10,
              color: Colors.white.withValues(alpha: 0.7)),
          const SizedBox(width: 2),
          Icon(Icons.battery_full,
              size: 10,
              color: Colors.white.withValues(alpha: 0.7)),
        ],
      ),
    );
  }

  Widget _contactAvatar() {
    return Container(
      width: 36,
      height: 36,
      decoration: BoxDecoration(
        color: Colors.white.withValues(alpha: 0.1),
        shape: BoxShape.circle,
      ),
    );
  }
}

// ── Push phone preview ────────────────────────────────────────────────────────
class _PushPhonePreview extends StatelessWidget {
  final String title;
  final String message;
  const _PushPhonePreview(
      {required this.title, required this.message});

  @override
  Widget build(BuildContext context) {
    final hasContent = title.isNotEmpty || message.isNotEmpty;
    return Container(
      width: 155,
      height: 320,
      decoration: BoxDecoration(
        color: const Color(0xFF111113),
        borderRadius: BorderRadius.circular(26),
        border:
            Border.all(color: Colors.white.withValues(alpha: 0.15), width: 2),
      ),
      child: ClipRRect(
        borderRadius: BorderRadius.circular(24),
        child: Column(
          children: [
            _statusBar(),
            Expanded(
              child: Center(
                child: hasContent
                    ? Padding(
                        padding:
                            const EdgeInsets.symmetric(horizontal: 8),
                        child: Container(
                          padding: const EdgeInsets.all(10),
                          decoration: BoxDecoration(
                            color:
                                Colors.white.withValues(alpha: 0.1),
                            borderRadius:
                                BorderRadius.circular(12),
                          ),
                          child: Column(
                            mainAxisSize: MainAxisSize.min,
                            crossAxisAlignment:
                                CrossAxisAlignment.start,
                            children: [
                              Row(
                                children: [
                                  Container(
                                    width: 14,
                                    height: 14,
                                    decoration: const BoxDecoration(
                                      color: _accent,
                                      shape: BoxShape.circle,
                                    ),
                                  ),
                                  const SizedBox(width: 4),
                                  Text(
                                    'App',
                                    style: TextStyle(
                                        color: Colors.white
                                            .withValues(alpha: 0.5),
                                        fontSize: 9),
                                  ),
                                  const Spacer(),
                                  Text('now',
                                      style: TextStyle(
                                          color: Colors.white
                                              .withValues(alpha: 0.4),
                                          fontSize: 9)),
                                ],
                              ),
                              if (title.isNotEmpty) ...[
                                const SizedBox(height: 4),
                                Text(title,
                                    style: const TextStyle(
                                        color: Colors.white,
                                        fontSize: 10,
                                        fontWeight:
                                            FontWeight.w600)),
                              ],
                              if (message.isNotEmpty) ...[
                                const SizedBox(height: 2),
                                Text(message,
                                    style: TextStyle(
                                        color: Colors.white
                                            .withValues(alpha: 0.7),
                                        fontSize: 9),
                                    maxLines: 3,
                                    overflow:
                                        TextOverflow.ellipsis),
                              ],
                            ],
                          ),
                        ),
                      )
                    : Padding(
                        padding:
                            const EdgeInsets.symmetric(horizontal: 14),
                        child: Text(
                          'Enter your message in the input field on the left to see it here',
                          style: TextStyle(
                              color: Colors.white.withValues(alpha: 0.3),
                              fontSize: 9),
                          textAlign: TextAlign.center,
                        ),
                      ),
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _statusBar() {
    return Container(
      height: 28,
      padding: const EdgeInsets.symmetric(horizontal: 14),
      child: Row(
        children: [
          Text('9:41',
              style: TextStyle(
                  color: Colors.white.withValues(alpha: 0.8),
                  fontSize: 10,
                  fontWeight: FontWeight.w600)),
          const Spacer(),
          Icon(Icons.signal_cellular_alt,
              size: 10,
              color: Colors.white.withValues(alpha: 0.7)),
          const SizedBox(width: 2),
          Icon(Icons.wifi,
              size: 10,
              color: Colors.white.withValues(alpha: 0.7)),
          const SizedBox(width: 2),
          Icon(Icons.battery_full,
              size: 10,
              color: Colors.white.withValues(alpha: 0.7)),
        ],
      ),
    );
  }
}

// ── Message detail ────────────────────────────────────────────────────────────
class _MessageDetail extends StatelessWidget {
  final _Msg msg;
  final VoidCallback onBack;

  const _MessageDetail({required this.msg, required this.onBack});

  @override
  Widget build(BuildContext context) {
    final cs = consoleColors(context);
    return Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          // Title row
          Row(
            children: [
              IconButton(
                icon: Icon(Icons.arrow_back,
                    size: 18, color: cs.textMuted),
                onPressed: onBack,
                padding: EdgeInsets.zero,
                constraints: const BoxConstraints(),
              ),
              const SizedBox(width: 8),
              Text(
                msg.subject.isNotEmpty ? msg.subject : _typeName(),
                style: TextStyle(
                    color: cs.textPrimary,
                    fontSize: 18,
                    fontWeight: FontWeight.w600),
              ),
              const SizedBox(width: 12),
              Container(
                padding: const EdgeInsets.symmetric(
                    horizontal: 8, vertical: 3),
                decoration: BoxDecoration(
                  color: cs.fill,
                  borderRadius: BorderRadius.circular(4),
                ),
                child: Row(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    Icon(LucideIcons.tag,
                        size: 10,
                        color: cs.textSubtle),
                    const SizedBox(width: 4),
                    IdText(id: msg.id, fontSize: 11),
                  ],
                ),
              ),
            ],
          ),
          const SizedBox(height: 24),
          Expanded(
            child: SingleChildScrollView(
              child: Column(
                children: [
                  // Type header card
                  _card(context,
                    child: Row(
                      children: [
                        Icon(_typeIconData(),
                            size: 16, color: cs.textMuted),
                        const SizedBox(width: 8),
                        Text(_typeName(),
                            style: TextStyle(
                                color: cs.textPrimary,
                                fontSize: 14,
                                fontWeight: FontWeight.w500)),
                        const Spacer(),
                        Text(
                          'Created: ${_fmtDate(msg.createdAt)}',
                          style: TextStyle(
                              color: cs.textSubtle,
                              fontSize: 12),
                        ),
                        const SizedBox(width: 16),
                        _StatusBadge(status: msg.status),
                      ],
                    ),
                  ),
                  const SizedBox(height: 12),
                  // Message card
                  _card(context,
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text('Message',
                            style: TextStyle(
                                color: cs.textPrimary,
                                fontSize: 14,
                                fontWeight: FontWeight.w500)),
                        Divider(color: cs.border),
                        if (msg.subject.isNotEmpty) ...[
                          _detailRow(context, 'Subject', msg.subject),
                          const SizedBox(height: 12),
                        ],
                        _detailRow(context,
                          msg.type == 'push' ? 'Body' : 'Message',
                          msg.body.isEmpty ? '-' : msg.body,
                        ),
                      ],
                    ),
                  ),
                  const SizedBox(height: 12),
                  // Targets card
                  _card(context,
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text('Targets',
                            style: TextStyle(
                                color: cs.textPrimary,
                                fontSize: 14,
                                fontWeight: FontWeight.w500)),
                        Divider(color: cs.border),
                        if (msg.recipients.isEmpty)
                          Padding(
                            padding:
                                const EdgeInsets.symmetric(vertical: 8),
                            child: Text(
                              'No targets selected',
                              style: TextStyle(
                                  color: cs.textSubtle,
                                  fontSize: 13),
                            ),
                          )
                        else
                          ...msg.recipients.map(
                            (r) => Padding(
                              padding:
                                  const EdgeInsets.symmetric(vertical: 4),
                              child: Row(
                                children: [
                                  Icon(_typeIconData(),
                                      size: 13,
                                      color: cs.textMuted),
                                  const SizedBox(width: 8),
                                  Text(r,
                                      style: TextStyle(
                                          color: cs.textSecondary,
                                          fontSize: 13)),
                                ],
                              ),
                            ),
                          ),
                      ],
                    ),
                  ),
                ],
              ),
            ),
          ),
        ],
    );
  }

  Widget _card(BuildContext context, {required Widget child}) {
    final cs = consoleColors(context);
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: cs.surface,
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: cs.border),
      ),
      child: child,
    );
  }

  Widget _detailRow(BuildContext context, String label, String value) {
    final cs = consoleColors(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(label,
            style: TextStyle(
                color: cs.textSubtle,
                fontSize: 11,
                fontWeight: FontWeight.w500)),
        const SizedBox(height: 4),
        Container(
          width: double.infinity,
          padding:
              const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
          decoration: BoxDecoration(
            color: cs.fill,
            borderRadius: BorderRadius.circular(6),
            border: Border.all(color: cs.border),
          ),
          child: Text(value,
              style: TextStyle(
                  color: cs.textSecondary, fontSize: 13)),
        ),
      ],
    );
  }

  IconData _typeIconData() {
    if (msg.type == 'email') return LucideIcons.mail;
    if (msg.type == 'sms') return LucideIcons.messageSquare;
    return LucideIcons.bell;
  }

  String _typeName() {
    if (msg.type == 'email') return 'Email';
    if (msg.type == 'sms') return 'SMS';
    return 'Push';
  }

  String _fmtDate(String iso) {
    try {
      final dt = DateTime.parse(iso).toLocal();
      const months = [
        'Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun',
        'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'
      ];
      return '${dt.day} ${months[dt.month - 1]} ${dt.year}, '
          '${dt.hour.toString().padLeft(2, '0')}:${dt.minute.toString().padLeft(2, '0')}';
    } catch (_) {
      return iso;
    }
  }
}

// ── Topics Tab ────────────────────────────────────────────────────────────────
class _TopicsTab extends ConsumerStatefulWidget {
  const _TopicsTab();

  @override
  ConsumerState<_TopicsTab> createState() => _TopicsTabState();
}

class _TopicsTabState extends ConsumerState<_TopicsTab> {
  final _searchCtrl = TextEditingController();
  List<Map<String, dynamic>>? _topics;
  Object? _error;
  bool _loading = true;

  @override
  void initState() {
    super.initState();
    _load();
  }

  @override
  void dispose() {
    _searchCtrl.dispose();
    super.dispose();
  }

  Future<void> _load() async {
    setState(() { _loading = true; _error = null; });
    try {
      final api = ref.read(apiClientProvider);
      final res = await api.get('/messaging/topics');
      final data = res.data as Map<String, dynamic>;
      setState(() {
        _topics = (data['topics'] as List<dynamic>?)
                ?.map((e) => e as Map<String, dynamic>)
                .toList() ??
            [];
      });
    } catch (e) {
      setState(() => _error = e);
    } finally {
      setState(() => _loading = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final cs = consoleColors(context);
    final topics = _topics ?? [];
    final search = _searchCtrl.text.toLowerCase();
    final filtered = search.isEmpty
        ? topics
        : topics
            .where((t) =>
                (t['name'] as String? ?? '').toLowerCase().contains(search))
            .toList();

    return Column(
      children: [
        Row(
          children: [
            SizedBox(
              width: 280,
              child: TextField(
                controller: _searchCtrl,
                onChanged: (_) => setState(() {}),
                style: TextStyle(fontSize: 13, color: cs.textPrimary),
                decoration: InputDecoration(
                  hintText: 'Search topics',
                  hintStyle: TextStyle(color: cs.textSubtle, fontSize: 13),
                  prefixIcon: Padding(
                    padding: const EdgeInsets.only(left: 10, right: 6),
                    child: Icon(LucideIcons.search, size: 15, color: cs.textSubtle),
                  ),
                  prefixIconConstraints: const BoxConstraints(minWidth: 32, minHeight: 0),
                  filled: true,
                  fillColor: cs.fieldFill,
                  isDense: true,
                  contentPadding: const EdgeInsets.symmetric(vertical: 10, horizontal: 12),
                  border: OutlineInputBorder(
                    borderRadius: BorderRadius.circular(8),
                    borderSide: BorderSide(color: cs.fieldBorder),
                  ),
                  enabledBorder: OutlineInputBorder(
                    borderRadius: BorderRadius.circular(8),
                    borderSide: BorderSide(color: cs.fieldBorder),
                  ),
                  focusedBorder: OutlineInputBorder(
                    borderRadius: BorderRadius.circular(8),
                    borderSide: const BorderSide(color: _accent),
                  ),
                ),
              ),
            ),
            const Spacer(),
            FilledButton.icon(
              onPressed: () => _createTopicDialog(context),
              style: FilledButton.styleFrom(
                backgroundColor: _accent,
                padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
                shape: RoundedRectangleBorder(
                    borderRadius: BorderRadius.circular(8)),
              ),
              icon: const Icon(LucideIcons.plus, size: 15),
              label: const Text('Create topic', style: TextStyle(fontSize: 13)),
            ),
          ],
        ),
        const SizedBox(height: 16),
        Expanded(
          child: _loading
              ? const Center(child: CircularProgressIndicator(strokeWidth: 2))
              : _error != null
                  ? AppErrorState(error: _error, onRetry: _load)
                  : topics.isEmpty
                      ? Container(
                          width: double.infinity,
                          decoration: BoxDecoration(
                            color: cs.surface,
                            borderRadius: BorderRadius.circular(8),
                            border: Border.all(color: cs.border),
                          ),
                          child: Column(
                            mainAxisAlignment: MainAxisAlignment.center,
                            children: [
                              Text(
                                'Create your first topic',
                                style: TextStyle(
                                    color: cs.textPrimary,
                                    fontSize: 15,
                                    fontWeight: FontWeight.w500),
                              ),
                              const SizedBox(height: 8),
                              Text(
                                'Group targets and broadcast messages to all of them at once.',
                                style: TextStyle(color: cs.textSecondary, fontSize: 13),
                              ),
                              const SizedBox(height: 20),
                              OutlinedButton(
                                onPressed: () => _createTopicDialog(context),
                                style: OutlinedButton.styleFrom(
                                  foregroundColor: cs.textSecondary,
                                  side: BorderSide(color: cs.border),
                                  padding: const EdgeInsets.symmetric(
                                      horizontal: 16, vertical: 10),
                                  shape: RoundedRectangleBorder(
                                      borderRadius: BorderRadius.circular(8)),
                                  textStyle: const TextStyle(fontSize: 13),
                                ),
                                child: const Text('Create topic'),
                              ),
                            ],
                          ),
                        )
                      : filtered.isEmpty
                          ? Center(
                              child: Text(
                                'No topics match "$search"',
                                style: TextStyle(color: cs.textMuted, fontSize: 13),
                              ),
                            )
                          : ListView.builder(
                              itemCount: filtered.length,
                              itemBuilder: (_, i) {
                                final t = filtered[i];
                                final subs =
                                    (t['subscribers'] as List<dynamic>?)?.length ?? 0;
                                return Container(
                                  margin: const EdgeInsets.only(bottom: 8),
                                  padding: const EdgeInsets.all(16),
                                  decoration: BoxDecoration(
                                    color: cs.surface,
                                    borderRadius: BorderRadius.circular(8),
                                    border: Border.all(color: cs.border),
                                  ),
                                  child: Row(
                                    children: [
                                      Icon(LucideIcons.hash, size: 14, color: cs.textSubtle),
                                      const SizedBox(width: 8),
                                      Text(t['name'] ?? '',
                                          style: TextStyle(
                                              color: cs.textPrimary, fontSize: 14)),
                                      const Spacer(),
                                      Text(
                                        '$subs subscriber${subs == 1 ? '' : 's'}',
                                        style: TextStyle(color: cs.textSubtle, fontSize: 12),
                                      ),
                                    ],
                                  ),
                                );
                              },
                            ),
        ),
      ],
    );
  }

  void _createTopicDialog(BuildContext context) {
    final ctrl = TextEditingController();
    showAppDialog(
      context: context,
      title: 'Create topic',
      subtitle: 'Group targets for broadcast messaging',
      content: AppDialogField(
          controller: ctrl,
          label: 'Topic name',
          hint: 'my-topic',
          autofocus: true),
      actions: [
        const AppDialogCancel(),
        AppDialogAction(
          label: 'Create',
          onTap: () async {
            if (ctrl.text.trim().isEmpty) return;
            try {
              final api = ref.read(apiClientProvider);
              await api.post('/messaging/topics',
                  data: {'name': ctrl.text.trim()});
            } catch (_) {}
            if (context.mounted) Navigator.of(context).pop();
            await _load();
          },
        ),
      ],
    );
  }
}

// ── Providers Tab ─────────────────────────────────────────────────────────────
class _ProvidersTab extends StatelessWidget {
  const _ProvidersTab();

  static const _providers = [
    // Email
    (
      icon: LucideIcons.mail,
      category: 'Email',
      title: 'SMTP',
      subtitle: 'Send emails via any SMTP server (Gmail, SendGrid, etc.)',
      vars: 'SMTP_HOST, SMTP_PORT, SMTP_USER, SMTP_PASS, SMTP_FROM',
    ),
    (
      icon: LucideIcons.mail,
      category: 'Email',
      title: 'Mailgun',
      subtitle: 'Send emails via the Mailgun API',
      vars: 'MAILGUN_API_KEY, MAILGUN_DOMAIN',
    ),
    (
      icon: LucideIcons.mail,
      category: 'Email',
      title: 'Resend',
      subtitle: 'Send emails via the Resend API',
      vars: 'RESEND_API_KEY',
    ),
    // SMS
    (
      icon: LucideIcons.messageSquare,
      category: 'SMS',
      title: 'Twilio',
      subtitle: 'Send SMS messages via Twilio',
      vars: 'TWILIO_SID, TWILIO_TOKEN, TWILIO_FROM',
    ),
    (
      icon: LucideIcons.messageSquare,
      category: 'SMS',
      title: 'Vonage (Nexmo)',
      subtitle: 'Send SMS messages via the Vonage API',
      vars: 'VONAGE_API_KEY, VONAGE_API_SECRET, VONAGE_FROM',
    ),
    (
      icon: LucideIcons.messageSquare,
      category: 'SMS',
      title: 'MSG91',
      subtitle: 'Send SMS messages via MSG91 (India)',
      vars: 'MSG91_AUTH_KEY, MSG91_SENDER_ID',
    ),
    // Push
    (
      icon: LucideIcons.bell,
      category: 'Push',
      title: 'Firebase Cloud Messaging (FCM)',
      subtitle: 'Send Android & web push notifications via FCM legacy API',
      vars: 'FCM_SERVER_KEY',
    ),
    (
      icon: LucideIcons.bell,
      category: 'Push',
      title: 'Apple Push Notification Service (APNS)',
      subtitle: 'Send iOS push notifications via APNS HTTP/2',
      vars: 'APNS_KEY_ID, APNS_TEAM_ID, APNS_KEY_PATH, APNS_BUNDLE_ID',
    ),
  ];

  @override
  Widget build(BuildContext context) {
    final cs = consoleColors(context);
    // Group by category
    final categories = ['Email', 'SMS', 'Push'];
    return SingleChildScrollView(
      padding: const EdgeInsets.only(bottom: 32),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          for (final cat in categories) ...[
            Padding(
              padding: const EdgeInsets.only(bottom: 10),
              child: Text(
                cat,
                style: TextStyle(
                  color: cs.textSubtle,
                  fontSize: 11,
                  fontWeight: FontWeight.w600,
                  letterSpacing: 0.6,
                ),
              ),
            ),
            ...(_providers
                .where((p) => p.category == cat)
                .map((p) => Padding(
                      padding: const EdgeInsets.only(bottom: 8),
                      child: _providerCard(context, p),
                    ))),
            const SizedBox(height: 16),
          ],
        ],
      ),
    );
  }

  Widget _providerCard(BuildContext context,
      ({IconData icon, String category, String title, String subtitle, String vars}) p) {
    final cs = consoleColors(context);
    return Container(
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: cs.surface,
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: cs.border),
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Container(
            width: 40,
            height: 40,
            decoration: BoxDecoration(
              color: cs.fill,
              borderRadius: BorderRadius.circular(8),
            ),
            child: Icon(p.icon,
                size: 18, color: cs.textMuted),
          ),
          const SizedBox(width: 16),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(p.title,
                    style: TextStyle(
                        color: cs.textPrimary,
                        fontSize: 14,
                        fontWeight: FontWeight.w500)),
                const SizedBox(height: 4),
                Text(p.subtitle,
                    style: TextStyle(
                        color: cs.textMuted,
                        fontSize: 13)),
                const SizedBox(height: 8),
                Text(
                  'Env vars: ${p.vars}',
                  style: TextStyle(
                      color: cs.textSubtle,
                      fontSize: 11,
                      fontFamily: 'monospace'),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

// ── Templates Tab ──────────────────────────────────────────────────────────────

class _TemplatesTab extends ConsumerStatefulWidget {
  const _TemplatesTab();

  @override
  ConsumerState<_TemplatesTab> createState() => _TemplatesTabState();
}

class _TemplatesTabState extends ConsumerState<_TemplatesTab> {
  @override
  Widget build(BuildContext context) {
    final cs = consoleColors(context);
    final async = ref.watch(_templatesApiProvider);

    return async.when(
      loading: () => const Center(child: CircularProgressIndicator()),
      error: (e, _) => AppErrorState(error: e),
      data: (data) {
        final templates = (data['templates'] as List<dynamic>? ?? [])
            .map((t) => Map<String, dynamic>.from(t as Map))
            .toList();

        return Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Text('${templates.length} templates',
                    style:
                        TextStyle(color: cs.textSecondary, fontSize: 13)),
                const Spacer(),
                FilledButton.icon(
                  style: FilledButton.styleFrom(backgroundColor: _accent),
                  onPressed: () => _showCreateDialog(context),
                  icon: const Icon(LucideIcons.plus, size: 14),
                  label: const Text('New Template'),
                ),
              ],
            ),
            const SizedBox(height: 16),
            if (templates.isEmpty)
              Center(
                child: Column(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    Icon(LucideIcons.fileText,
                        size: 40, color: cs.textSubtle),
                    const SizedBox(height: 12),
                    Text('No templates yet',
                        style: TextStyle(
                            color: cs.textPrimary,
                            fontSize: 15,
                            fontWeight: FontWeight.w500)),
                    const SizedBox(height: 4),
                    Text('Create reusable message templates with variables.',
                        style: TextStyle(
                            color: cs.textSecondary, fontSize: 13)),
                  ],
                ),
              )
            else
              Expanded(
                child: ListView.separated(
                  itemCount: templates.length,
                  separatorBuilder: (_, __) =>
                      Divider(height: 1, color: cs.border),
                  itemBuilder: (context, i) =>
                      _TemplateRow(tmpl: templates[i]),
                ),
              ),
          ],
        );
      },
    );
  }

  void _showCreateDialog(BuildContext context) {
    showAppDialog(
      context: context,
      title: 'New Template',
      subtitle: 'Create a reusable message with {{variable}} placeholders',
      width: 500,
      content: _CreateTemplateForm(
        onCreated: () => ref.invalidate(_templatesApiProvider),
      ),
      actions: const [AppDialogCancel()],
    );
  }
}

// ── Create Template Form (stateful widget used inside showAppDialog) ───────────

class _CreateTemplateForm extends ConsumerStatefulWidget {
  final VoidCallback onCreated;
  const _CreateTemplateForm({required this.onCreated});

  @override
  ConsumerState<_CreateTemplateForm> createState() =>
      _CreateTemplateFormState();
}

class _CreateTemplateFormState extends ConsumerState<_CreateTemplateForm> {
  final _nameCtrl = TextEditingController();
  final _subjectCtrl = TextEditingController();
  final _bodyCtrl = TextEditingController();
  String _type = 'email';
  bool _saving = false;

  @override
  void dispose() {
    _nameCtrl.dispose();
    _subjectCtrl.dispose();
    _bodyCtrl.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    if (_nameCtrl.text.trim().isEmpty) return;
    setState(() => _saving = true);
    try {
      final api = ref.read(apiClientProvider);
      await api.post('/messaging/templates', data: {
        'templateId': 'unique()',
        'name': _nameCtrl.text.trim(),
        'type': _type,
        'subject': _subjectCtrl.text.trim(),
        'body': _bodyCtrl.text.trim(),
        'variables': [],
      });
      widget.onCreated();
      if (mounted) Navigator.of(context).pop();
    } finally {
      if (mounted) setState(() => _saving = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final cs = consoleColors(context);

    InputDecoration fieldDeco(String hint) => InputDecoration(
          hintText: hint,
          hintStyle: TextStyle(color: cs.textSubtle),
          filled: true,
          fillColor: cs.fieldFill,
          isDense: true,
          contentPadding:
              const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
          border: OutlineInputBorder(
              borderRadius: BorderRadius.circular(8),
              borderSide: BorderSide(color: cs.fieldBorder)),
          enabledBorder: OutlineInputBorder(
              borderRadius: BorderRadius.circular(8),
              borderSide: BorderSide(color: cs.fieldBorder)),
          focusedBorder: OutlineInputBorder(
              borderRadius: BorderRadius.circular(8),
              borderSide: const BorderSide(color: _accent)),
        );

    return Column(
      mainAxisSize: MainAxisSize.min,
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text('Name',
            style: TextStyle(
                color: cs.textSecondary,
                fontSize: 12,
                fontWeight: FontWeight.w500)),
        const SizedBox(height: 4),
        TextField(
          controller: _nameCtrl,
          style: TextStyle(color: cs.textPrimary, fontSize: 13),
          decoration: fieldDeco('Welcome Email'),
        ),
        const SizedBox(height: 12),
        Text('Type',
            style: TextStyle(
                color: cs.textSecondary,
                fontSize: 12,
                fontWeight: FontWeight.w500)),
        const SizedBox(height: 4),
        SegmentedButton<String>(
          segments: const [
            ButtonSegment(value: 'email', label: Text('Email')),
            ButtonSegment(value: 'sms', label: Text('SMS')),
            ButtonSegment(value: 'push', label: Text('Push')),
          ],
          selected: {_type},
          onSelectionChanged: (s) => setState(() => _type = s.first),
        ),
        const SizedBox(height: 12),
        Text('Subject',
            style: TextStyle(
                color: cs.textSecondary,
                fontSize: 12,
                fontWeight: FontWeight.w500)),
        const SizedBox(height: 4),
        TextField(
          controller: _subjectCtrl,
          style: TextStyle(color: cs.textPrimary, fontSize: 13),
          decoration: fieldDeco('Hello {{name}}!'),
        ),
        const SizedBox(height: 12),
        Text('Body',
            style: TextStyle(
                color: cs.textSecondary,
                fontSize: 12,
                fontWeight: FontWeight.w500)),
        const SizedBox(height: 4),
        TextField(
          controller: _bodyCtrl,
          maxLines: 5,
          style: TextStyle(
              color: cs.textPrimary,
              fontSize: 13,
              fontFamily: 'monospace'),
          decoration:
              fieldDeco('Hi {{name}}, welcome to {{appName}}!'),
        ),
        const SizedBox(height: 16),
        Row(
          mainAxisAlignment: MainAxisAlignment.end,
          children: [
            AppDialogAction(
              label: _saving ? 'Creating…' : 'Create',
              loading: _saving,
              onTap: _submit,
            ),
          ],
        ),
      ],
    );
  }
}

class _TemplateRow extends ConsumerWidget {
  final Map<String, dynamic> tmpl;
  const _TemplateRow({required this.tmpl});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final cs = consoleColors(context);
    final id = tmpl[r'$id'] as String? ?? '';
    final name = tmpl['name'] as String? ?? '';
    final type = tmpl['type'] as String? ?? 'email';
    final subject = tmpl['subject'] as String? ?? '';

    final typeColor = switch (type) {
      'email' => const Color(0xFF3472A4),
      'sms' => const Color(0xFF10B981),
      'push' => const Color(0xFFF59E0B),
      _ => const Color(0xFF6B7280),
    };

    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
      child: Row(
        children: [
          Container(
            width: 36,
            height: 36,
            decoration: BoxDecoration(
              color: typeColor.withAlpha(30),
              borderRadius: BorderRadius.circular(8),
            ),
            child: Icon(
              type == 'email'
                  ? LucideIcons.mail
                  : type == 'sms'
                      ? LucideIcons.messageSquare
                      : LucideIcons.bell,
              size: 16,
              color: typeColor,
            ),
          ),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(name,
                    style: TextStyle(
                        color: cs.textPrimary,
                        fontSize: 13,
                        fontWeight: FontWeight.w500)),
                if (subject.isNotEmpty)
                  Text(subject,
                      style: TextStyle(
                          color: cs.textSubtle, fontSize: 12),
                      overflow: TextOverflow.ellipsis),
              ],
            ),
          ),
          Container(
            padding:
                const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
            decoration: BoxDecoration(
              color: typeColor.withAlpha(30),
              borderRadius: BorderRadius.circular(12),
            ),
            child: Text(type.toUpperCase(),
                style: TextStyle(
                    color: typeColor,
                    fontSize: 10,
                    fontWeight: FontWeight.w600)),
          ),
          const SizedBox(width: 8),
          IconButton(
            icon: Icon(LucideIcons.trash2,
                size: 14, color: cs.textSubtle),
            onPressed: () async {
              final confirmed = await showAppDialog<bool>(
                context: context,
                title: 'Delete template',
                content: Text(
                  'Are you sure? This action cannot be undone.',
                  style: TextStyle(color: cs.textSecondary),
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
              await api.delete('/messaging/templates/$id');
              ref.invalidate(_templatesApiProvider);
            },
            tooltip: 'Delete template',
          ),
        ],
      ),
    );
  }
}
