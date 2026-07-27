import 'package:applad/applad.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import '../applad_service.dart';
import '../config.dart';

/// One channel's conversation. Messages are rows in the `chat.messages` table,
/// each permissioned `read/write("team:<channelId>")` so only members of the
/// channel's team can see or post (audit G2). Live delivery is a realtime
/// subscription to the messages table (audit G5 notes this is table-wide today,
/// so we filter by channel client-side).
class ChatScreen extends StatefulWidget {
  const ChatScreen({super.key, required this.channelId, required this.channelName});

  final String channelId;
  final String channelName;

  @override
  State<ChatScreen> createState() => _ChatScreenState();
}

class _ChatScreenState extends State<ChatScreen> {
  final _composer = TextEditingController();
  final _scroll = ScrollController();
  final List<Map<String, dynamic>> _messages = [];
  final Set<String> _seen = {};
  RealtimeSubscription? _sub;
  bool _loading = true;
  bool _sending = false;

  Applad get _client => AppladService.instance.client;

  @override
  void initState() {
    super.initState();
    _load();
    _subscribe();
  }

  @override
  void dispose() {
    _sub?.cancel();
    _composer.dispose();
    _scroll.dispose();
    super.dispose();
  }

  Future<void> _load() async {
    try {
      final res = await _client.databases
          .from(Config.databaseId, Config.messagesTable)
          .equal('channel_id', widget.channelId)
          .orderAsc('created_at')
          .limit(200)
          .get();
      setState(() {
        for (final row in res.rows) {
          _add(row);
        }
        _loading = false;
      });
      _scrollToBottom();
    } catch (_) {
      setState(() => _loading = false);
    }
  }

  void _subscribe() {
    _sub = _client.realtime
        .database(Config.databaseId, Config.messagesTable)
        .onInsert((row) {
      if (row['channel_id']?.toString() != widget.channelId) return;
      if (!mounted) return;
      setState(() => _add(row));
      _scrollToBottom();
    }).subscribe();
  }

  // Dedupe by row id so an optimistic insert and its realtime echo do not
  // both show.
  void _add(Map<String, dynamic> row) {
    final id = row['\$id']?.toString();
    if (id == null || _seen.contains(id)) return;
    _seen.add(id);
    _messages.add(row);
  }

  Future<void> _send() async {
    final text = _composer.text.trim();
    if (text.isEmpty || _sending) return;
    setState(() => _sending = true);
    final svc = AppladService.instance;
    try {
      final row = await _client.databases.createRow(
        databaseId: Config.databaseId,
        tableId: Config.messagesTable,
        data: {
          'channel_id': widget.channelId,
          'user_id': svc.userId,
          'author_name': svc.userName,
          'body': text,
        },
        // Scope the message to this channel's team: its members, and no one
        // else, can read it. Only the author may edit or delete it.
        permissions: [
          'read("team:${widget.channelId}")',
          'update("user:${svc.userId}")',
          'delete("user:${svc.userId}")',
        ],
      );
      _composer.clear();
      setState(() => _add(row));
      _scrollToBottom();
    } catch (_) {
      _snack('Message failed to send.');
    } finally {
      if (mounted) setState(() => _sending = false);
    }
  }

  Future<void> _invite() async {
    final email = await _promptText('Invite to #${widget.channelName}', 'Their email');
    if (email == null || email.trim().isEmpty) return;
    try {
      final m = await _client.teams
          .createMembership(widget.channelId, email: email.trim());
      final code = '${widget.channelId}:${m['\$id']}:${m['secret']}';
      if (!mounted) return;
      showDialog<void>(
        context: context,
        builder: (ctx) => AlertDialog(
          title: const Text('Invite code'),
          content: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              const Text(
                'Send this code to your teammate. They paste it into '
                '"Join a channel". A self-hosted instance may have no email to '
                'send it for you.',
              ),
              const SizedBox(height: 12),
              SelectableText(code, style: const TextStyle(fontFamily: 'monospace')),
            ],
          ),
          actions: [
            TextButton(
              onPressed: () {
                Clipboard.setData(ClipboardData(text: code));
                Navigator.of(ctx).pop();
                _snack('Invite code copied.');
              },
              child: const Text('Copy'),
            ),
            FilledButton(
              onPressed: () => Navigator.of(ctx).pop(),
              child: const Text('Done'),
            ),
          ],
        ),
      );
    } catch (_) {
      _snack('Could not create an invite.');
    }
  }

  @override
  Widget build(BuildContext context) {
    final me = AppladService.instance.userId;
    return Scaffold(
      appBar: AppBar(
        title: Text('#${widget.channelName}'),
        actions: [
          IconButton(
            tooltip: 'Invite',
            onPressed: _invite,
            icon: const Icon(Icons.person_add_alt),
          ),
        ],
      ),
      body: Column(
        children: [
          Expanded(
            child: _loading
                ? const Center(child: CircularProgressIndicator())
                : _messages.isEmpty
                    ? const Center(child: Text('No messages yet. Say hello.'))
                    : ListView.builder(
                        controller: _scroll,
                        padding: const EdgeInsets.all(12),
                        itemCount: _messages.length,
                        itemBuilder: (_, i) => _bubble(_messages[i], me),
                      ),
          ),
          SafeArea(
            top: false,
            child: Padding(
              padding: const EdgeInsets.fromLTRB(12, 4, 12, 12),
              child: Row(
                children: [
                  Expanded(
                    child: TextField(
                      controller: _composer,
                      minLines: 1,
                      maxLines: 4,
                      textInputAction: TextInputAction.send,
                      onSubmitted: (_) => _send(),
                      decoration: const InputDecoration(
                        hintText: 'Message',
                        border: OutlineInputBorder(),
                        isDense: true,
                      ),
                    ),
                  ),
                  const SizedBox(width: 8),
                  IconButton.filled(
                    onPressed: _sending ? null : _send,
                    icon: const Icon(Icons.send),
                  ),
                ],
              ),
            ),
          ),
        ],
      ),
    );
  }

  Widget _bubble(Map<String, dynamic> m, String me) {
    final mine = m['user_id']?.toString() == me;
    final author = (m['author_name'] as String?) ?? 'Someone';
    final body = (m['body'] as String?) ?? '';
    return Align(
      alignment: mine ? Alignment.centerRight : Alignment.centerLeft,
      child: Container(
        margin: const EdgeInsets.symmetric(vertical: 4),
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
        constraints: BoxConstraints(
          maxWidth: MediaQuery.of(context).size.width * 0.72,
        ),
        decoration: BoxDecoration(
          color: mine ? const Color(0xFF6C47FF) : const Color(0xFF1E1F26),
          borderRadius: BorderRadius.circular(12),
        ),
        child: Column(
          crossAxisAlignment:
              mine ? CrossAxisAlignment.end : CrossAxisAlignment.start,
          children: [
            if (!mine)
              Padding(
                padding: const EdgeInsets.only(bottom: 2),
                child: Text(
                  author,
                  style: const TextStyle(
                      fontSize: 12, color: Colors.white70, fontWeight: FontWeight.w600),
                ),
              ),
            Text(body, style: const TextStyle(color: Colors.white)),
          ],
        ),
      ),
    );
  }

  void _scrollToBottom() {
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (_scroll.hasClients) {
        _scroll.animateTo(
          _scroll.position.maxScrollExtent,
          duration: const Duration(milliseconds: 200),
          curve: Curves.easeOut,
        );
      }
    });
  }

  void _snack(String m) {
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(m)));
  }

  Future<String?> _promptText(String title, String label) {
    final controller = TextEditingController();
    return showDialog<String>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(title),
        content: TextField(
          controller: controller,
          autofocus: true,
          keyboardType: TextInputType.emailAddress,
          decoration: InputDecoration(labelText: label),
          onSubmitted: (v) => Navigator.of(ctx).pop(v),
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(ctx).pop(),
            child: const Text('Cancel'),
          ),
          FilledButton(
            onPressed: () => Navigator.of(ctx).pop(controller.text),
            child: const Text('OK'),
          ),
        ],
      ),
    );
  }
}
