import 'package:flutter/material.dart';

import '../applad_service.dart';
import '../chat_model.dart';
import 'chat_screen.dart';
import 'login_screen.dart';

/// The channels a user belongs to. A channel is an Applad Team (audit G2/G3/G4):
/// creating one enrols you as owner, and `teams.list()` returns only the teams
/// you have joined (G6), so this is genuinely "your channels", not everyone's.
class ChannelsScreen extends StatefulWidget {
  const ChannelsScreen({super.key});

  @override
  State<ChannelsScreen> createState() => _ChannelsScreenState();
}

class _ChannelsScreenState extends State<ChannelsScreen> {
  List<Map<String, dynamic>> _channels = [];
  bool _loading = true;
  String? _error;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final res = await AppladService.instance.client.teams.list();
      final teams = (res['teams'] as List? ?? [])
          .map((e) => Map<String, dynamic>.from(e as Map))
          .toList();
      setState(() => _channels = teams);
    } catch (e) {
      setState(() => _error = 'Could not load channels.');
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  Future<void> _createChannel() async {
    final name = await _promptText('New channel', 'Channel name');
    if (name == null || name.trim().isEmpty) return;
    try {
      await AppladService.instance.client.teams.create(name: name.trim());
      await _load();
    } catch (_) {
      _snack('Could not create the channel.');
    }
  }

  Future<void> _joinChannel() async {
    final token = await _promptText(
      'Join a channel',
      'Paste an invite code',
      hint: 'teamId:membershipId:secret',
    );
    if (token == null) return;
    final invite = InviteCode.tryParse(token);
    if (invite == null) {
      _snack('That invite code does not look right.');
      return;
    }
    try {
      await AppladService.instance.client.teams
          .acceptMembership(invite.teamId, invite.membershipId, invite.secret);
      await _load();
      _snack('Joined.');
    } catch (_) {
      _snack('That invite is invalid or already used.');
    }
  }

  Future<void> _signOut() async {
    await AppladService.instance.signOut();
    if (!mounted) return;
    Navigator.of(context).pushReplacement(
      MaterialPageRoute(builder: (_) => const LoginScreen()),
    );
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Channels'),
        actions: [
          IconButton(
            tooltip: 'Refresh',
            onPressed: _load,
            icon: const Icon(Icons.refresh),
          ),
          IconButton(
            tooltip: 'Join a channel',
            onPressed: _joinChannel,
            icon: const Icon(Icons.group_add_outlined),
          ),
          PopupMenuButton<String>(
            onSelected: (v) {
              if (v == 'signout') _signOut();
            },
            itemBuilder: (_) => [
              PopupMenuItem(
                enabled: false,
                child: Text(AppladService.instance.userName),
              ),
              const PopupMenuItem(value: 'signout', child: Text('Sign out')),
            ],
          ),
        ],
      ),
      floatingActionButton: FloatingActionButton.extended(
        onPressed: _createChannel,
        icon: const Icon(Icons.add),
        label: const Text('Channel'),
      ),
      body: RefreshIndicator(
        onRefresh: _load,
        child: _buildBody(),
      ),
    );
  }

  Widget _buildBody() {
    if (_loading) {
      return const Center(child: CircularProgressIndicator());
    }
    if (_error != null) {
      return _centeredList(_error!);
    }
    if (_channels.isEmpty) {
      return _centeredList(
        'No channels yet.\nCreate one, or join with an invite code.',
      );
    }
    return ListView.separated(
      itemCount: _channels.length,
      separatorBuilder: (context, index) => const Divider(height: 1),
      itemBuilder: (_, i) {
        final c = _channels[i];
        final name = (c['name'] as String?) ?? 'Channel';
        return ListTile(
          leading: CircleAvatar(
            child: Text(name.isNotEmpty ? name[0].toUpperCase() : '#'),
          ),
          title: Text(name),
          trailing: const Icon(Icons.chevron_right),
          onTap: () => Navigator.of(context).push(
            MaterialPageRoute(
              builder: (_) => ChatScreen(
                channelId: c['\$id'].toString(),
                channelName: name,
              ),
            ),
          ),
        );
      },
    );
  }

  // Scrollable so RefreshIndicator works even when there is nothing to show.
  Widget _centeredList(String message) {
    return ListView(
      children: [
        Padding(
          padding: const EdgeInsets.only(top: 120),
          child: Text(message, textAlign: TextAlign.center),
        ),
      ],
    );
  }

  void _snack(String m) {
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(m)));
  }

  Future<String?> _promptText(String title, String label, {String? hint}) {
    final controller = TextEditingController();
    return showDialog<String>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(title),
        content: TextField(
          controller: controller,
          autofocus: true,
          decoration: InputDecoration(labelText: label, hintText: hint),
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
