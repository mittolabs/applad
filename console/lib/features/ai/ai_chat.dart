// Applad AI — global floating chat overlay.
//
// Two modes:
//   1. Compact — Intercom Fin-style floating panel
//   2. Expanded — full-screen developer assistant workspace
//
// Injected in MaterialApp.router's builder — above the navigator.
// Rules to avoid crashes in that context:
//   • No Tooltip (needs Overlay from navigator)
//   • GoRouter wrapped in try/catch
//   • No InkWell (use MouseRegion + GestureDetector)
//   • No Material(color: transparent) on panel root (hover bleed)

import 'dart:async';
import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:http/http.dart' as http;
import 'package:lucide_icons/lucide_icons.dart';
import '../../core/api/client.dart';
import '../../core/providers/auth_provider.dart';

// ── Colors ────────────────────────────────────────────────────────────────────

const _panelBg      = Color(0xFF1A1B1E);
const _headerBg     = Color(0xFF1E1F23);
const _msgBg        = Color(0xFF25262A);
const _inputBg      = Color(0xFF25262A);
const _codeBg       = Color(0xFF0D0E10);
const _expandedBg   = Color(0xFF0F1011);
const _sidebarBg    = Color(0xFF131416);
const _aiAccent     = Color(0xFF3472A4);
const _aiAccentDim  = Color(0xFF1D5A8A);
const _divider      = Color(0x14FFFFFF);
const _textPri      = Colors.white;
const _textSec      = Color(0xFFAAAAAA);
const _textMuted    = Color(0xFF606068);

// ── Sizes ─────────────────────────────────────────────────────────────────────

const _bubbleSize = 52.0;
const _panelW     = 380.0;
const _panelH     = 520.0;

// ── Message model ─────────────────────────────────────────────────────────────

class _Msg {
  final String role;
  final String text;
  final DateTime time;
  _Msg(this.role, this.text) : time = DateTime.now();
  _Msg.withTime(this.role, this.text, this.time);
}

// ── Quick actions (sidebar shortcuts) ────────────────────────────────────────

class _QuickAction {
  final IconData icon;
  final String label;
  final String prompt;
  const _QuickAction(this.icon, this.label, this.prompt);
}

const _quickActions = [
  _QuickAction(LucideIcons.database,    'Create a table',        'Help me create a new database table with proper columns and types.'),
  _QuickAction(LucideIcons.code2,       'Write a function',      'Help me write and deploy a serverless function.'),
  _QuickAction(LucideIcons.users,       'Set up OAuth',          'How do I configure OAuth (Google, GitHub, Apple) for my project?'),
  _QuickAction(LucideIcons.hardDrive,   'Create a bucket',       'How do I create a storage bucket and handle file uploads?'),
  _QuickAction(LucideIcons.rocket,      'Deploy a container',    'Walk me through deploying a Docker container with Applad.'),
  _QuickAction(LucideIcons.zap,         'Configure webhooks',    'How do I set up webhooks to react to database changes?'),
  _QuickAction(LucideIcons.mail,        'Send emails',           'How do I send transactional emails from my project?'),
  _QuickAction(LucideIcons.workflow,    'Build a workflow',      'Help me create an automated workflow using the DAG engine.'),
];

// ── AI config provider ────────────────────────────────────────────────────────

final _aiConfigProvider = FutureProvider.autoDispose<Map<String, dynamic>>((ref) async {
  final api   = ref.read(apiClientProvider);
  final token = ref.read(consoleTokenProvider);
  if (token == null) return {};
  api.setAuthToken(token);
  try {
    final res = await api.get('/ai/config');
    return Map<String, dynamic>.from(res.data as Map? ?? {});
  } catch (_) {
    return {};
  }
});

// ── Bubble position ───────────────────────────────────────────────────────────

final _bubblePosProvider = StateProvider<Offset>(
  (ref) => const Offset(0, 0),
);

// ── Root overlay ──────────────────────────────────────────────────────────────

class AiChatOverlay extends ConsumerStatefulWidget {
  final Widget child;
  const AiChatOverlay({super.key, required this.child});

  @override
  ConsumerState<AiChatOverlay> createState() => _AiChatOverlayState();
}

// A saved conversation snapshot shown in the sidebar
class _Session {
  final String title;
  final List<_Msg> messages;
  _Session(this.title, this.messages);
}

class _AiChatOverlayState extends ConsumerState<AiChatOverlay> {
  bool _open      = false;
  bool _expanded  = false;
  bool _posInit   = false;
  bool _loading   = false;   // waiting for first token (shows thinking bubble)
  bool _streaming = false;   // tokens arriving (prevents re-send)

  final List<_Msg>     _messages = [];
  final List<_Session> _sessions = [];
  final _inputCtrl  = TextEditingController();
  final _scrollCtrl = ScrollController();

  @override
  void dispose() {
    _inputCtrl.dispose();
    _scrollCtrl.dispose();
    super.dispose();
  }

  void _initPos(Size screen) {
    if (_posInit) return;
    _posInit = true;
    WidgetsBinding.instance.addPostFrameCallback((_) {
      ref.read(_bubblePosProvider.notifier).state = Offset(
        screen.width  - _bubbleSize - 20,
        screen.height - _bubbleSize - 28,
      );
    });
  }

  Future<void> _send() async {
    final text = _inputCtrl.text.trim();
    if (text.isEmpty || _loading || _streaming) return;
    _inputCtrl.clear();

    setState(() {
      _messages.add(_Msg('user', text));
      _loading = true;
      _streaming = true;
    });
    _scrollToBottom();

    try {
      final token = ref.read(consoleTokenProvider);
      if (token == null) throw Exception('Not authenticated');

      String ctx = '';
      try {
        final path = GoRouter.of(context)
            .routerDelegate.currentConfiguration.uri.path;
        ctx = _routeLabel(path);
      } catch (_) {}

      final payload = jsonEncode({
        'messages': _messages
            .map((m) => {'role': m.role, 'content': m.text})
            .toList(),
        'context': ctx,
      });

      // Build absolute URL — package:http requires one.
      final rawBase = ref.read(apiClientProvider).baseUrl;
      final streamUri = rawBase.startsWith('http')
          ? Uri.parse('$rawBase/ai/stream')
          : Uri.base.resolve('$rawBase/ai/stream');

      final request = http.Request('POST', streamUri)
        ..headers['Content-Type']  = 'application/json'
        ..headers['Authorization'] = 'Bearer $token'
        ..body = payload;

      final httpClient = http.Client();
      try {
        final response = await httpClient.send(request);
        String leftover = '';
        bool   gotFirst = false;
        DateTime? firstTime;

        await for (final bytes in response.stream) {
          if (!mounted) break;
          final chunk    = utf8.decode(bytes);
          final combined = leftover + chunk;
          final lines    = combined.split('\n');
          leftover       = lines.removeLast();

          for (final line in lines) {
            if (!line.startsWith('data: ')) continue;
            final data = line.substring(6).trim();
            if (data == '[DONE]') break;
            try {
              final ev = jsonDecode(data) as Map;
              if (ev.containsKey('error')) {
                final errMsg = ev['error'] as String;
                setState(() {
                  _loading = false;
                  _messages.add(_Msg('assistant', errMsg));
                  gotFirst = true;
                });
                _scrollToBottom();
              } else if (ev.containsKey('delta')) {
                final delta = ev['delta'] as String;
                setState(() {
                  if (!gotFirst) {
                    // First token — switch from thinking to streaming text.
                    gotFirst  = true;
                    firstTime = DateTime.now();
                    _loading  = false;
                    _messages.add(_Msg.withTime('assistant', delta, firstTime!));
                  } else {
                    final prev = _messages.last;
                    _messages[_messages.length - 1] =
                        _Msg.withTime('assistant', prev.text + delta, prev.time);
                  }
                });
                _scrollToBottom();
              }
            } catch (_) {}
          }
        }

        // If we never got a token (e.g. server error), add error message.
        if (!gotFirst && mounted) {
          setState(() {
            _messages.add(_Msg('assistant', 'Sorry, I ran into an issue. Please try again.'));
          });
        }
      } finally {
        httpClient.close();
      }
    } catch (e) {
      if (mounted) {
        setState(() {
          _messages.add(_Msg('assistant', 'Sorry, I ran into an issue. Please try again.'));
        });
      }
    } finally {
      if (mounted) {
        setState(() {
          _loading   = false;
          _streaming = false;
        });
      }
    }
    _scrollToBottom();
  }

  void _newChat() {
    if (_messages.isNotEmpty) {
      final title = _messages
          .firstWhere((m) => m.role == 'user',
              orElse: () => _Msg('user', 'Chat'))
          .text;
      _sessions.insert(
          0,
          _Session(
              title.length > 40 ? '${title.substring(0, 40)}…' : title,
              List.from(_messages)));
    }
    setState(() => _messages.clear());
  }

  void _fillPrompt(String text) {
    _inputCtrl.text = text;
    _inputCtrl.selection = TextSelection.fromPosition(
      TextPosition(offset: text.length),
    );
  }

  void _scrollToBottom() {
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (_scrollCtrl.hasClients) {
        _scrollCtrl.animateTo(
          _scrollCtrl.position.maxScrollExtent,
          duration: const Duration(milliseconds: 200),
          curve: Curves.easeOut,
        );
      }
    });
  }

  String _routeLabel(String path) {
    if (path.contains('/databases'))  return 'Databases page';
    if (path.contains('/functions'))  return 'Functions page';
    if (path.contains('/storage'))    return 'Storage page';
    if (path.contains('/auth'))       return 'Auth page';
    if (path.contains('/deploy') || path.contains('/sites') ||
        path.contains('/containers') || path.contains('/mobile') ||
        path.contains('/desktop'))    return 'Deploy page';
    if (path.contains('/messaging'))  return 'Messaging page';
    if (path.contains('/workflows'))  return 'Workflows page';
    if (path.contains('/settings'))   return 'Project settings';
    if (path.contains('/overview'))   return 'Project overview';
    return '';
  }

  @override
  Widget build(BuildContext context) {
    final screen = MediaQuery.of(context).size;
    _initPos(screen);
    final pos = ref.watch(_bubblePosProvider);

    // Only show when logged in and not on the login page.
    final token = ref.watch(consoleTokenProvider);
    if (token == null) return widget.child;
    try {
      final path = GoRouter.of(context)
          .routerDelegate.currentConfiguration.uri.path;
      if (path == '/login') return widget.child;
    } catch (_) {}

    // Hide when AI is not configured on the backend.
    final aiConfig    = ref.watch(_aiConfigProvider);
    final configured  = aiConfig.whenOrNull(data: (d) => d['configured'] as bool?) ?? false;
    if (!configured) return widget.child;

    return Stack(
      children: [
        widget.child,

        // ── Full-screen expanded workspace ──────────────────────────────────
        if (_expanded)
          Positioned.fill(
            child: _ExpandedWorkspace(
              messages:      _messages,
              sessions:      _sessions,
              loading:       _loading,
              streaming:     _streaming,
              inputCtrl:     _inputCtrl,
              scrollCtrl:    _scrollCtrl,
              modelName:     aiConfig.whenOrNull(data: (d) => d['model'] as String?) ?? '',
              onSend:        _send,
              onFillPrompt:  _fillPrompt,
              onCollapse: () => setState(() {
                _expanded = false;
                _open = true;
              }),
              onClose: () => setState(() {
                _expanded = false;
                _open = false;
              }),
              onClear: _newChat,
              onLoadSession: (s) => setState(() {
                _messages
                  ..clear()
                  ..addAll(s.messages);
              }),
            ),
          ),

        // ── Compact panel ───────────────────────────────────────────────────
        if (_open && !_expanded)
          _ChatPanel(
            pos:       pos,
            screen:    screen,
            messages:  _messages,
            loading:   _loading,
            streaming: _streaming,
            inputCtrl: _inputCtrl,
            scrollCtrl: _scrollCtrl,
            onSend:    _send,
            onExpand: () => setState(() {
              _open = false;
              _expanded = true;
            }),
            onClose:  () => setState(() => _open = false),
            onClear:  _newChat,
          ),

        // ── Draggable bubble ────────────────────────────────────────────────
        if (!_expanded)
          _DraggableBubble(
            pos:       pos,
            screen:    screen,
            open:      _open,
            hasUnread: _messages.isNotEmpty && !_open,
            onTap:     () => setState(() => _open = !_open),
            onDragUpdate: (delta) {
              final cur = ref.read(_bubblePosProvider);
              final next = Offset(
                (cur.dx + delta.dx).clamp(0.0, screen.width  - _bubbleSize),
                (cur.dy + delta.dy).clamp(0.0, screen.height - _bubbleSize),
              );
              ref.read(_bubblePosProvider.notifier).state = next;
            },
          ),
      ],
    );
  }
}

// ── Draggable bubble ──────────────────────────────────────────────────────────

class _DraggableBubble extends StatefulWidget {
  final Offset pos;
  final Size   screen;
  final bool   open;
  final bool   hasUnread;
  final VoidCallback         onTap;
  final ValueChanged<Offset> onDragUpdate;

  const _DraggableBubble({
    required this.pos,
    required this.screen,
    required this.open,
    required this.hasUnread,
    required this.onTap,
    required this.onDragUpdate,
  });

  @override
  State<_DraggableBubble> createState() => _DraggableBubbleState();
}

class _DraggableBubbleState extends State<_DraggableBubble>
    with SingleTickerProviderStateMixin {
  late final AnimationController _pulse;

  @override
  void initState() {
    super.initState();
    _pulse = AnimationController(
      vsync: this,
      duration: const Duration(seconds: 2),
    )..repeat(reverse: true);
  }

  @override
  void dispose() {
    _pulse.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Positioned(
      left: widget.pos.dx,
      top:  widget.pos.dy,
      child: GestureDetector(
        onPanUpdate: (d) => widget.onDragUpdate(d.delta),
        onTap: widget.onTap,
        child: AnimatedBuilder(
          animation: _pulse,
          builder: (context, _) {
            final glow = widget.open ? 0.35 : (0.12 + _pulse.value * 0.08);
            return Container(
              width:  _bubbleSize,
              height: _bubbleSize,
              decoration: BoxDecoration(
                shape: BoxShape.circle,
                color: Colors.white,
                boxShadow: [
                  BoxShadow(
                    color:      Colors.black.withValues(alpha: glow),
                    blurRadius: 20,
                    spreadRadius: 2,
                  ),
                ],
              ),
              child: Stack(
                alignment: Alignment.center,
                children: [
                  widget.open
                      ? const Icon(LucideIcons.x, color: Color(0xFF444444), size: 22)
                      : ClipOval(
                          child: Image.asset(
                            'assets/applad-mascot-head.png',
                            width:  _bubbleSize,
                            height: _bubbleSize,
                            fit: BoxFit.cover,
                          ),
                        ),
                  if (widget.hasUnread)
                    Positioned(
                      top: 8, right: 8,
                      child: Container(
                        width: 9, height: 9,
                        decoration: const BoxDecoration(
                          color: Color(0xFF10B981),
                          shape: BoxShape.circle,
                        ),
                      ),
                    ),
                ],
              ),
            );
          },
        ),
      ),
    );
  }
}

// ── Compact panel ─────────────────────────────────────────────────────────────

class _ChatPanel extends StatelessWidget {
  final Offset pos;
  final Size   screen;
  final List<_Msg> messages;
  final bool   loading;
  final bool   streaming;
  final TextEditingController inputCtrl;
  final ScrollController      scrollCtrl;
  final VoidCallback onSend;
  final VoidCallback onExpand;
  final VoidCallback onClose;
  final VoidCallback onClear;

  const _ChatPanel({
    required this.pos,
    required this.screen,
    required this.messages,
    required this.loading,
    required this.streaming,
    required this.inputCtrl,
    required this.scrollCtrl,
    required this.onSend,
    required this.onExpand,
    required this.onClose,
    required this.onClear,
  });

  Offset _panelPos() {
    double x = pos.dx - _panelW - 12;
    if (x < 12) x = pos.dx + _bubbleSize + 12;
    x = x.clamp(12.0, (screen.width - _panelW - 12).clamp(12.0, double.infinity));

    final availH = screen.height - 72;
    final h = _panelH.clamp(0.0, availH);
    double y = (screen.height - h - 12).clamp(60.0, double.infinity);

    return Offset(x, y);
  }

  @override
  Widget build(BuildContext context) {
    final p      = _panelPos();
    final availH = screen.height - 72;
    final panelH = _panelH.clamp(0.0, availH);

    return Positioned(
      left: p.dx,
      top:  p.dy,
      child: Material(
        color: Colors.transparent,
        child: Container(
          width:  _panelW,
          height: panelH,
          decoration: BoxDecoration(
            color: _panelBg,
            borderRadius: BorderRadius.circular(16),
            border: Border.all(color: Colors.white.withValues(alpha: 0.08)),
            boxShadow: [
              BoxShadow(
                color: Colors.black.withValues(alpha: 0.55),
                blurRadius: 48,
                offset: const Offset(0, 16),
              ),
            ],
          ),
          child: ClipRRect(
            borderRadius: BorderRadius.circular(16),
            child: Column(
              children: [
                _FinHeader(onExpand: onExpand, onClose: onClose, onClear: onClear),
                const Divider(height: 1, color: _divider),
                Expanded(
                  child: _MessageList(
                    messages:  messages,
                    loading:   loading,
                    scrollCtrl: scrollCtrl,
                    compact:   true,
                  ),
                ),
                const Divider(height: 1, color: _divider),
                _FinInput(ctrl: inputCtrl, onSend: onSend, loading: loading || streaming),
                _FinFooter(),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

// ── Header (compact) ──────────────────────────────────────────────────────────

class _FinHeader extends StatelessWidget {
  final VoidCallback onExpand;
  final VoidCallback onClose;
  final VoidCallback onClear;
  const _FinHeader({required this.onExpand, required this.onClose, required this.onClear});

  @override
  Widget build(BuildContext context) {
    return Container(
      color: _headerBg,
      padding: const EdgeInsets.fromLTRB(16, 14, 12, 14),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.center,
        children: [
          ClipOval(
            child: Image.asset(
              'assets/applad-mascot-head.png',
              width: 34, height: 34, fit: BoxFit.cover,
            ),
          ),
          const SizedBox(width: 12),
          const Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              mainAxisSize: MainAxisSize.min,
              children: [
                Text('Applad AI',
                    style: TextStyle(
                        color: _textPri, fontSize: 14,
                        fontWeight: FontWeight.w600, height: 1.2)),
                SizedBox(height: 2),
                Text('Ask anything about your project',
                    style: TextStyle(color: _textSec, fontSize: 11.5, height: 1.2)),
              ],
            ),
          ),
          _IconBtn(icon: LucideIcons.maximize2, onTap: onExpand),
          const SizedBox(width: 2),
          _IconBtn(icon: LucideIcons.x, onTap: onClose),
        ],
      ),
    );
  }
}

// ── Full-screen expanded workspace ────────────────────────────────────────────

class _DotGridPainter extends CustomPainter {
  @override
  void paint(Canvas canvas, Size size) {
    final paint = Paint()
      ..color = Colors.white.withValues(alpha: 0.035)
      ..style = PaintingStyle.fill;
    const spacing = 28.0;
    for (double x = 0; x < size.width; x += spacing) {
      for (double y = 0; y < size.height; y += spacing) {
        canvas.drawCircle(Offset(x, y), 1.1, paint);
      }
    }
  }

  @override
  bool shouldRepaint(_DotGridPainter _) => false;
}

class _ExpandedWorkspace extends StatelessWidget {
  final List<_Msg>     messages;
  final List<_Session> sessions;
  final bool           loading;
  final bool           streaming;
  final String         modelName;
  final TextEditingController inputCtrl;
  final ScrollController      scrollCtrl;
  final VoidCallback           onSend;
  final ValueChanged<String>   onFillPrompt;
  final VoidCallback           onCollapse;
  final VoidCallback           onClose;
  final VoidCallback           onClear;
  final ValueChanged<_Session> onLoadSession;

  const _ExpandedWorkspace({
    required this.messages,
    required this.sessions,
    required this.loading,
    required this.streaming,
    required this.modelName,
    required this.inputCtrl,
    required this.scrollCtrl,
    required this.onSend,
    required this.onFillPrompt,
    required this.onCollapse,
    required this.onClose,
    required this.onClear,
    required this.onLoadSession,
  });

  @override
  Widget build(BuildContext context) {
    return ColoredBox(
      color: _expandedBg,
      child: Row(
        children: [
          _ExpandedSidebar(
            sessions:      sessions,
            onNewChat:     onClear,
            onLoadSession: onLoadSession,
          ),
          Container(width: 1, color: _divider),
          Expanded(
            child: CustomPaint(
              painter: _DotGridPainter(),
              child: Column(
                children: [
                  _ExpandedTopBar(onCollapse: onCollapse, onClose: onClose, onClear: onClear),
                  Expanded(
                    child: messages.isEmpty
                        ? const SizedBox.shrink()
                        : _ExpandedMessageList(
                            messages:   messages,
                            loading:    loading,
                            scrollCtrl: scrollCtrl,
                          ),
                  ),
                  _ExpandedBottom(
                    ctrl:            inputCtrl,
                    onSend:          onSend,
                    loading:         loading || streaming,
                    showSuggestions: messages.isEmpty,
                    onFillPrompt:    onFillPrompt,
                    modelName:       modelName,
                  ),
                ],
              ),
            ),
          ),
        ],
      ),
    );
  }
}

// ── Expanded sidebar ──────────────────────────────────────────────────────────

class _ExpandedSidebar extends StatelessWidget {
  final List<_Session>         sessions;
  final VoidCallback           onNewChat;
  final ValueChanged<_Session> onLoadSession;

  const _ExpandedSidebar({
    required this.sessions,
    required this.onNewChat,
    required this.onLoadSession,
  });

  @override
  Widget build(BuildContext context) {
    return Container(
      width: 210,
      color: _sidebarBg,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 16, 12, 12),
            child: Row(children: [
              ClipOval(
                child: Image.asset(
                  'assets/applad-mascot-head.png',
                  width: 26, height: 26, fit: BoxFit.cover,
                ),
              ),
              const SizedBox(width: 9),
              const Expanded(
                child: Text('Applad AI',
                    style: TextStyle(
                        color: _textPri, fontSize: 13,
                        fontWeight: FontWeight.w600,
                        decoration: TextDecoration.none)),
              ),
              _IconBtn(icon: LucideIcons.plus, onTap: onNewChat),
            ]),
          ),
          Container(height: 1, color: _divider),
          const SizedBox(height: 8),
          Expanded(
            child: sessions.isEmpty
                ? Padding(
                    padding: const EdgeInsets.fromLTRB(16, 8, 16, 0),
                    child: Text(
                      'Previous conversations will appear here.',
                      style: TextStyle(
                          color: _textMuted, fontSize: 12,
                          height: 1.55, decoration: TextDecoration.none),
                    ),
                  )
                : ListView.builder(
                    padding: const EdgeInsets.symmetric(horizontal: 4),
                    itemCount: sessions.length,
                    itemBuilder: (ctx, i) => _SessionRow(
                      session: sessions[i],
                      onTap:   () => onLoadSession(sessions[i]),
                    ),
                  ),
          ),
        ],
      ),
    );
  }
}

class _SessionRow extends StatefulWidget {
  final _Session session;
  final VoidCallback onTap;
  const _SessionRow({required this.session, required this.onTap});

  @override
  State<_SessionRow> createState() => _SessionRowState();
}

class _SessionRowState extends State<_SessionRow> {
  bool _hovered = false;

  @override
  Widget build(BuildContext context) {
    return MouseRegion(
      cursor: SystemMouseCursors.click,
      onEnter: (_) => setState(() => _hovered = true),
      onExit:  (_) => setState(() => _hovered = false),
      child: GestureDetector(
        onTap: widget.onTap,
        child: AnimatedContainer(
          duration: const Duration(milliseconds: 100),
          margin:  const EdgeInsets.only(bottom: 1),
          padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 9),
          decoration: BoxDecoration(
            color: _hovered
                ? Colors.white.withValues(alpha: 0.06)
                : Colors.transparent,
            borderRadius: BorderRadius.circular(8),
          ),
          child: Row(children: [
            Icon(LucideIcons.messageSquare,
                size: 13,
                color: _hovered ? _textSec : _textMuted),
            const SizedBox(width: 9),
            Expanded(
              child: Text(
                widget.session.title,
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
                style: TextStyle(
                    color: _hovered ? _textPri : _textSec,
                    fontSize: 12.5,
                    decoration: TextDecoration.none),
              ),
            ),
          ]),
        ),
      ),
    );
  }
}

class _SidebarAction extends StatefulWidget {
  final _QuickAction action;
  final VoidCallback onTap;
  const _SidebarAction({required this.action, required this.onTap});

  @override
  State<_SidebarAction> createState() => _SidebarActionState();
}

class _SidebarActionState extends State<_SidebarAction> {
  bool _hovered = false;

  @override
  Widget build(BuildContext context) {
    return MouseRegion(
      cursor: SystemMouseCursors.click,
      onEnter: (_) => setState(() => _hovered = true),
      onExit:  (_) => setState(() => _hovered = false),
      child: GestureDetector(
        onTap: widget.onTap,
        child: AnimatedContainer(
          duration: const Duration(milliseconds: 100),
          margin:  const EdgeInsets.only(bottom: 2),
          padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 9),
          decoration: BoxDecoration(
            color: _hovered
                ? Colors.white.withValues(alpha: 0.06)
                : Colors.transparent,
            borderRadius: BorderRadius.circular(8),
          ),
          child: Row(children: [
            Icon(widget.action.icon, size: 14,
                color: _hovered ? _aiAccent : _textMuted),
            const SizedBox(width: 10),
            Expanded(
              child: Text(
                widget.action.label,
                style: TextStyle(
                    color: _hovered ? _textPri : _textSec,
                    fontSize: 13,
                    decoration: TextDecoration.none),
              ),
            ),
          ]),
        ),
      ),
    );
  }
}

// ── Expanded message list ─────────────────────────────────────────────────────

class _ExpandedMessageList extends StatelessWidget {
  final List<_Msg>      messages;
  final bool            loading;
  final ScrollController scrollCtrl;

  const _ExpandedMessageList({
    required this.messages,
    required this.loading,
    required this.scrollCtrl,
  });

  @override
  Widget build(BuildContext context) {
    final count = messages.length + (loading ? 1 : 0);
    return ListView.builder(
      controller: scrollCtrl,
      padding: const EdgeInsets.fromLTRB(0, 16, 0, 8),
      itemCount: count,
      itemBuilder: (ctx, i) {
        if (i >= messages.length) return const _ExpandedThinking();
        final m = messages[i];
        return m.role == 'user'
            ? _ExpandedUserMsg(msg: m)
            : _ExpandedAiMsg(msg: m);
      },
    );
  }
}

class _ExpandedAiMsg extends StatelessWidget {
  final _Msg msg;
  const _ExpandedAiMsg({required this.msg});

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(32, 0, 60, 20),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Container(
            margin: const EdgeInsets.only(top: 1, right: 12),
            child: ClipOval(
              child: Image.asset(
                'assets/applad-mascot-head.png',
                width: 22, height: 22, fit: BoxFit.cover,
              ),
            ),
          ),
          Expanded(child: _AssistantBubble(text: msg.text, compact: false)),
        ],
      ),
    );
  }
}

class _ExpandedUserMsg extends StatelessWidget {
  final _Msg msg;
  const _ExpandedUserMsg({required this.msg});

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(60, 0, 32, 20),
      child: Align(
        alignment: Alignment.centerRight,
        child: Container(
          padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
          decoration: BoxDecoration(
            color: Colors.white.withValues(alpha: 0.06),
            borderRadius: BorderRadius.circular(12),
            border: Border.all(color: Colors.white.withValues(alpha: 0.08)),
          ),
          child: SelectableText(
            msg.text,
            style: const TextStyle(
                color: _textPri, fontSize: 14, height: 1.55,
                decoration: TextDecoration.none),
          ),
        ),
      ),
    );
  }
}

class _ExpandedThinking extends StatefulWidget {
  const _ExpandedThinking();

  @override
  State<_ExpandedThinking> createState() => _ExpandedThinkingState();
}

class _ExpandedThinkingState extends State<_ExpandedThinking>
    with SingleTickerProviderStateMixin {
  late final AnimationController _ctrl;

  @override
  void initState() {
    super.initState();
    _ctrl = AnimationController(
      vsync: this,
      duration: const Duration(milliseconds: 700),
    )..repeat(reverse: true);
  }

  @override
  void dispose() {
    _ctrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(32, 0, 60, 20),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.center,
        children: [
          Container(
            margin: const EdgeInsets.only(right: 12),
            child: ClipOval(
              child: Image.asset(
                'assets/applad-mascot-head.png',
                width: 22, height: 22, fit: BoxFit.cover,
              ),
            ),
          ),
          AnimatedBuilder(
            animation: _ctrl,
            builder: (ctx, _) => Row(
              mainAxisSize: MainAxisSize.min,
              children: List.generate(3, (i) {
                final v = (_ctrl.value - i * 0.25).clamp(0.0, 1.0);
                return Container(
                  width: 6, height: 6,
                  margin: const EdgeInsets.symmetric(horizontal: 3),
                  decoration: BoxDecoration(
                    shape: BoxShape.circle,
                    color: _aiAccent.withValues(alpha: 0.3 + v * 0.7),
                  ),
                );
              }),
            ),
          ),
        ],
      ),
    );
  }
}

// ── Expanded top bar ──────────────────────────────────────────────────────────

class _ExpandedTopBar extends StatelessWidget {
  final VoidCallback onCollapse;
  final VoidCallback onClose;
  final VoidCallback onClear;
  const _ExpandedTopBar(
      {required this.onCollapse, required this.onClose, required this.onClear});

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(20, 10, 12, 6),
      child: Row(children: [
        const Spacer(),
        _IconBtn(icon: LucideIcons.minimize2, onTap: onCollapse),
        const SizedBox(width: 2),
        _IconBtn(icon: LucideIcons.x, onTap: onClose),
      ]),
    );
  }
}

// ── Expanded bottom ───────────────────────────────────────────────────────────

class _ExpandedBottom extends StatelessWidget {
  final TextEditingController ctrl;
  final VoidCallback           onSend;
  final bool                   loading;
  final bool                   showSuggestions;
  final ValueChanged<String>   onFillPrompt;
  final String                 modelName;

  const _ExpandedBottom({
    required this.ctrl,
    required this.onSend,
    required this.loading,
    required this.showSuggestions,
    required this.onFillPrompt,
    required this.modelName,
  });

  @override
  Widget build(BuildContext context) {
    return Container(
      color: _expandedBg,
      padding: const EdgeInsets.fromLTRB(24, 0, 24, 28),
      child: Center(
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 580),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            mainAxisSize: MainAxisSize.min,
            children: [
              if (showSuggestions) ...[
                SingleChildScrollView(
                  scrollDirection: Axis.horizontal,
                  child: Row(
                    children: _quickActions
                        .take(4)
                        .map((a) => Padding(
                              padding: const EdgeInsets.only(right: 8),
                              child: _SuggestionChip(
                                icon: a.icon,
                                label: a.label,
                                onTap: () => onFillPrompt(a.prompt),
                              ),
                            ))
                        .toList(),
                  ),
                ),
                const SizedBox(height: 12),
              ],
              Container(
                decoration: BoxDecoration(
                  color: _inputBg,
                  borderRadius: BorderRadius.circular(14),
                  border: Border.all(color: Colors.white.withValues(alpha: 0.1)),
                  boxShadow: [
                    BoxShadow(
                        color: Colors.black.withValues(alpha: 0.3),
                        blurRadius: 20)
                  ],
                ),
                child: Column(
                  mainAxisSize: MainAxisSize.min,
                  crossAxisAlignment: CrossAxisAlignment.stretch,
                  children: [
                    Padding(
                      padding: const EdgeInsets.fromLTRB(16, 14, 16, 2),
                      child: TextField(
                        controller: ctrl,
                        maxLines: 6,
                        minLines: 1,
                        style: const TextStyle(
                            color: _textPri, fontSize: 14.5, height: 1.5),
                        decoration: const InputDecoration(
                          hintText: 'Ask anything about your project...',
                          hintStyle: TextStyle(color: _textMuted, fontSize: 14.5),
                          border: InputBorder.none,
                          enabledBorder: InputBorder.none,
                          focusedBorder: InputBorder.none,
                          filled: false,
                          isDense: true,
                          contentPadding: EdgeInsets.zero,
                        ),
                        onSubmitted: (_) => onSend(),
                        textInputAction: TextInputAction.send,
                      ),
                    ),
                    Padding(
                      padding: const EdgeInsets.fromLTRB(12, 4, 12, 12),
                      child: Row(children: [
                        _ToolbarIcon(icon: LucideIcons.paperclip),
                        const SizedBox(width: 2),
                        _ToolbarIcon(icon: LucideIcons.smile),
                        const SizedBox(width: 2),
                        _ToolbarIcon(icon: LucideIcons.mic),
                        const Spacer(),
                        if (modelName.isNotEmpty)
                          Container(
                            padding: const EdgeInsets.symmetric(
                                horizontal: 10, vertical: 4),
                            decoration: BoxDecoration(
                              color: Colors.white.withValues(alpha: 0.05),
                              borderRadius: BorderRadius.circular(20),
                              border: Border.all(
                                  color: Colors.white.withValues(alpha: 0.08)),
                            ),
                            child: Row(children: [
                              ClipOval(
                                child: Image.asset(
                                  'assets/applad-mascot-head.png',
                                  width: 11, height: 11, fit: BoxFit.cover,
                                ),
                              ),
                              const SizedBox(width: 5),
                              Text(modelName,
                                  style: const TextStyle(
                                      color: _textMuted,
                                      fontSize: 11.5,
                                      decoration: TextDecoration.none)),
                            ]),
                          ),
                        const SizedBox(width: 8),
                        _SendBtn(onSend: onSend, loading: loading),
                      ]),
                    ),
                  ],
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _SuggestionChip extends StatefulWidget {
  final IconData     icon;
  final String       label;
  final VoidCallback onTap;
  const _SuggestionChip(
      {required this.icon, required this.label, required this.onTap});

  @override
  State<_SuggestionChip> createState() => _SuggestionChipState();
}

class _SuggestionChipState extends State<_SuggestionChip> {
  bool _hovered = false;

  @override
  Widget build(BuildContext context) {
    return MouseRegion(
      cursor: SystemMouseCursors.click,
      onEnter: (_) => setState(() => _hovered = true),
      onExit:  (_) => setState(() => _hovered = false),
      child: GestureDetector(
        onTap: widget.onTap,
        child: AnimatedContainer(
          duration: const Duration(milliseconds: 120),
          padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
          decoration: BoxDecoration(
            color: _hovered
                ? _aiAccent.withValues(alpha: 0.12)
                : Colors.white.withValues(alpha: 0.04),
            border: Border.all(
                color: _hovered
                    ? _aiAccent.withValues(alpha: 0.4)
                    : Colors.white.withValues(alpha: 0.1)),
            borderRadius: BorderRadius.circular(24),
          ),
          child: Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              Icon(widget.icon, size: 14,
                  color: _hovered ? _aiAccent : _textSec),
              const SizedBox(width: 8),
              Text(widget.label,
                  style: TextStyle(
                      color: _hovered ? _textPri : _textSec,
                      fontSize: 13,
                      decoration: TextDecoration.none)),
            ],
          ),
        ),
      ),
    );
  }
}

// ── Shared: message list ──────────────────────────────────────────────────────

class _MessageList extends StatelessWidget {
  final List<_Msg>       messages;
  final bool             loading;
  final ScrollController scrollCtrl;
  final bool             compact;

  const _MessageList({
    required this.messages,
    required this.loading,
    required this.scrollCtrl,
    required this.compact,
  });

  @override
  Widget build(BuildContext context) {
    final itemCount = 1 + messages.length + (loading ? 1 : 0);
    return ListView.builder(
      controller: scrollCtrl,
      padding: EdgeInsets.fromLTRB(
          compact ? 16 : 24, compact ? 16 : 24,
          compact ? 16 : 24, 12),
      itemCount: itemCount,
      itemBuilder: (ctx, i) {
        if (i == 0) return _WelcomeBubble(compact: compact);
        final mi = i - 1;
        if (mi >= messages.length) return _ThinkingBubble(compact: compact);
        return _MsgBubble(msg: messages[mi], compact: compact);
      },
    );
  }
}

// ── Welcome bubble ────────────────────────────────────────────────────────────

class _WelcomeBubble extends StatelessWidget {
  final bool compact;
  const _WelcomeBubble({required this.compact});

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: EdgeInsets.only(bottom: compact ? 12 : 16),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Container(
            padding: EdgeInsets.symmetric(
                horizontal: compact ? 12 : 16,
                vertical:   compact ? 10 : 12),
            decoration: BoxDecoration(
              color: _msgBg,
              borderRadius: const BorderRadius.only(
                topLeft:     Radius.circular(4),
                topRight:    Radius.circular(16),
                bottomLeft:  Radius.circular(16),
                bottomRight: Radius.circular(16),
              ),
              border: Border.all(color: Colors.white.withValues(alpha: 0.06)),
            ),
            child: Text(
              'Hi there 👋\n\nI\'m your Applad AI assistant. Ask me anything about your project — databases, functions, auth, storage, deployments, workflows, and more.',
              style: TextStyle(
                  color: _textPri,
                  fontSize: compact ? 13 : 14,
                  height: 1.55),
            ),
          ),
          const SizedBox(height: 5),
          const Text('Applad AI  •  AI Assistant  •  Just now',
              style: TextStyle(color: _textMuted, fontSize: 11)),
          SizedBox(height: compact ? 12 : 20),
        ],
      ),
    );
  }
}

// ── Message bubble ────────────────────────────────────────────────────────────

class _MsgBubble extends StatelessWidget {
  final _Msg msg;
  final bool compact;
  const _MsgBubble({required this.msg, required this.compact});

  String _timeLabel() {
    final diff = DateTime.now().difference(msg.time);
    if (diff.inSeconds < 60) return 'Just now';
    if (diff.inMinutes < 60) return '${diff.inMinutes}m ago';
    return '${diff.inHours}h ago';
  }

  @override
  Widget build(BuildContext context) {
    final isUser = msg.role == 'user';
    return Padding(
      padding: EdgeInsets.only(bottom: compact ? 12 : 16),
      child: Column(
        crossAxisAlignment:
            isUser ? CrossAxisAlignment.end : CrossAxisAlignment.start,
        children: [
          if (isUser)
            _UserBubble(text: msg.text, compact: compact)
          else
            _AssistantBubble(text: msg.text, compact: compact),
          const SizedBox(height: 5),
          Text(
            isUser
                ? 'You  •  ${_timeLabel()}'
                : 'Applad AI  •  AI Assistant  •  ${_timeLabel()}',
            style: const TextStyle(color: _textMuted, fontSize: 11),
          ),
          SizedBox(height: compact ? 4 : 8),
        ],
      ),
    );
  }
}

class _UserBubble extends StatelessWidget {
  final String text;
  final bool   compact;
  const _UserBubble({required this.text, required this.compact});

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: EdgeInsets.symmetric(
          horizontal: compact ? 12 : 14, vertical: compact ? 8 : 10),
      decoration: BoxDecoration(
        color: _aiAccent.withValues(alpha: 0.22),
        borderRadius: const BorderRadius.only(
          topLeft:     Radius.circular(16),
          topRight:    Radius.circular(4),
          bottomLeft:  Radius.circular(16),
          bottomRight: Radius.circular(16),
        ),
        border: Border.all(color: _aiAccent.withValues(alpha: 0.3)),
      ),
      child: SelectableText(
        text,
        style: TextStyle(
            color: _textPri,
            fontSize: compact ? 13 : 14,
            height: 1.55),
      ),
    );
  }
}

class _AssistantBubble extends StatelessWidget {
  final String text;
  final bool   compact;
  const _AssistantBubble({required this.text, required this.compact});

  @override
  Widget build(BuildContext context) {
    final parts = _splitCodeBlocks(text);
    return Container(
      padding: EdgeInsets.symmetric(
          horizontal: compact ? 12 : 16, vertical: compact ? 8 : 12),
      decoration: BoxDecoration(
        color: _msgBg,
        borderRadius: const BorderRadius.only(
          topLeft:     Radius.circular(4),
          topRight:    Radius.circular(16),
          bottomLeft:  Radius.circular(16),
          bottomRight: Radius.circular(16),
        ),
        border: Border.all(color: Colors.white.withValues(alpha: 0.06)),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: parts.map((p) {
          if (p.isCode) {
            return Container(
              margin:  const EdgeInsets.symmetric(vertical: 6),
              padding: const EdgeInsets.all(12),
              decoration: BoxDecoration(
                color: _codeBg,
                borderRadius: BorderRadius.circular(8),
                border: Border.all(color: Colors.white.withValues(alpha: 0.08)),
              ),
              child: SelectableText(
                p.text,
                style: TextStyle(
                  color: const Color(0xFF7DD3FC),
                  fontFamily: 'monospace',
                  fontSize: compact ? 12 : 13,
                  height: 1.5,
                ),
              ),
            );
          }
          return SelectableText(
            p.text,
            style: TextStyle(
                color: _textPri,
                fontSize: compact ? 13 : 14,
                height: 1.55),
          );
        }).toList(),
      ),
    );
  }

  List<_TextPart> _splitCodeBlocks(String raw) {
    final parts = <_TextPart>[];
    final regex = RegExp(r'```[\w]*\n?([\s\S]*?)```');
    int last = 0;
    for (final m in regex.allMatches(raw)) {
      if (m.start > last) {
        parts.add(_TextPart(raw.substring(last, m.start).trim(), false));
      }
      parts.add(_TextPart(m.group(1)?.trim() ?? '', true));
      last = m.end;
    }
    if (last < raw.length) {
      final tail = raw.substring(last).trim();
      if (tail.isNotEmpty) parts.add(_TextPart(tail, false));
    }
    if (parts.isEmpty) parts.add(_TextPart(raw, false));
    return parts;
  }
}

class _TextPart {
  final String text;
  final bool   isCode;
  const _TextPart(this.text, this.isCode);
}

// ── Thinking bubble ───────────────────────────────────────────────────────────

class _ThinkingBubble extends StatefulWidget {
  final bool compact;
  const _ThinkingBubble({required this.compact});

  @override
  State<_ThinkingBubble> createState() => _ThinkingBubbleState();
}

class _ThinkingBubbleState extends State<_ThinkingBubble>
    with SingleTickerProviderStateMixin {
  late final AnimationController _ctrl;

  @override
  void initState() {
    super.initState();
    _ctrl = AnimationController(
      vsync: this,
      duration: const Duration(milliseconds: 700),
    )..repeat(reverse: true);
  }

  @override
  void dispose() {
    _ctrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: EdgeInsets.only(bottom: widget.compact ? 12 : 16),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Container(
            padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 14),
            decoration: BoxDecoration(
              color: _msgBg,
              borderRadius: const BorderRadius.only(
                topLeft:     Radius.circular(4),
                topRight:    Radius.circular(16),
                bottomLeft:  Radius.circular(16),
                bottomRight: Radius.circular(16),
              ),
              border: Border.all(color: Colors.white.withValues(alpha: 0.06)),
            ),
            child: AnimatedBuilder(
              animation: _ctrl,
              builder: (ctx, _) => Row(
                mainAxisSize: MainAxisSize.min,
                children: List.generate(3, (i) {
                  final delay = i * 0.25;
                  final v = (_ctrl.value - delay).clamp(0.0, 1.0);
                  return Container(
                    width: 7, height: 7,
                    margin: const EdgeInsets.symmetric(horizontal: 3),
                    decoration: BoxDecoration(
                      shape: BoxShape.circle,
                      color: _aiAccent.withValues(alpha: 0.3 + v * 0.7),
                    ),
                  );
                }),
              ),
            ),
          ),
          const SizedBox(height: 5),
          const Text('Applad AI  •  Thinking...',
              style: TextStyle(color: _textMuted, fontSize: 11)),
        ],
      ),
    );
  }
}

// ── Compact input bar ─────────────────────────────────────────────────────────

class _FinInput extends StatelessWidget {
  final TextEditingController ctrl;
  final VoidCallback          onSend;
  final bool                  loading;
  const _FinInput({required this.ctrl, required this.onSend, required this.loading});

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(12, 8, 12, 8),
      child: Container(
        decoration: BoxDecoration(
          color: _inputBg,
          borderRadius: BorderRadius.circular(14),
          border: Border.all(color: Colors.white.withValues(alpha: 0.1)),
        ),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Padding(
              padding: const EdgeInsets.fromLTRB(14, 10, 14, 2),
              child: TextField(
                controller: ctrl,
                maxLines: 4,
                minLines: 1,
                style: const TextStyle(
                    color: _textPri, fontSize: 13.5, height: 1.5),
                decoration: const InputDecoration(
                  hintText: 'Ask a question...',
                  hintStyle: TextStyle(color: _textMuted, fontSize: 13.5),
                  border: InputBorder.none,
                  enabledBorder: InputBorder.none,
                  focusedBorder: InputBorder.none,
                  filled: false,
                  isDense: true,
                  contentPadding: EdgeInsets.zero,
                ),
                onSubmitted: (_) => onSend(),
                textInputAction: TextInputAction.send,
              ),
            ),
            Padding(
              padding: const EdgeInsets.fromLTRB(10, 2, 10, 10),
              child: Row(children: [
                _ToolbarIcon(icon: LucideIcons.paperclip),
                const SizedBox(width: 2),
                _ToolbarIcon(icon: LucideIcons.smile),
                const SizedBox(width: 2),
                _ToolbarIcon(icon: LucideIcons.mic),
                const Spacer(),
                _SendBtn(onSend: onSend, loading: loading),
              ]),
            ),
          ],
        ),
      ),
    );
  }
}

// ── Footer ────────────────────────────────────────────────────────────────────

class _FinFooter extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    return Container(
      color: _headerBg,
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
      child: const Row(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          Icon(LucideIcons.messageCircle, size: 11, color: _textMuted),
          SizedBox(width: 5),
          Text('Chat with Applad',
              style: TextStyle(color: _textMuted, fontSize: 11)),
        ],
      ),
    );
  }
}

// ── Icon button ───────────────────────────────────────────────────────────────

class _IconBtn extends StatefulWidget {
  final IconData     icon;
  final VoidCallback onTap;
  const _IconBtn({required this.icon, required this.onTap});

  @override
  State<_IconBtn> createState() => _IconBtnState();
}

class _IconBtnState extends State<_IconBtn> {
  bool _hovered = false;

  @override
  Widget build(BuildContext context) {
    return MouseRegion(
      cursor: SystemMouseCursors.click,
      onEnter: (_) => setState(() => _hovered = true),
      onExit:  (_) => setState(() => _hovered = false),
      child: GestureDetector(
        onTap: widget.onTap,
        child: AnimatedContainer(
          duration: const Duration(milliseconds: 120),
          padding: const EdgeInsets.all(7),
          decoration: BoxDecoration(
            color: _hovered
                ? Colors.white.withValues(alpha: 0.08)
                : Colors.transparent,
            borderRadius: BorderRadius.circular(8),
          ),
          child: Icon(widget.icon,
              size: 16,
              color: _hovered ? _textPri : _textSec),
        ),
      ),
    );
  }
}

// ── Toolbar icon ──────────────────────────────────────────────────────────────

class _ToolbarIcon extends StatefulWidget {
  final IconData icon;
  const _ToolbarIcon({required this.icon});

  @override
  State<_ToolbarIcon> createState() => _ToolbarIconState();
}

class _ToolbarIconState extends State<_ToolbarIcon> {
  bool _hovered = false;

  @override
  Widget build(BuildContext context) {
    return MouseRegion(
      cursor: SystemMouseCursors.click,
      onEnter: (_) => setState(() => _hovered = true),
      onExit:  (_) => setState(() => _hovered = false),
      child: GestureDetector(
        onTap: () {},
        child: AnimatedContainer(
          duration: const Duration(milliseconds: 100),
          padding: const EdgeInsets.all(5),
          decoration: BoxDecoration(
            color: _hovered
                ? Colors.white.withValues(alpha: 0.06)
                : Colors.transparent,
            borderRadius: BorderRadius.circular(6),
          ),
          child: Icon(widget.icon,
              size: 16,
              color: _hovered ? _textSec : _textMuted),
        ),
      ),
    );
  }
}

// ── Send button ───────────────────────────────────────────────────────────────

class _SendBtn extends StatefulWidget {
  final VoidCallback onSend;
  final bool         loading;
  const _SendBtn({required this.onSend, required this.loading});

  @override
  State<_SendBtn> createState() => _SendBtnState();
}

class _SendBtnState extends State<_SendBtn> {
  bool _hovered = false;

  @override
  Widget build(BuildContext context) {
    return MouseRegion(
      cursor: widget.loading
          ? SystemMouseCursors.basic
          : SystemMouseCursors.click,
      onEnter: (_) => setState(() => _hovered = true),
      onExit:  (_) => setState(() => _hovered = false),
      child: GestureDetector(
        onTap: widget.loading ? null : widget.onSend,
        child: AnimatedContainer(
          duration: const Duration(milliseconds: 120),
          width: 32, height: 32,
          decoration: BoxDecoration(
            shape: BoxShape.circle,
            gradient: widget.loading
                ? null
                : LinearGradient(
                    begin: Alignment.topLeft,
                    end:   Alignment.bottomRight,
                    colors: [
                      _hovered ? _aiAccent.withValues(alpha: 0.85) : _aiAccent,
                      _aiAccentDim,
                    ],
                  ),
            color: widget.loading
                ? Colors.white.withValues(alpha: 0.06)
                : null,
            boxShadow: widget.loading || !_hovered
                ? null
                : [BoxShadow(color: _aiAccent.withValues(alpha: 0.4), blurRadius: 12)],
          ),
          child: widget.loading
              ? Padding(
                  padding: const EdgeInsets.all(8),
                  child: CircularProgressIndicator(
                    strokeWidth: 1.5,
                    color: _aiAccent.withValues(alpha: 0.6),
                  ),
                )
              : const Icon(LucideIcons.arrowUp, size: 15, color: Colors.white),
        ),
      ),
    );
  }
}
