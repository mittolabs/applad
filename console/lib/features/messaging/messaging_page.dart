import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:lucide_icons/lucide_icons.dart';
import '../../core/api/client.dart';
import '../../core/widgets/app_dialog.dart';
import '../../core/widgets/page_tabs.dart';

// ── Constants ──────────────────────────────────────────────────────────────────
const _bgColor = Color(0xFF0B0B0F);
const _cardColor = Color(0xFF16171B);
const _accent = Color(0xFF3472A4);
const _dimText = Color(0x80FFFFFF);
const _subtleText = Color(0x40FFFFFF);
const _border = Color(0x14FFFFFF);
const _red = Color(0xFFEF4444);

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
final _msgTabProvider = StateProvider<int>((ref) => 0);
final _msgSearchProvider = StateProvider<String>((ref) => '');
final _msgRefreshProvider = StateProvider<int>((ref) => 0);

final _messagesApiProvider =
    FutureProvider.autoDispose<Map<String, dynamic>>((ref) async {
  final api = ref.read(apiClientProvider);
  ref.watch(_msgRefreshProvider); // rebuild on refresh
  final search = ref.watch(_msgSearchProvider);
  final params = <String, dynamic>{'limit': 50};
  if (search.isNotEmpty) params['search'] = search;
  final res = await api.get('/messaging/messages', params: params);
  return res.data as Map<String, dynamic>;
});

// ── Page ──────────────────────────────────────────────────────────────────────
class MessagingPage extends ConsumerWidget {
  const MessagingPage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final tab = ref.watch(_msgTabProvider);

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
            const Text(
              'Messaging',
              style: TextStyle(
                color: Colors.white,
                fontSize: 22,
                fontWeight: FontWeight.w600,
              ),
            ),
            const SizedBox(height: 24),
            PageTabs(
              tabs: const ['Messages', 'Topics', 'Providers'],
              selected: tab,
              onChanged: (i) => ref.read(_msgTabProvider.notifier).state = i,
            ),
            const SizedBox(height: 20),
            Expanded(
              child: IndexedStack(
                index: tab,
                children: const [
                  _MessagesTab(),
                  _TopicsTab(),
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

  void _refresh() =>
      ref.read(_msgRefreshProvider.notifier).state++;

  @override
  Widget build(BuildContext context) {
    if (_selected != null) {
      return _MessageDetail(
        msg: _selected!,
        onBack: () => setState(() => _selected = null),
      );
    }

    final async = ref.watch(_messagesApiProvider);

    return Column(
        children: [
          Row(
            children: [
              SizedBox(
                width: 280,
                child: TextField(
                  controller: _searchCtrl,
                  onChanged: (v) {
                    ref.read(_msgSearchProvider.notifier).state = v;
                    _refresh();
                  },
                  style: const TextStyle(fontSize: 13, color: Colors.white),
                  decoration: InputDecoration(
                    hintText: 'Search by description, type, status, or ID',
                    hintStyle: const TextStyle(color: _subtleText, fontSize: 13),
                    prefixIcon: const Padding(
                      padding: EdgeInsets.only(left: 10, right: 6),
                      child: Icon(Icons.search, size: 16, color: _subtleText),
                    ),
                    prefixIconConstraints: const BoxConstraints(minWidth: 32, minHeight: 0),
                    filled: true,
                    fillColor: const Color(0x0AFFFFFF),
                    isDense: true,
                    contentPadding: const EdgeInsets.symmetric(vertical: 10, horizontal: 12),
                    border: OutlineInputBorder(
                      borderRadius: BorderRadius.circular(8),
                      borderSide: BorderSide(color: Colors.white.withOpacity(0.08)),
                    ),
                    enabledBorder: OutlineInputBorder(
                      borderRadius: BorderRadius.circular(8),
                      borderSide: BorderSide(color: Colors.white.withOpacity(0.08)),
                    ),
                    focusedBorder: OutlineInputBorder(
                      borderRadius: BorderRadius.circular(8),
                      borderSide: const BorderSide(color: _accent),
                    ),
                  ),
                ),
              ),
              const Spacer(),
              _CreateMessageButton(onCreated: _refresh),
            ],
          ),
          const SizedBox(height: 16),
          Expanded(
            child: async.when(
              loading: () => const Center(
                  child: CircularProgressIndicator(strokeWidth: 2)),
              error: (e, _) => Center(
                  child: Text('Error: $e',
                      style: const TextStyle(color: Color(0xFFEF4444)))),
              data: (data) {
                final messages = (data['messages'] as List<dynamic>? ?? [])
                    .map((e) => _Msg.fromJson(e as Map<String, dynamic>))
                    .toList();
                return messages.isEmpty
                    ? _EmptyMessages(onCreateTap: () {})
                    : _MessageTable(
                        messages: messages,
                        onRowTap: (msg) =>
                            setState(() => _selected = msg),
                      );
              },
            ),
          ),
        ],
    );
  }
}

// ── Empty state ───────────────────────────────────────────────────────────────
class _EmptyMessages extends StatelessWidget {
  final VoidCallback onCreateTap;
  const _EmptyMessages({required this.onCreateTap});

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Container(
            width: 48,
            height: 48,
            decoration: BoxDecoration(
              color: Colors.white.withOpacity(0.04),
              borderRadius: BorderRadius.circular(12),
            ),
            child: const Icon(LucideIcons.messageSquare,
                size: 22, color: _subtleText),
          ),
          const SizedBox(height: 16),
          const Text('No messages',
              style: TextStyle(
                  color: Colors.white,
                  fontSize: 15,
                  fontWeight: FontWeight.w500)),
          const SizedBox(height: 6),
          const Text('Create a message to send email, SMS, or push notifications.',
              style: TextStyle(color: _dimText, fontSize: 13)),
          const SizedBox(height: 16),
          FilledButton(
            style: FilledButton.styleFrom(
              backgroundColor: _accent,
              padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 10),
              shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(8)),
            ),
            onPressed: onCreateTap,
            child: const Text('Create message', style: TextStyle(fontSize: 13)),
          ),
        ],
      ),
    );
  }
}

// ── Message Table ─────────────────────────────────────────────────────────────
class _MessageTable extends StatelessWidget {
  final List<_Msg> messages;
  final ValueChanged<_Msg> onRowTap;

  const _MessageTable({required this.messages, required this.onRowTap});

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        // Header row
        Container(
          padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
          decoration: BoxDecoration(
            border: Border(
                bottom: BorderSide(color: Colors.white.withOpacity(0.06))),
          ),
          child: const Row(
            children: [
              Expanded(flex: 3, child: Text('Message ID', style: TextStyle(color: _dimText, fontSize: 12, fontWeight: FontWeight.w500))),
              Expanded(flex: 2, child: Text('Type', style: TextStyle(color: _dimText, fontSize: 12, fontWeight: FontWeight.w500))),
              Expanded(flex: 2, child: Text('Status', style: TextStyle(color: _dimText, fontSize: 12, fontWeight: FontWeight.w500))),
              Expanded(flex: 2, child: Text('Created', style: TextStyle(color: _dimText, fontSize: 12, fontWeight: FontWeight.w500))),
            ],
          ),
        ),
        // Data rows
        Expanded(
          child: ListView.builder(
            itemCount: messages.length,
            itemBuilder: (_, i) => _MsgRow(
              msg: messages[i],
              onTap: () => onRowTap(messages[i]),
            ),
          ),
        ),
      ],
    );
  }
}

// ── Message row ───────────────────────────────────────────────────────────────
class _MsgRow extends StatefulWidget {
  final _Msg msg;
  final VoidCallback onTap;

  const _MsgRow({required this.msg, required this.onTap});

  @override
  State<_MsgRow> createState() => _MsgRowState();
}

class _MsgRowState extends State<_MsgRow> {
  bool _hovered = false;

  @override
  Widget build(BuildContext context) {
    final msg = widget.msg;
    return MouseRegion(
      onEnter: (_) => setState(() => _hovered = true),
      onExit: (_) => setState(() => _hovered = false),
      cursor: SystemMouseCursors.click,
      child: GestureDetector(
        onTap: widget.onTap,
        child: Container(
          padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
          decoration: BoxDecoration(
            color: _hovered ? Colors.white.withOpacity(0.02) : null,
            border: Border(
                bottom: BorderSide(color: Colors.white.withOpacity(0.04))),
          ),
          child: Row(
            children: [
              Expanded(
                flex: 3,
                child: Row(
                  children: [
                    Icon(_typeIcon(msg.type), size: 14, color: _accent),
                    const SizedBox(width: 8),
                    Expanded(
                      child: Text(msg.id,
                          style: const TextStyle(
                              color: Colors.white,
                              fontSize: 13,
                              fontFamily: 'monospace'),
                          overflow: TextOverflow.ellipsis),
                    ),
                  ],
                ),
              ),
              Expanded(
                flex: 2,
                child: Text(_typeName(msg.type),
                    style: const TextStyle(color: _dimText, fontSize: 13)),
              ),
              Expanded(
                flex: 2,
                child: _StatusBadge(status: msg.status),
              ),
              Expanded(
                flex: 2,
                child: Text(
                    msg.createdAt.isNotEmpty ? msg.createdAt : '—',
                    style: const TextStyle(color: _dimText, fontSize: 12)),
              ),
            ],
          ),
        ),
      ),
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
    return PopupMenuButton<String>(
      offset: const Offset(0, 40),
      color: const Color(0xFF1E1F24),
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(8),
        side: const BorderSide(color: _border),
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
    return Scaffold(
      backgroundColor: _bgColor,
      body: Padding(
        padding: EdgeInsets.symmetric(
          horizontal: MediaQuery.of(context).size.width > 1400 ? 80 : 40,
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
    return Padding(
      padding: const EdgeInsets.only(top: 32, bottom: 4),
      child: Row(
        children: [
          GestureDetector(
            onTap: () => Navigator.of(context).pop(),
            child: MouseRegion(
              cursor: SystemMouseCursors.click,
              child: Icon(LucideIcons.arrowLeft,
                  size: 20, color: Colors.white.withOpacity(0.5)),
            ),
          ),
          const SizedBox(width: 12),
          Text(
            _title,
            style: const TextStyle(
              color: Colors.white,
              fontSize: 22,
              fontWeight: FontWeight.w600,
            ),
          ),
        ],
      ),
    );
  }

  Widget _dialogActions() {
    return Container(
      padding: const EdgeInsets.symmetric(vertical: 16),
      decoration: BoxDecoration(
        border: Border(top: BorderSide(color: Colors.white.withOpacity(0.06))),
      ),
      child: Row(
        mainAxisAlignment: MainAxisAlignment.end,
        children: [
          TextButton(
            onPressed: () => Navigator.of(context).pop(),
            style: TextButton.styleFrom(
              foregroundColor: Colors.white54,
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
              foregroundColor: Colors.white70,
              side: BorderSide(
                  color: Colors.white.withOpacity(0.18)),
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
                      color: Colors.white.withOpacity(0.3),
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
                      color: Colors.white.withOpacity(0.3),
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
    return Padding(
      padding: const EdgeInsets.only(bottom: 14),
      child: Text(
        label,
        style: TextStyle(
          color: Colors.white.withOpacity(0.45),
          fontSize: 11,
          fontWeight: FontWeight.w600,
          letterSpacing: 0.6,
        ),
      ),
    );
  }

  Widget _fieldLabel(String label, {bool optional = false}) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 6),
      child: Row(
        children: [
          Text(label,
              style: const TextStyle(
                  color: Colors.white70, fontSize: 13)),
          if (optional) ...[
            const SizedBox(width: 4),
            Text('optional',
                style: TextStyle(
                    color: Colors.white.withOpacity(0.3),
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
    final multi = maxLines > 1;
    return TextField(
      controller: ctrl,
      autofocus: autofocus,
      maxLines: multi ? null : 1,
      minLines: multi ? maxLines : 1,
      style: const TextStyle(color: Colors.white, fontSize: 13),
      decoration: InputDecoration(
        hintText: hint,
        hintStyle: TextStyle(
            color: Colors.white.withOpacity(0.22), fontSize: 13),
        filled: true,
        fillColor: const Color(0x0AFFFFFF),
        isDense: true,
        contentPadding: multi
            ? const EdgeInsets.all(12)
            : const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
        border: OutlineInputBorder(
          borderRadius: BorderRadius.circular(8),
          borderSide:
              BorderSide(color: Colors.white.withOpacity(0.1)),
        ),
        enabledBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(8),
          borderSide:
              BorderSide(color: Colors.white.withOpacity(0.1)),
        ),
        focusedBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(8),
          borderSide: const BorderSide(color: _accent),
        ),
      ),
    );
  }

  Widget _htmlToggle() {
    return Row(
      children: [
        Switch(
          value: _htmlMode,
          onChanged: (v) => setState(() => _htmlMode = v),
          activeColor: _accent,
          materialTapTargetSize: MaterialTapTargetSize.shrinkWrap,
        ),
        const SizedBox(width: 10),
        Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const Text('HTML mode',
                style: TextStyle(color: Colors.white70, fontSize: 13)),
            const SizedBox(height: 2),
            Text(
              'Enable the HTML mode if your message contains HTML tags.',
              style: TextStyle(
                  color: Colors.white.withOpacity(0.3),
                  fontSize: 11),
            ),
          ],
        ),
      ],
    );
  }

  Widget _mediaUpload() {
    return Container(
      height: 90,
      width: double.infinity,
      decoration: BoxDecoration(
        color: const Color(0x0AFFFFFF),
        borderRadius: BorderRadius.circular(8),
        border:
            Border.all(color: Colors.white.withOpacity(0.1)),
      ),
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          Icon(LucideIcons.upload,
              size: 20,
              color: Colors.white.withOpacity(0.3)),
          const SizedBox(height: 6),
          Text('Select a file to upload',
              style: TextStyle(
                  color: Colors.white.withOpacity(0.45),
                  fontSize: 13)),
          const SizedBox(height: 4),
          Row(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              Text('Max file size: 1MB',
                  style: TextStyle(
                      color: Colors.white.withOpacity(0.25),
                      fontSize: 11)),
              const SizedBox(width: 8),
              MouseRegion(
                cursor: SystemMouseCursors.click,
                child: Text(
                  'Browse',
                  style: TextStyle(
                    color: Colors.white.withOpacity(0.5),
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
    return Container(
      decoration: BoxDecoration(
        color: const Color(0x0AFFFFFF),
        borderRadius: BorderRadius.circular(8),
        border:
            Border.all(color: Colors.white.withOpacity(0.1)),
      ),
      padding:
          const EdgeInsets.symmetric(horizontal: 12, vertical: 2),
      child: DropdownButtonHideUnderline(
        child: DropdownButton<String>(
          value: _schedule,
          isExpanded: true,
          dropdownColor: const Color(0xFF1E1F24),
          style:
              const TextStyle(color: Colors.white, fontSize: 13),
          icon: Icon(Icons.keyboard_arrow_down,
              color: Colors.white.withOpacity(0.4), size: 18),
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
    return Row(
      children: [
        Icon(Icons.info_outline,
            size: 12,
            color: Colors.white.withOpacity(0.3)),
        const SizedBox(width: 5),
        Text(
          'The message will be sent immediately',
          style: TextStyle(
              color: Colors.white.withOpacity(0.3),
              fontSize: 12),
        ),
      ],
    );
  }

  Widget _divider() {
    return Column(
      children: [
        Divider(
            color: Colors.white.withOpacity(0.06), height: 1),
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
            Border.all(color: Colors.white.withOpacity(0.15), width: 2),
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
                            color: Colors.white.withOpacity(0.3),
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
                  color: Colors.white.withOpacity(0.8),
                  fontSize: 10,
                  fontWeight: FontWeight.w600)),
          const Spacer(),
          Icon(Icons.signal_cellular_alt,
              size: 10,
              color: Colors.white.withOpacity(0.7)),
          const SizedBox(width: 2),
          Icon(Icons.wifi,
              size: 10,
              color: Colors.white.withOpacity(0.7)),
          const SizedBox(width: 2),
          Icon(Icons.battery_full,
              size: 10,
              color: Colors.white.withOpacity(0.7)),
        ],
      ),
    );
  }

  Widget _contactAvatar() {
    return Container(
      width: 36,
      height: 36,
      decoration: BoxDecoration(
        color: Colors.white.withOpacity(0.1),
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
            Border.all(color: Colors.white.withOpacity(0.15), width: 2),
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
                                Colors.white.withOpacity(0.1),
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
                                            .withOpacity(0.5),
                                        fontSize: 9),
                                  ),
                                  const Spacer(),
                                  Text('now',
                                      style: TextStyle(
                                          color: Colors.white
                                              .withOpacity(0.4),
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
                                            .withOpacity(0.7),
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
                              color: Colors.white.withOpacity(0.3),
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
                  color: Colors.white.withOpacity(0.8),
                  fontSize: 10,
                  fontWeight: FontWeight.w600)),
          const Spacer(),
          Icon(Icons.signal_cellular_alt,
              size: 10,
              color: Colors.white.withOpacity(0.7)),
          const SizedBox(width: 2),
          Icon(Icons.wifi,
              size: 10,
              color: Colors.white.withOpacity(0.7)),
          const SizedBox(width: 2),
          Icon(Icons.battery_full,
              size: 10,
              color: Colors.white.withOpacity(0.7)),
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
    return Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          // Title row
          Row(
            children: [
              IconButton(
                icon: const Icon(Icons.arrow_back,
                    size: 18, color: Colors.white70),
                onPressed: onBack,
                padding: EdgeInsets.zero,
                constraints: const BoxConstraints(),
              ),
              const SizedBox(width: 8),
              Text(
                msg.subject.isNotEmpty ? msg.subject : _typeName(),
                style: const TextStyle(
                    color: Colors.white,
                    fontSize: 18,
                    fontWeight: FontWeight.w600),
              ),
              const SizedBox(width: 12),
              Container(
                padding: const EdgeInsets.symmetric(
                    horizontal: 8, vertical: 3),
                decoration: BoxDecoration(
                  color: Colors.white.withOpacity(0.07),
                  borderRadius: BorderRadius.circular(4),
                ),
                child: Row(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    Icon(LucideIcons.tag,
                        size: 10,
                        color: Colors.white.withOpacity(0.35)),
                    const SizedBox(width: 4),
                    Text(
                      msg.id,
                      style: TextStyle(
                        color: Colors.white.withOpacity(0.45),
                        fontSize: 11,
                        fontFamily: 'monospace',
                      ),
                    ),
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
                  _card(
                    child: Row(
                      children: [
                        Icon(_typeIconData(),
                            size: 16, color: Colors.white54),
                        const SizedBox(width: 8),
                        Text(_typeName(),
                            style: const TextStyle(
                                color: Colors.white,
                                fontSize: 14,
                                fontWeight: FontWeight.w500)),
                        const Spacer(),
                        Text(
                          'Created: ${_fmtDate(msg.createdAt)}',
                          style: TextStyle(
                              color: Colors.white.withOpacity(0.4),
                              fontSize: 12),
                        ),
                        const SizedBox(width: 16),
                        _StatusBadge(status: msg.status),
                      ],
                    ),
                  ),
                  const SizedBox(height: 12),
                  // Message card
                  _card(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        const Text('Message',
                            style: TextStyle(
                                color: Colors.white,
                                fontSize: 14,
                                fontWeight: FontWeight.w500)),
                        Divider(
                            color: Colors.white.withOpacity(0.06)),
                        if (msg.subject.isNotEmpty) ...[
                          _detailRow('Subject', msg.subject),
                          const SizedBox(height: 12),
                        ],
                        _detailRow(
                          msg.type == 'push' ? 'Body' : 'Message',
                          msg.body.isEmpty ? '-' : msg.body,
                        ),
                      ],
                    ),
                  ),
                  const SizedBox(height: 12),
                  // Targets card
                  _card(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        const Text('Targets',
                            style: TextStyle(
                                color: Colors.white,
                                fontSize: 14,
                                fontWeight: FontWeight.w500)),
                        Divider(
                            color: Colors.white.withOpacity(0.06)),
                        if (msg.recipients.isEmpty)
                          Padding(
                            padding:
                                const EdgeInsets.symmetric(vertical: 8),
                            child: Text(
                              'No targets selected',
                              style: TextStyle(
                                  color: Colors.white.withOpacity(0.3),
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
                                      color: Colors.white38),
                                  const SizedBox(width: 8),
                                  Text(r,
                                      style: const TextStyle(
                                          color: Colors.white70,
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

  Widget _card({required Widget child}) {
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: _cardColor,
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: Colors.white.withOpacity(0.08)),
      ),
      child: child,
    );
  }

  Widget _detailRow(String label, String value) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(label,
            style: TextStyle(
                color: Colors.white.withOpacity(0.4),
                fontSize: 11,
                fontWeight: FontWeight.w500)),
        const SizedBox(height: 4),
        Container(
          width: double.infinity,
          padding:
              const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
          decoration: BoxDecoration(
            color: const Color(0x08FFFFFF),
            borderRadius: BorderRadius.circular(6),
            border: Border.all(
                color: Colors.white.withOpacity(0.08)),
          ),
          child: Text(value,
              style: const TextStyle(
                  color: Colors.white70, fontSize: 13)),
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
    setState(() => _loading = true);
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
    } catch (_) {
      setState(() => _topics = []);
    } finally {
      setState(() => _loading = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final topics = _topics ?? [];
    final search = _searchCtrl.text.toLowerCase();
    final filtered = search.isEmpty
        ? topics
        : topics
            .where((t) =>
                (t['name'] as String? ?? '')
                    .toLowerCase()
                    .contains(search))
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
                  style: const TextStyle(fontSize: 13, color: Colors.white),
                  decoration: InputDecoration(
                    hintText: 'Search topics',
                    hintStyle: const TextStyle(color: _subtleText, fontSize: 13),
                    prefixIcon: const Padding(
                      padding: EdgeInsets.only(left: 10, right: 6),
                      child: Icon(Icons.search, size: 16, color: _subtleText),
                    ),
                    prefixIconConstraints: const BoxConstraints(minWidth: 32, minHeight: 0),
                    filled: true,
                    fillColor: const Color(0x0AFFFFFF),
                    isDense: true,
                    contentPadding: const EdgeInsets.symmetric(vertical: 10, horizontal: 12),
                    border: OutlineInputBorder(
                      borderRadius: BorderRadius.circular(8),
                      borderSide: BorderSide(color: Colors.white.withOpacity(0.08)),
                    ),
                    enabledBorder: OutlineInputBorder(
                      borderRadius: BorderRadius.circular(8),
                      borderSide: BorderSide(color: Colors.white.withOpacity(0.08)),
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
                  padding: const EdgeInsets.symmetric(
                      horizontal: 16, vertical: 10),
                  shape: RoundedRectangleBorder(
                      borderRadius: BorderRadius.circular(8)),
                ),
                icon: const Icon(Icons.add, size: 16),
                label: const Text('Create topic',
                    style: TextStyle(fontSize: 13)),
              ),
            ],
          ),
          const SizedBox(height: 16),
          Expanded(
            child: _loading
                ? const Center(
                    child: CircularProgressIndicator(strokeWidth: 2))
                : filtered.isEmpty
                    ? Container(
                        width: double.infinity,
                        decoration: BoxDecoration(
                          color: _cardColor,
                          borderRadius: BorderRadius.circular(8),
                          border: Border.all(color: _border),
                        ),
                        child: Column(
                          mainAxisAlignment: MainAxisAlignment.center,
                          children: [
                            const Text(
                              'Create your first topic',
                              style: TextStyle(
                                  color: Colors.white,
                                  fontSize: 16,
                                  fontWeight: FontWeight.w500),
                            ),
                            const SizedBox(height: 8),
                            const Text(
                              'Group targets and broadcast messages to all of them at once.',
                              style:
                                  TextStyle(color: _dimText, fontSize: 13),
                            ),
                            const SizedBox(height: 20),
                            OutlinedButton(
                              onPressed: () => _createTopicDialog(context),
                              style: OutlinedButton.styleFrom(
                                foregroundColor: Colors.white70,
                                side: const BorderSide(color: _border),
                                padding: const EdgeInsets.symmetric(
                                    horizontal: 16, vertical: 10),
                                shape: RoundedRectangleBorder(
                                    borderRadius: BorderRadius.circular(8)),
                              ),
                              child: const Text('Create topic',
                                  style: TextStyle(fontSize: 13)),
                            ),
                          ],
                        ),
                      )
                    : ListView.builder(
                        itemCount: filtered.length,
                        itemBuilder: (_, i) {
                          final t = filtered[i];
                          final subs =
                              (t['subscribers'] as List<dynamic>?)?.length ??
                                  0;
                          return Container(
                            margin: const EdgeInsets.only(bottom: 8),
                            padding: const EdgeInsets.all(16),
                            decoration: BoxDecoration(
                              color: _cardColor,
                              borderRadius: BorderRadius.circular(8),
                              border: Border.all(color: _border),
                            ),
                            child: Row(
                              children: [
                                Icon(LucideIcons.hash,
                                    size: 14,
                                    color: Colors.white.withOpacity(0.4)),
                                const SizedBox(width: 8),
                                Text(t['name'] ?? '',
                                    style: const TextStyle(
                                        color: Colors.white, fontSize: 14)),
                                const Spacer(),
                                Text(
                                  '$subs subscriber${subs == 1 ? '' : 's'}',
                                  style: TextStyle(
                                      color: Colors.white.withOpacity(0.4),
                                      fontSize: 12),
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
                  color: Colors.white.withOpacity(0.4),
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
                      child: _providerCard(p),
                    ))),
            const SizedBox(height: 16),
          ],
        ],
      ),
    );
  }

  Widget _providerCard(
      ({IconData icon, String category, String title, String subtitle, String vars}) p) {
    return Container(
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: _cardColor,
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: _border),
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Container(
            width: 40,
            height: 40,
            decoration: BoxDecoration(
              color: Colors.white.withOpacity(0.07),
              borderRadius: BorderRadius.circular(8),
            ),
            child: Icon(p.icon,
                size: 18, color: Colors.white.withOpacity(0.55)),
          ),
          const SizedBox(width: 16),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(p.title,
                    style: const TextStyle(
                        color: Colors.white,
                        fontSize: 14,
                        fontWeight: FontWeight.w500)),
                const SizedBox(height: 4),
                Text(p.subtitle,
                    style: TextStyle(
                        color: Colors.white.withOpacity(0.45),
                        fontSize: 13)),
                const SizedBox(height: 8),
                Text(
                  'Env vars: ${p.vars}',
                  style: TextStyle(
                      color: Colors.white.withOpacity(0.28),
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

// ── Shared search field ───────────────────────────────────────────────────────
class _SearchField extends StatelessWidget {
  final TextEditingController controller;
  final String hint;
  final ValueChanged<String> onChanged;

  const _SearchField({
    required this.controller,
    required this.hint,
    required this.onChanged,
  });

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      width: 300,
      child: TextField(
        controller: controller,
        onChanged: onChanged,
        style: const TextStyle(fontSize: 13, color: Colors.white),
        decoration: InputDecoration(
          hintText: hint,
          hintStyle:
              const TextStyle(color: _subtleText, fontSize: 13),
          prefixIcon: const Padding(
            padding: EdgeInsets.only(left: 10, right: 6),
            child: Icon(Icons.search, size: 16, color: _subtleText),
          ),
          prefixIconConstraints:
              const BoxConstraints(minWidth: 32),
          filled: true,
          fillColor: const Color(0x0AFFFFFF),
          isDense: true,
          contentPadding: const EdgeInsets.symmetric(
              vertical: 10, horizontal: 12),
          border: OutlineInputBorder(
            borderRadius: BorderRadius.circular(8),
            borderSide: const BorderSide(color: _border),
          ),
          enabledBorder: OutlineInputBorder(
            borderRadius: BorderRadius.circular(8),
            borderSide: const BorderSide(color: _border),
          ),
          focusedBorder: OutlineInputBorder(
            borderRadius: BorderRadius.circular(8),
            borderSide: const BorderSide(color: _accent),
          ),
        ),
      ),
    );
  }
}
