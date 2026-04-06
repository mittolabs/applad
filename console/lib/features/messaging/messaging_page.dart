import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../core/api/client.dart';

class MessagingPage extends ConsumerWidget {
  const MessagingPage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return Scaffold(
      appBar: AppBar(title: const Text('Messaging')),
      body: SingleChildScrollView(
        padding: const EdgeInsets.all(24),
        child: Center(
          child: ConstrainedBox(
            constraints: const BoxConstraints(maxWidth: 600),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text('Send Email',
                    style: Theme.of(context).textTheme.headlineSmall),
                const SizedBox(height: 24),
                _SendEmailForm(),
                const SizedBox(height: 32),
                const Divider(),
                const SizedBox(height: 16),
                Text('SMTP Configuration',
                    style: Theme.of(context).textTheme.titleMedium),
                const SizedBox(height: 8),
                const Text(
                  'Configure SMTP settings via environment variables:\n'
                  'SMTP_HOST, SMTP_PORT, SMTP_USER, SMTP_PASS, SMTP_FROM',
                  style: TextStyle(color: Colors.grey),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

class _SendEmailForm extends ConsumerStatefulWidget {
  @override
  ConsumerState<_SendEmailForm> createState() => _SendEmailFormState();
}

class _SendEmailFormState extends ConsumerState<_SendEmailForm> {
  final _toCtrl = TextEditingController();
  final _subjectCtrl = TextEditingController();
  final _bodyCtrl = TextEditingController();
  bool _sending = false;
  String? _status;

  @override
  void dispose() {
    _toCtrl.dispose();
    _subjectCtrl.dispose();
    _bodyCtrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            TextField(
              controller: _toCtrl,
              decoration: const InputDecoration(
                labelText: 'To',
                hintText: 'user@example.com',
                border: OutlineInputBorder(),
              ),
            ),
            const SizedBox(height: 16),
            TextField(
              controller: _subjectCtrl,
              decoration: const InputDecoration(
                labelText: 'Subject',
                border: OutlineInputBorder(),
              ),
            ),
            const SizedBox(height: 16),
            TextField(
              controller: _bodyCtrl,
              maxLines: 6,
              decoration: const InputDecoration(
                labelText: 'HTML Body',
                border: OutlineInputBorder(),
                alignLabelWithHint: true,
              ),
            ),
            const SizedBox(height: 16),
            Row(
              children: [
                FilledButton.icon(
                  onPressed: _sending ? null : _send,
                  icon: _sending
                      ? const SizedBox(
                          width: 16,
                          height: 16,
                          child: CircularProgressIndicator(strokeWidth: 2),
                        )
                      : const Icon(Icons.send),
                  label: Text(_sending ? 'Sending...' : 'Send Email'),
                ),
                if (_status != null) ...[
                  const SizedBox(width: 16),
                  Chip(
                    label: Text(_status!),
                    backgroundColor: _status == 'Sent'
                        ? Colors.green.shade100
                        : Colors.red.shade100,
                  ),
                ],
              ],
            ),
          ],
        ),
      ),
    );
  }

  Future<void> _send() async {
    if (_toCtrl.text.isEmpty || _subjectCtrl.text.isEmpty) {
      setState(() => _status = 'To and Subject required');
      return;
    }
    setState(() {
      _sending = true;
      _status = null;
    });
    try {
      final api = ref.read(apiClientProvider);
      await api.post('/messaging/email', data: {
        'to': _toCtrl.text.split(',').map((e) => e.trim()).toList(),
        'subject': _subjectCtrl.text,
        'html': _bodyCtrl.text,
      });
      setState(() => _status = 'Sent');
      _toCtrl.clear();
      _subjectCtrl.clear();
      _bodyCtrl.clear();
    } catch (e) {
      setState(() => _status = 'Failed: $e');
    } finally {
      setState(() => _sending = false);
    }
  }
}
