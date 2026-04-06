import 'dart:convert';
import 'dart:math';
import 'package:flutter/material.dart';
import 'package:flutter/gestures.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:lucide_icons/lucide_icons.dart';
import '../../core/api/client.dart';
import '../../core/providers/project_provider.dart';
import '../../core/widgets/app_dialog.dart';

// ═════════════════════════════════════════════════════════════════════════════
// Constants & theme
// ═════════════════════════════════════════════════════════════════════════════

const _bg = Color(0xFF0B0B0F);
const _surface = Color(0xFF16171B);
const _surfaceHover = Color(0xFF1C1D22);
const _accent = Color(0xFF3472A4);
const _border = Color(0x14FFFFFF);
const _dimText = Color(0x80FFFFFF);
const _subtleText = Color(0x40FFFFFF);
const _gridDot = Color(0x0AFFFFFF);
const _nodeW = 220.0;
const _nodeH = 72.0;
const _handleR = 6.0;
const _gridSnap = 20.0;
const _green = Color(0xFF10B981);
const _orange = Color(0xFFF59E0B);
const _red = Color(0xFFEF4444);

// ═════════════════════════════════════════════════════════════════════════════
// Providers
// ═════════════════════════════════════════════════════════════════════════════

final workflowsProvider =
    FutureProvider<Map<String, dynamic>>((ref) async {
  // Wait for project to be set before fetching
  final projectId = ref.watch(currentProjectProvider);
  if (projectId == null) return {'workflows': [], 'total': 0};
  final api = ref.read(apiClientProvider);
  final res = await api.get('/workflows');
  return res.data as Map<String, dynamic>;
});

// ═════════════════════════════════════════════════════════════════════════════
// Node type definitions — categorized like n8n
// ═════════════════════════════════════════════════════════════════════════════

class _NodeDef {
  final String type, label, description, category;
  final IconData icon;
  final Color color;
  final int outputs; // number of output handles (IF=2, Switch=3, default=1)
  final int inputs;
  final List<String>? outputLabels;
  const _NodeDef(this.type, this.label, this.description, this.icon, this.color,
      {this.category = 'Core', this.outputs = 1, this.inputs = 1,
      this.outputLabels});
}

const _triggerDef = _NodeDef(
    'trigger', 'Trigger', 'Workflow start point', LucideIcons.play, _green,
    category: 'Triggers', inputs: 0);

// ── Category definitions ──

class _Category {
  final String name, description;
  final IconData icon;
  const _Category(this.name, this.description, this.icon);
}

const _categories = <_Category>[
  _Category('Applad', 'Auth, Databases, Storage, Functions, Messaging',
      LucideIcons.box),
  _Category('AI', 'LLM agents, summarize, transform', LucideIcons.sparkles),
  _Category('Integrations', 'Connect to external services',
      LucideIcons.plug),
  _Category('Data transformation', 'Manipulate, filter or convert data',
      LucideIcons.pencil),
  _Category('Flow', 'Branch, merge or loop the flow', LucideIcons.gitBranch),
  _Category('Core', 'HTTP requests, code, webhooks', LucideIcons.box),
  _Category('Triggers', 'Start your workflow', LucideIcons.zap),
];

// ── All node types ──

final _allNodeDefs = <_NodeDef>[
  // ── Flow ──
  _NodeDef('if_condition', 'IF', 'Branch on condition',
      LucideIcons.gitBranch, Color(0xFFF97316),
      category: 'Flow', outputs: 2, outputLabels: ['true', 'false']),
  _NodeDef('switch', 'Switch', 'Route to multiple branches',
      LucideIcons.split, Color(0xFFF97316),
      category: 'Flow', outputs: 3, outputLabels: ['1', '2', 'default']),
  _NodeDef('merge', 'Merge', 'Combine data from branches',
      LucideIcons.merge, Color(0xFFF97316),
      category: 'Flow', inputs: 2),
  _NodeDef('loop', 'Loop', 'Iterate over items',
      LucideIcons.repeat, Color(0xFFF97316), category: 'Flow'),
  _NodeDef('wait', 'Wait', 'Pause execution',
      LucideIcons.clock, Color(0xFF64748B), category: 'Flow'),
  _NodeDef('no_operation', 'No Operation', 'Pass-through',
      LucideIcons.arrowRight, Color(0xFF64748B), category: 'Flow'),
  _NodeDef('execute_sub_workflow', 'Sub-Workflow', 'Call another workflow',
      LucideIcons.workflow, Color(0xFFF97316), category: 'Flow'),
  _NodeDef('filter', 'Filter', 'Keep items matching a condition',
      LucideIcons.filter, Color(0xFFF97316), category: 'Flow'),

  // ── Core ──
  _NodeDef('http_request', 'HTTP Request', 'Make an API call',
      LucideIcons.globe, Color(0xFF8B5CF6), category: 'Core'),
  _NodeDef('code', 'Code', 'Run template expression',
      LucideIcons.code2, Color(0xFF06B6D4), category: 'Core'),
  _NodeDef('javascript', 'JavaScript', 'Run JS-like code with helpers',
      LucideIcons.braces, Color(0xFFF7DF1E), category: 'Core'),
  _NodeDef('send_email', 'Send Email', 'Send via SMTP',
      LucideIcons.mail, Color(0xFFEC4899), category: 'Core'),
  _NodeDef('set_variable', 'Set Variable', 'Set a context value',
      LucideIcons.variable, Color(0xFFF59E0B), category: 'Core'),
  _NodeDef('delay', 'Delay', 'Wait before continuing',
      LucideIcons.timer, Color(0xFF64748B), category: 'Core'),

  // ── Data transformation ──
  _NodeDef('edit_fields', 'Edit Fields', 'Set multiple fields at once',
      LucideIcons.pencil, Color(0xFF06B6D4), category: 'Data transformation'),
  _NodeDef('aggregate', 'Aggregate', 'Count, sum, avg, min, max',
      LucideIcons.sigma, Color(0xFF06B6D4), category: 'Data transformation'),
  _NodeDef('summarize', 'Summarize', 'Group and count items',
      LucideIcons.barChart3, Color(0xFF06B6D4),
      category: 'Data transformation'),
  _NodeDef('limit', 'Limit', 'Restrict number of items',
      LucideIcons.arrowDownToLine, Color(0xFF06B6D4),
      category: 'Data transformation'),
  _NodeDef('split_out', 'Split Out', 'Split array into items',
      LucideIcons.splitSquareVertical, Color(0xFF06B6D4),
      category: 'Data transformation'),
  _NodeDef('remove_duplicates', 'Remove Duplicates', 'Deduplicate items',
      LucideIcons.copyMinus, Color(0xFF06B6D4),
      category: 'Data transformation'),
  _NodeDef('date_time', 'Date & Time', 'Format or manipulate dates',
      LucideIcons.calendar, Color(0xFF06B6D4),
      category: 'Data transformation'),
  _NodeDef('convert_to_json', 'Convert to JSON', 'Serialize to JSON string',
      LucideIcons.fileJson, Color(0xFF06B6D4),
      category: 'Data transformation'),
  _NodeDef('extract_from_json', 'Extract from JSON', 'Parse JSON field',
      LucideIcons.fileSearch, Color(0xFF06B6D4),
      category: 'Data transformation'),
  _NodeDef('html_parse', 'HTML', 'Work with HTML content',
      LucideIcons.code, Color(0xFF06B6D4),
      category: 'Data transformation'),
  _NodeDef('crypto', 'Crypto', 'Hash or encode data',
      LucideIcons.lock, Color(0xFF06B6D4),
      category: 'Data transformation'),

  // ── Error handling ──
  _NodeDef('try_catch', 'Try / Catch', 'Handle errors gracefully',
      LucideIcons.shield, Color(0xFFEF4444),
      category: 'Flow', outputs: 2, outputLabels: ['success', 'error']),
  _NodeDef('stop_and_error', 'Stop and Error', 'Fail workflow with message',
      LucideIcons.octagon, Color(0xFFEF4444), category: 'Flow'),

  // ── AI ──
  _NodeDef('ai_transform', 'AI Transform', 'Transform data with LLM',
      LucideIcons.sparkles, Color(0xFF8B5CF6), category: 'AI'),
  _NodeDef('ai_agent', 'AI Agent', 'Multi-step LLM agent with tools',
      LucideIcons.bot, Color(0xFF8B5CF6), category: 'AI'),
  _NodeDef('ai_summarize', 'AI Summarize', 'Summarize text content',
      LucideIcons.fileText, Color(0xFF8B5CF6), category: 'AI'),

  // ── Integrations ──
  _NodeDef('slack', 'Slack', 'Send Slack messages',
      LucideIcons.hash, Color(0xFF4A154B), category: 'Integrations'),
  _NodeDef('discord', 'Discord', 'Send Discord messages',
      LucideIcons.messageCircle, Color(0xFF5865F2),
      category: 'Integrations'),
  _NodeDef('telegram', 'Telegram', 'Send Telegram messages',
      LucideIcons.send, Color(0xFF26A5E4), category: 'Integrations'),
  _NodeDef('github', 'GitHub', 'Create issues, PRs',
      LucideIcons.github, Color(0xFFE6EDF3), category: 'Integrations'),
  _NodeDef('google_sheets', 'Google Sheets', 'Read/write spreadsheets',
      LucideIcons.sheet, Color(0xFF34A853), category: 'Integrations'),
  _NodeDef('notion', 'Notion', 'Query databases, create pages',
      LucideIcons.bookOpen, Color(0xFFFFFFFF), category: 'Integrations'),
  _NodeDef('stripe', 'Stripe', 'Charges, customers, payments',
      LucideIcons.creditCard, Color(0xFF635BFF), category: 'Integrations'),
  _NodeDef('twilio_sms', 'Twilio SMS', 'Send SMS messages',
      LucideIcons.phone, Color(0xFFF22F46), category: 'Integrations'),
  _NodeDef('postgres_query', 'PostgreSQL', 'Run SQL queries',
      LucideIcons.database, Color(0xFF336791), category: 'Integrations'),
  _NodeDef('mysql_query', 'MySQL', 'Run SQL queries',
      LucideIcons.database, Color(0xFF4479A1), category: 'Integrations'),
  _NodeDef('redis_command', 'Redis', 'Run Redis commands',
      LucideIcons.database, Color(0xFFDC382D), category: 'Integrations'),
  _NodeDef('s3', 'AWS S3', 'Get, put, list objects',
      LucideIcons.cloud, Color(0xFFFF9900), category: 'Integrations'),
  _NodeDef('sendgrid', 'SendGrid', 'Send transactional emails',
      LucideIcons.mail, Color(0xFF1A82E2), category: 'Integrations'),
  _NodeDef('jira', 'Jira', 'Create and manage issues',
      LucideIcons.ticket, Color(0xFF0052CC), category: 'Integrations'),

  // ── Applad-native ──
  _NodeDef('applad_auth', 'Applad Auth', 'Manage users and sessions',
      LucideIcons.users, Color(0xFF3472A4), category: 'Applad'),
  _NodeDef('applad_database', 'Applad Database', 'CRUD documents in collections',
      LucideIcons.database, Color(0xFF3472A4), category: 'Applad'),
  _NodeDef('applad_storage', 'Applad Storage', 'Manage files in buckets',
      LucideIcons.folderClosed, Color(0xFF3472A4), category: 'Applad'),
  _NodeDef('applad_functions', 'Applad Functions', 'Invoke serverless targets',
      LucideIcons.zap, Color(0xFF3472A4), category: 'Applad'),
  _NodeDef('applad_messaging', 'Applad Messaging', 'Send email, SMS, push',
      LucideIcons.messageSquare, Color(0xFF3472A4), category: 'Applad'),

  // ── Additional flow ──
  _NodeDef('sort', 'Sort', 'Sort items by field',
      LucideIcons.arrowUpDown, Color(0xFF06B6D4), category: 'Data transformation'),
  _NodeDef('rename_keys', 'Rename Keys', 'Rename fields on items',
      LucideIcons.replace, Color(0xFF06B6D4), category: 'Data transformation'),
  _NodeDef('compare_datasets', 'Compare Datasets', 'Find added/removed/unchanged',
      LucideIcons.gitCompare, Color(0xFF06B6D4), category: 'Data transformation'),
];

// Flat list for quick lookup
List<_NodeDef> get _nodeDefs => _allNodeDefs;

_NodeDef _def(String t) {
  if (t == 'trigger') return _triggerDef;
  return _allNodeDefs.firstWhere((d) => d.type == t,
      orElse: () => _allNodeDefs.firstWhere((d) => d.type == 'http_request'));
}

// ═════════════════════════════════════════════════════════════════════════════
// Undo / redo command stack
// ═════════════════════════════════════════════════════════════════════════════

class _Snapshot {
  final List<Map<String, dynamic>> nodes;
  final List<Map<String, dynamic>> edges;
  _Snapshot(this.nodes, this.edges);
}

class _UndoStack {
  final _undos = <_Snapshot>[];
  final _redos = <_Snapshot>[];

  void push(_Snapshot s) {
    _undos.add(s);
    _redos.clear();
  }

  _Snapshot? undo(_Snapshot current) {
    if (_undos.isEmpty) return null;
    _redos.add(current);
    return _undos.removeLast();
  }

  _Snapshot? redo(_Snapshot current) {
    if (_redos.isEmpty) return null;
    _undos.add(current);
    return _redos.removeLast();
  }

  bool get canUndo => _undos.isNotEmpty;
  bool get canRedo => _redos.isNotEmpty;
}

// ═════════════════════════════════════════════════════════════════════════════
// Top-level page
// ═════════════════════════════════════════════════════════════════════════════

class WorkflowsPage extends ConsumerStatefulWidget {
  const WorkflowsPage({super.key});
  @override
  ConsumerState<WorkflowsPage> createState() => _WorkflowsPageState();
}

class _WorkflowsPageState extends ConsumerState<WorkflowsPage> {
  Map<String, dynamic>? _editing;

  @override
  Widget build(BuildContext context) {
    if (_editing != null) {
      return _Editor(
        workflow: _editing!,
        onBack: () => setState(() => _editing = null),
        onSaved: () => ref.invalidate(workflowsProvider),
      );
    }
    return _ListPage(
      onSelect: (wf) => setState(() => _editing = wf),
      onCreate: _create,
    );
  }

  void _create() {
    final nameCtrl = TextEditingController();
    String trigger = 'manual';
    final cronCtrl = TextEditingController();

    showDialog(
      context: context,
      barrierColor: Colors.black.withOpacity(0.6),
      builder: (ctx) => StatefulBuilder(
        builder: (ctx, setD) => Center(
          child: Material(
            color: Colors.transparent,
            child: Container(
              width: 440,
              decoration: BoxDecoration(
                color: const Color(0xFF16171B),
                borderRadius: BorderRadius.circular(12),
                border: Border.all(color: Colors.white.withOpacity(0.08)),
                boxShadow: [
                  BoxShadow(
                    color: Colors.black.withOpacity(0.5),
                    blurRadius: 32,
                    offset: const Offset(0, 8),
                  ),
                ],
              ),
              child: Column(
                mainAxisSize: MainAxisSize.min,
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Padding(
                    padding: const EdgeInsets.fromLTRB(20, 20, 20, 0),
                    child: Row(
                      children: [
                        const Expanded(
                          child: Text('New Workflow',
                              style: TextStyle(
                                color: Colors.white,
                                fontSize: 16,
                                fontWeight: FontWeight.w600,
                              )),
                        ),
                        GestureDetector(
                          onTap: () => Navigator.of(ctx).pop(),
                          child: Icon(LucideIcons.x,
                              size: 16, color: Colors.white.withOpacity(0.3)),
                        ),
                      ],
                    ),
                  ),
                  const SizedBox(height: 16),
                  Padding(
                    padding: const EdgeInsets.symmetric(horizontal: 20),
                    child: Container(
                        height: 1, color: Colors.white.withOpacity(0.06)),
                  ),
                  const SizedBox(height: 16),
                  Padding(
                    padding: const EdgeInsets.symmetric(horizontal: 20),
                    child: Column(mainAxisSize: MainAxisSize.min, children: [
                      _dField(nameCtrl, 'Workflow name'),
                      const SizedBox(height: 16),
                      Row(children: [
                        _trigChip('manual', 'Manual', LucideIcons.play, trigger,
                            (v) => setD(() => trigger = v)),
                        const SizedBox(width: 8),
                        _trigChip('webhook', 'Webhook', LucideIcons.webhook,
                            trigger, (v) => setD(() => trigger = v)),
                        const SizedBox(width: 8),
                        _trigChip('cron', 'Schedule', LucideIcons.clock, trigger,
                            (v) => setD(() => trigger = v)),
                      ]),
                      if (trigger == 'cron') ...[
                        const SizedBox(height: 12),
                        _dField(cronCtrl, 'Cron (e.g. */5 * * * *)'),
                      ],
                    ]),
                  ),
                  const SizedBox(height: 16),
                  Padding(
                    padding: const EdgeInsets.fromLTRB(20, 0, 20, 20),
                    child: Row(
                      mainAxisAlignment: MainAxisAlignment.end,
                      children: [
                        const AppDialogCancel(),
                        AppDialogAction(
                          label: 'Create',
                          onTap: () async {
                            if (nameCtrl.text.trim().isEmpty) return;
                            try {
                              final api = ref.read(apiClientProvider);
                              final trigLabel = trigger == 'webhook'
                                  ? 'Webhook Trigger'
                                  : trigger == 'cron'
                                      ? 'Schedule Trigger'
                                      : 'Manual Trigger';
                              final res = await api.post('/workflows', data: {
                                'name': nameCtrl.text.trim(),
                                'description': '',
                                'triggerType': trigger,
                                if (trigger == 'cron')
                                  'triggerConfig': {'cron': cronCtrl.text.trim()},
                                'nodes': [
                                  {
                                    'id': 'trigger_0',
                                    'type': 'trigger',
                                    'label': trigLabel,
                                    'config': <String, dynamic>{},
                                    'position': {'x': 300.0, 'y': 300.0},
                                  }
                                ],
                                'edges': <Map<String, dynamic>>[],
                              });
                              if (ctx.mounted) Navigator.pop(ctx);
                              ref.invalidate(workflowsProvider);
                              setState(
                                  () => _editing = res.data as Map<String, dynamic>);
                            } catch (_) {
                              if (ctx.mounted) Navigator.pop(ctx);
                            }
                          },
                        ),
                      ],
                    ),
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }

  Widget _trigChip(String val, String label, IconData icon, String cur,
      ValueChanged<String> onTap) {
    final on = val == cur;
    return Expanded(
      child: GestureDetector(
        onTap: () => onTap(val),
        child: Container(
          padding: const EdgeInsets.symmetric(vertical: 12),
          decoration: BoxDecoration(
            color: on ? _accent.withOpacity(0.15) : _surface,
            borderRadius: BorderRadius.circular(8),
            border: Border.all(color: on ? _accent : _border),
          ),
          child: Column(children: [
            Icon(icon, size: 18, color: on ? _accent : _dimText),
            const SizedBox(height: 6),
            Text(label,
                style: TextStyle(
                    color: on ? Colors.white : _dimText,
                    fontSize: 12,
                    fontWeight: on ? FontWeight.w600 : FontWeight.w400)),
          ]),
        ),
      ),
    );
  }

  Widget _dField(TextEditingController c, String hint) => TextField(
        controller: c,
        style: const TextStyle(color: Colors.white),
        decoration: InputDecoration(
          hintText: hint,
          hintStyle: const TextStyle(color: _subtleText),
          filled: true,
          fillColor: _bg,
          border: OutlineInputBorder(
              borderRadius: BorderRadius.circular(8),
              borderSide: const BorderSide(color: _border)),
          enabledBorder: OutlineInputBorder(
              borderRadius: BorderRadius.circular(8),
              borderSide: const BorderSide(color: _border)),
          focusedBorder: OutlineInputBorder(
              borderRadius: BorderRadius.circular(8),
              borderSide: const BorderSide(color: _accent)),
        ),
      );
}

// ═════════════════════════════════════════════════════════════════════════════
// Workflow list page
// ═════════════════════════════════════════════════════════════════════════════

class _ListPage extends ConsumerWidget {
  final ValueChanged<Map<String, dynamic>> onSelect;
  final VoidCallback onCreate;
  const _ListPage({required this.onSelect, required this.onCreate});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final wAsync = ref.watch(workflowsProvider);
    return Container(
      color: _bg,
      child: Column(children: [
        Padding(
          padding: const EdgeInsets.fromLTRB(32, 28, 32, 0),
          child: Row(children: [
            const Expanded(
              child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text('Workflows',
                        style: TextStyle(
                            color: Colors.white,
                            fontSize: 24,
                            fontWeight: FontWeight.w700)),
                    SizedBox(height: 4),
                    Text('Automate tasks with visual pipelines',
                        style: TextStyle(color: _dimText, fontSize: 14)),
                  ]),
            ),
            FilledButton.icon(
              style: FilledButton.styleFrom(
                  backgroundColor: _accent,
                  shape: RoundedRectangleBorder(
                      borderRadius: BorderRadius.circular(8)),
                  padding: const EdgeInsets.symmetric(
                      horizontal: 20, vertical: 12)),
              icon: const Icon(LucideIcons.plus, size: 16),
              label:
                  const Text('New Workflow', style: TextStyle(fontSize: 13)),
              onPressed: onCreate,
            ),
          ]),
        ),
        const SizedBox(height: 24),
        Expanded(
          child: wAsync.when(
            loading: () =>
                const Center(child: CircularProgressIndicator(color: _accent)),
            error: (e, _) => Center(
                child: Text('$e', style: const TextStyle(color: _dimText))),
            data: (data) {
              final list = List<Map<String, dynamic>>.from(
                  data['workflows'] ?? []);
              if (list.isEmpty) return _empty();
              return _grid(list);
            },
          ),
        ),
      ]),
    );
  }

  Widget _empty() => Center(
        child: Column(mainAxisSize: MainAxisSize.min, children: [
          Container(
            width: 72,
            height: 72,
            decoration: BoxDecoration(
                color: _accent.withOpacity(0.1), shape: BoxShape.circle),
            child: const Icon(LucideIcons.gitBranch, size: 32, color: _accent),
          ),
          const SizedBox(height: 20),
          const Text('No workflows yet',
              style: TextStyle(
                  color: Colors.white,
                  fontSize: 18,
                  fontWeight: FontWeight.w600)),
          const SizedBox(height: 8),
          const Text('Create your first workflow to get started',
              style: TextStyle(color: _dimText, fontSize: 14)),
          const SizedBox(height: 24),
          FilledButton.icon(
            style: FilledButton.styleFrom(
                backgroundColor: _accent,
                shape: RoundedRectangleBorder(
                    borderRadius: BorderRadius.circular(8)),
                padding: const EdgeInsets.symmetric(
                    horizontal: 20, vertical: 12)),
            icon: const Icon(LucideIcons.plus, size: 16),
            label: const Text('New Workflow'),
            onPressed: onCreate,
          ),
        ]),
      );

  Widget _grid(List<Map<String, dynamic>> wfs) => Padding(
        padding: const EdgeInsets.symmetric(horizontal: 32),
        child: LayoutBuilder(builder: (ctx, box) {
          final cols = (box.maxWidth / 340).floor().clamp(1, 4);
          final w = (box.maxWidth - (cols - 1) * 16) / cols;
          return SingleChildScrollView(
            child: Wrap(
                spacing: 16,
                runSpacing: 16,
                children: wfs.map((wf) => _Card(wf, w, onSelect)).toList()),
          );
        }),
      );
}

class _Card extends StatefulWidget {
  final Map<String, dynamic> wf;
  final double w;
  final ValueChanged<Map<String, dynamic>> onSelect;
  const _Card(this.wf, this.w, this.onSelect);
  @override
  State<_Card> createState() => _CardState();
}

class _CardState extends State<_Card> {
  bool _h = false;

  @override
  Widget build(BuildContext context) {
    final name = widget.wf['name'] ?? 'Unnamed';
    final status = widget.wf['status'] ?? 'draft';
    final trigger = widget.wf['triggerType'] ?? 'manual';
    final nc = (widget.wf['nodes'] as List?)?.length ?? 0;

    return MouseRegion(
      onEnter: (_) => setState(() => _h = true),
      onExit: (_) => setState(() => _h = false),
      child: GestureDetector(
        onTap: () => widget.onSelect(widget.wf),
        child: AnimatedContainer(
          duration: const Duration(milliseconds: 120),
          width: widget.w,
          padding: const EdgeInsets.all(20),
          decoration: BoxDecoration(
            color: _h ? _surfaceHover : _surface,
            borderRadius: BorderRadius.circular(12),
            border:
                Border.all(color: _h ? _accent.withOpacity(0.3) : _border),
          ),
          child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
            Row(children: [
              Container(
                width: 36,
                height: 36,
                decoration: BoxDecoration(
                    color: _accent.withOpacity(0.12),
                    borderRadius: BorderRadius.circular(8)),
                child: const Icon(LucideIcons.gitBranch,
                    size: 18, color: _accent),
              ),
              const SizedBox(width: 12),
              Expanded(
                  child: Text(name,
                      style: const TextStyle(
                          color: Colors.white,
                          fontSize: 15,
                          fontWeight: FontWeight.w600),
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis)),
              _dot(status),
            ]),
            const SizedBox(height: 16),
            Row(children: [
              _chip(_trigIcon(trigger), trigger),
              const SizedBox(width: 8),
              _chip(LucideIcons.boxes, '$nc nodes'),
            ]),
          ]),
        ),
      ),
    );
  }

  Widget _dot(String s) {
    final c = s == 'active'
        ? _green
        : s == 'paused'
            ? _orange
            : const Color(0xFF64748B);
    return Row(mainAxisSize: MainAxisSize.min, children: [
      Container(
          width: 8,
          height: 8,
          decoration: BoxDecoration(color: c, shape: BoxShape.circle)),
      const SizedBox(width: 6),
      Text(s,
          style: TextStyle(color: c, fontSize: 12, fontWeight: FontWeight.w500)),
    ]);
  }

  IconData _trigIcon(String t) => t == 'webhook'
      ? LucideIcons.webhook
      : t == 'cron'
          ? LucideIcons.clock
          : LucideIcons.play;

  Widget _chip(IconData ic, String l) => Container(
        padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
        decoration: BoxDecoration(
            color: Colors.white.withOpacity(0.05),
            borderRadius: BorderRadius.circular(6)),
        child: Row(mainAxisSize: MainAxisSize.min, children: [
          Icon(ic, size: 12, color: _subtleText),
          const SizedBox(width: 5),
          Text(l, style: const TextStyle(color: _subtleText, fontSize: 11)),
        ]),
      );
}

// ═════════════════════════════════════════════════════════════════════════════
// Workflow editor — n8n-style canvas
// ═════════════════════════════════════════════════════════════════════════════

enum _Mode { idle, panCanvas, dragNode, dragConn, selectRect }

class _Editor extends ConsumerStatefulWidget {
  final Map<String, dynamic> workflow;
  final VoidCallback onBack, onSaved;
  const _Editor(
      {required this.workflow, required this.onBack, required this.onSaved});
  @override
  ConsumerState<_Editor> createState() => _EditorState();
}

class _EditorState extends ConsumerState<_Editor> {
  // ── Data ──
  late String _name, _status, _triggerType;
  late Map<String, dynamic> _triggerConfig;
  late List<Map<String, dynamic>> _nodes, _edges;
  String get _wfId => widget.workflow['\$id'] as String;

  // ── Canvas state ──
  Offset _pan = Offset.zero;
  double _zoom = 1.0;
  _Mode _mode = _Mode.idle;

  // ── Interaction state ──
  final Set<String> _selected = {};
  int? _dragIdx;
  Offset _dragStart = Offset.zero;
  Map<String, Offset>? _dragStartPositions;
  String? _connFrom;
  Offset? _connCursor;
  Offset? _rectStart, _rectEnd;

  // ── UI state ──
  String? _configNodeId;
  bool _showPalette = false;
  Offset? _palettePos;
  String? _paletteConnFrom;
  bool _saving = false;
  bool _dirty = false;

  // ── Undo ──
  final _undo = _UndoStack();

  // ── Execution state ──
  Map<String, String>? _execStatus; // nodeId → 'running'|'completed'|'failed'
  Map<String, int>? _execCounts; // edgeId → item count

  // ── Tags, logs, minimap, sticky notes ──
  List<String> _tags = [];
  bool _showLogs = false;
  bool _showMinimap = true;
  List<Map<String, dynamic>> _stickyNotes = [];
  int _topTab = 0; // 0=Editor, 1=Executions
  int _configTab = 0; // 0=Settings, 1=Input, 2=Output
  Map<String, dynamic>? _lastExecData; // nodeId → {input, output}
  Map<String, dynamic>? _pinnedData; // nodeId → pinned output

  @override
  void initState() {
    super.initState();
    final wf = widget.workflow;
    _name = wf['name'] ?? 'Unnamed';
    _status = wf['status'] ?? 'draft';
    _triggerType = wf['triggerType'] ?? 'manual';
    _triggerConfig = Map<String, dynamic>.from(wf['triggerConfig'] ?? {});
    _nodes = _cloneList(wf['nodes'] as List? ?? []);
    _edges = _cloneList(wf['edges'] as List? ?? []);
    for (var i = 0; i < _nodes.length; i++) {
      _nodes[i]['position'] ??= {'x': 300.0 + i * 280.0, 'y': 300.0};
      _nodes[i]['disabled'] ??= false;
    }
  }

  List<Map<String, dynamic>> _cloneList(List src) =>
      src.map((e) => Map<String, dynamic>.from(e as Map)).toList();

  // ── Snapshot helpers ──
  _Snapshot _snap() => _Snapshot(_cloneList(_nodes), _cloneList(_edges));
  void _pushUndo() => _undo.push(_snap());
  void _applySnap(_Snapshot s) {
    _nodes = _cloneList(s.nodes);
    _edges = _cloneList(s.edges);
  }

  // ── Coordinate helpers ──
  Offset _toCanvas(Offset screen) => (screen - _pan) / _zoom;

  Offset _nodePos(Map<String, dynamic> n) {
    final p = n['position'] as Map<String, dynamic>?;
    return Offset(
        (p?['x'] as num?)?.toDouble() ?? 0, (p?['y'] as num?)?.toDouble() ?? 0);
  }

  void _setPos(int i, Offset o) {
    _nodes[i]['position'] = {
      'x': (o.dx / _gridSnap).round() * _gridSnap,
      'y': (o.dy / _gridSnap).round() * _gridSnap,
    };
  }

  /// Output handle position for output index [oi] (0-based).
  Offset _outHandle(Map<String, dynamic> n, [int oi = 0]) {
    final d = _def(n['type'] ?? '');
    final total = d.outputs;
    if (total <= 1) return _nodePos(n) + const Offset(_nodeW, _nodeH / 2);
    final spacing = _nodeH / (total + 1);
    return _nodePos(n) + Offset(_nodeW, spacing * (oi + 1));
  }

  /// Input handle position for input index [ii] (0-based).
  Offset _inHandle(Map<String, dynamic> n, [int ii = 0]) {
    final d = _def(n['type'] ?? '');
    final total = d.inputs;
    if (total <= 1) return _nodePos(n) + const Offset(0, _nodeH / 2);
    final spacing = _nodeH / (total + 1);
    return _nodePos(n) + Offset(0, spacing * (ii + 1));
  }

  Map<String, dynamic>? _byId(String id) =>
      _nodes.where((n) => n['id'] == id).firstOrNull;

  int? _hitNode(Offset canvasPos) {
    for (var i = _nodes.length - 1; i >= 0; i--) {
      final p = _nodePos(_nodes[i]);
      if (canvasPos.dx >= p.dx &&
          canvasPos.dx <= p.dx + _nodeW &&
          canvasPos.dy >= p.dy &&
          canvasPos.dy <= p.dy + _nodeH) {
        return i;
      }
    }
    return null;
  }

  int? _hitOutputHandle(Offset canvasPos) {
    for (var i = 0; i < _nodes.length; i++) {
      final d = _def(_nodes[i]['type'] ?? '');
      for (var oi = 0; oi < d.outputs; oi++) {
        final h = _outHandle(_nodes[i], oi);
        if ((canvasPos - h).distance < 14) return i;
      }
    }
    return null;
  }

  // ── Mutators ──
  void _addNode(String type, Offset pos, {String? connectFrom}) {
    _pushUndo();
    final id = 'node_${DateTime.now().millisecondsSinceEpoch}';
    final d = _def(type);
    _nodes.add({
      'id': id,
      'type': type,
      'label': d.label,
      'config': <String, dynamic>{},
      'position': {
        'x': (pos.dx / _gridSnap).round() * _gridSnap,
        'y': (pos.dy / _gridSnap).round() * _gridSnap,
      },
      'disabled': false,
    });
    if (connectFrom != null) {
      _edges.add({
        'id': 'e_${DateTime.now().millisecondsSinceEpoch}',
        'source': connectFrom,
        'target': id,
      });
    }
    _selected
      ..clear()
      ..add(id);
    _dirty = true;
  }

  void _deleteSelected() {
    if (_selected.isEmpty) return;
    _pushUndo();
    final ids = Set<String>.from(_selected);
    _nodes.removeWhere((n) => ids.contains(n['id']) && n['type'] != 'trigger');
    _edges.removeWhere(
        (e) => ids.contains(e['source']) || ids.contains(e['target']));
    _selected.clear();
    _configNodeId = null;
    _dirty = true;
  }

  void _duplicateSelected() {
    if (_selected.isEmpty) return;
    _pushUndo();
    final idMap = <String, String>{};
    final newIds = <String>{};
    for (final id in _selected) {
      final orig = _byId(id);
      if (orig == null || orig['type'] == 'trigger') continue;
      final nid = 'node_${DateTime.now().millisecondsSinceEpoch}_${idMap.length}';
      idMap[id] = nid;
      final pos = _nodePos(orig);
      _nodes.add({
        ...Map<String, dynamic>.from(orig),
        'id': nid,
        'position': {'x': pos.dx + 40, 'y': pos.dy + 40},
      });
      newIds.add(nid);
    }
    // Duplicate internal edges
    for (final e in List<Map<String, dynamic>>.from(_edges)) {
      final ns = idMap[e['source']];
      final nt = idMap[e['target']];
      if (ns != null && nt != null) {
        _edges.add({
          'id': 'e_${DateTime.now().millisecondsSinceEpoch}_${_edges.length}',
          'source': ns,
          'target': nt,
        });
      }
    }
    _selected
      ..clear()
      ..addAll(newIds);
    _dirty = true;
  }

  void _toggleDisable() {
    if (_selected.isEmpty) return;
    _pushUndo();
    for (final id in _selected) {
      final n = _byId(id);
      if (n != null && n['type'] != 'trigger') {
        n['disabled'] = !(n['disabled'] == true);
      }
    }
    _dirty = true;
  }

  void _addEdge(String src, String tgt) {
    if (src == tgt) return;
    if (_edges.any((e) => e['source'] == src && e['target'] == tgt)) return;
    _pushUndo();
    _edges.add({
      'id': 'e_${DateTime.now().millisecondsSinceEpoch}',
      'source': src,
      'target': tgt,
    });
    _dirty = true;
  }

  // ── Save & execute ──
  Future<void> _save() async {
    setState(() => _saving = true);
    try {
      final api = ref.read(apiClientProvider);
      final saveNodes = _nodes.map((n) {
        final c = Map<String, dynamic>.from(n);
        if (c['type'] == 'trigger') {
          c['type'] = 'set_variable';
          c['config'] = {'key': '_trigger', 'value': 'start'};
        }
        c.remove('disabled');
        return c;
      }).toList();
      await api.put('/workflows/$_wfId', data: {
        'name': _name,
        'description': widget.workflow['description'] ?? '',
        'status': _status,
        'triggerType': _triggerType,
        'triggerConfig': _triggerConfig,
        'nodes': saveNodes,
        'edges': _edges,
      });
      _dirty = false;
      widget.onSaved();
    } catch (_) {}
    if (mounted) setState(() => _saving = false);
  }

  Future<void> _execute() async {
    try {
      final api = ref.read(apiClientProvider);
      // Show running state on all nodes
      setState(() {
        _execStatus = {};
        _execCounts = null;
        for (final n in _nodes) {
          _execStatus![n['id'] as String] = 'running';
        }
      });
      final res = await api.post('/workflows/$_wfId/execute',
          data: {'triggerData': <String, dynamic>{}});
      // Parse execution result
      if (res.data is Map) {
        final exec = res.data as Map<String, dynamic>;
        final logs = List<Map<String, dynamic>>.from(exec['logs'] ?? []);
        final statuses = <String, String>{};
        for (final log in logs) {
          statuses[log['nodeId'] as String? ?? ''] =
              log['status'] as String? ?? 'completed';
        }
        setState(() => _execStatus = statuses);
      }
    } catch (_) {
      setState(() => _execStatus = null);
    }
  }

  // ═══════════════════════════════════════════════════════════════════════════
  // Build
  // ═══════════════════════════════════════════════════════════════════════════

  @override
  Widget build(BuildContext context) {
    return Shortcuts(
      shortcuts: {
        LogicalKeySet(LogicalKeyboardKey.meta, LogicalKeyboardKey.keyZ):
            const _UndoIntent(),
        LogicalKeySet(
                LogicalKeyboardKey.meta,
                LogicalKeyboardKey.shift,
                LogicalKeyboardKey.keyZ):
            const _RedoIntent(),
        LogicalKeySet(LogicalKeyboardKey.meta, LogicalKeyboardKey.keyC):
            const _CopyIntent(),
        LogicalKeySet(LogicalKeyboardKey.meta, LogicalKeyboardKey.keyV):
            const _PasteIntent(),
        LogicalKeySet(LogicalKeyboardKey.meta, LogicalKeyboardKey.keyD):
            const _DuplicateIntent(),
        LogicalKeySet(LogicalKeyboardKey.meta, LogicalKeyboardKey.keyA):
            const _SelectAllIntent(),
        LogicalKeySet(LogicalKeyboardKey.meta, LogicalKeyboardKey.keyS):
            const _SaveIntent(),
        LogicalKeySet(LogicalKeyboardKey.delete): const _DeleteIntent(),
        LogicalKeySet(LogicalKeyboardKey.backspace): const _DeleteIntent(),
        const SingleActivator(LogicalKeyboardKey.keyD): const _DisableIntent(),
        const SingleActivator(LogicalKeyboardKey.tab): const _AddNodeIntent(),
        const SingleActivator(LogicalKeyboardKey.escape): const _EscapeIntent(),
      },
      child: Actions(
        actions: {
          _UndoIntent: CallbackAction<_UndoIntent>(onInvoke: (_) {
            final s = _undo.undo(_snap());
            if (s != null) setState(() => _applySnap(s));
            return null;
          }),
          _RedoIntent: CallbackAction<_RedoIntent>(onInvoke: (_) {
            final s = _undo.redo(_snap());
            if (s != null) setState(() => _applySnap(s));
            return null;
          }),
          _DeleteIntent:
              CallbackAction<_DeleteIntent>(onInvoke: (_) {
            setState(() => _deleteSelected());
            return null;
          }),
          _DuplicateIntent:
              CallbackAction<_DuplicateIntent>(onInvoke: (_) {
            setState(() => _duplicateSelected());
            return null;
          }),
          _SelectAllIntent:
              CallbackAction<_SelectAllIntent>(onInvoke: (_) {
            setState(() {
              _selected.clear();
              for (final n in _nodes) {
                _selected.add(n['id'] as String);
              }
            });
            return null;
          }),
          _SaveIntent: CallbackAction<_SaveIntent>(onInvoke: (_) {
            _save();
            return null;
          }),
          _DisableIntent:
              CallbackAction<_DisableIntent>(onInvoke: (_) {
            setState(() => _toggleDisable());
            return null;
          }),
          _AddNodeIntent:
              CallbackAction<_AddNodeIntent>(onInvoke: (_) {
            setState(() {
              _showPalette = true;
              _palettePos = _toCanvas(Offset(
                  MediaQuery.of(context).size.width / 2,
                  MediaQuery.of(context).size.height / 2));
              _paletteConnFrom = null;
            });
            return null;
          }),
          _EscapeIntent:
              CallbackAction<_EscapeIntent>(onInvoke: (_) {
            setState(() {
              _selected.clear();
              _configNodeId = null;
              _showPalette = false;
              _connFrom = null;
              _connCursor = null;
            });
            return null;
          }),
          _CopyIntent: CallbackAction<_CopyIntent>(onInvoke: (_) => null),
          _PasteIntent: CallbackAction<_PasteIntent>(onInvoke: (_) => null),
        },
        child: Focus(
          autofocus: true,
          child: Scaffold(
            backgroundColor: _bg,
            body: Column(children: [
              _toolbar(),
              // Editor / Executions tabs
              _topTabs(),
              Expanded(
                child: Stack(children: [
                        Column(children: [
                          Expanded(child: _canvas()),
                          if (_showLogs) _logsPanel(),
                        ]),
                        _zoomControls(),
                        if (_showMinimap) _minimap(),
                        if (_showPalette) _palette(),
                        if (_configNodeId != null) _configPanel(),
                      ]),
              ),
            ]),
          ),
        ),
      ),
    );
  }

  // ═══════════════════════════════════════════════════════════════════════════
  // Toolbar
  // ═══════════════════════════════════════════════════════════════════════════

  Widget _toolbar() {
    return Container(
      height: 52,
      padding: const EdgeInsets.symmetric(horizontal: 16),
      decoration: BoxDecoration(
          color: _bg,
          border: Border(
              bottom: BorderSide(color: Colors.white.withOpacity(0.06)))),
      child: Row(children: [
        IconButton(
            onPressed: widget.onBack,
            icon: const Icon(LucideIcons.arrowLeft, size: 18, color: _dimText),
            tooltip: 'Back'),
        const SizedBox(width: 8),
        IntrinsicWidth(
          child: TextField(
            controller: TextEditingController(text: _name),
            style: const TextStyle(
                color: Colors.white, fontSize: 15, fontWeight: FontWeight.w600),
            decoration: const InputDecoration(
                border: InputBorder.none,
                enabledBorder: InputBorder.none,
                focusedBorder: InputBorder.none,
                filled: false,
                contentPadding:
                    EdgeInsets.symmetric(horizontal: 4, vertical: 8)),
            onChanged: (v) {
              _name = v;
              _dirty = true;
            },
          ),
        ),
        const SizedBox(width: 12),
        _statusChip(),
        const Spacer(),

        // Undo/redo
        IconButton(
          onPressed: _undo.canUndo
              ? () {
                  final s = _undo.undo(_snap());
                  if (s != null) setState(() => _applySnap(s));
                }
              : null,
          icon: Icon(LucideIcons.undo2,
              size: 15,
              color: _undo.canUndo
                  ? _dimText
                  : Colors.white.withOpacity(0.15)),
          tooltip: 'Undo (⌘Z)',
        ),
        IconButton(
          onPressed: _undo.canRedo
              ? () {
                  final s = _undo.redo(_snap());
                  if (s != null) setState(() => _applySnap(s));
                }
              : null,
          icon: Icon(LucideIcons.redo2,
              size: 15,
              color: _undo.canRedo
                  ? _dimText
                  : Colors.white.withOpacity(0.15)),
          tooltip: 'Redo (⌘⇧Z)',
        ),

        const SizedBox(width: 4),
        Container(width: 1, height: 24, color: _border),
        const SizedBox(width: 4),

        // History
        TextButton.icon(
          onPressed: _showExecs,
          icon: const Icon(LucideIcons.history, size: 15),
          label: const Text('History', style: TextStyle(fontSize: 13)),
          style: TextButton.styleFrom(foregroundColor: _dimText),
        ),
        const SizedBox(width: 4),

        // Execute
        OutlinedButton.icon(
          onPressed: _execute,
          icon: const Icon(LucideIcons.play, size: 14),
          label: const Text('Execute', style: TextStyle(fontSize: 13)),
          style: OutlinedButton.styleFrom(
            foregroundColor: _green,
            side: const BorderSide(color: _green),
            shape: RoundedRectangleBorder(
                borderRadius: BorderRadius.circular(8)),
            padding:
                const EdgeInsets.symmetric(horizontal: 14, vertical: 8),
          ),
        ),
        const SizedBox(width: 8),

        // Save
        FilledButton.icon(
          onPressed: _saving ? null : _save,
          icon: _saving
              ? const SizedBox(
                  width: 14,
                  height: 14,
                  child: CircularProgressIndicator(
                      strokeWidth: 2, color: Colors.white))
              : const Icon(LucideIcons.save, size: 14),
          label: Text(_dirty ? 'Save*' : 'Saved',
              style: const TextStyle(fontSize: 13)),
          style: FilledButton.styleFrom(
            backgroundColor: _dirty ? _accent : Colors.white.withOpacity(0.06),
            foregroundColor: _dirty ? Colors.white : _dimText,
            shape: RoundedRectangleBorder(
                borderRadius: BorderRadius.circular(8)),
            padding:
                const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
          ),
        ),
      ]),
    );
  }

  Widget _statusChip() {
    final c = _status == 'active'
        ? _green
        : _status == 'paused'
            ? _orange
            : const Color(0xFF64748B);
    return PopupMenuButton<String>(
      offset: const Offset(0, 32),
      color: const Color(0xFF1A1A22),
      shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(8),
          side: BorderSide(color: Colors.white.withOpacity(0.08))),
      onSelected: (v) => setState(() {
        _status = v;
        _dirty = true;
      }),
      itemBuilder: (_) => ['draft', 'active', 'paused']
          .map((s) => PopupMenuItem(
              value: s,
              child:
                  Text(s, style: const TextStyle(color: Colors.white70))))
          .toList(),
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
        decoration: BoxDecoration(
          color: c.withOpacity(0.12),
          borderRadius: BorderRadius.circular(20),
          border: Border.all(color: c.withOpacity(0.3)),
        ),
        child: Row(mainAxisSize: MainAxisSize.min, children: [
          Container(
              width: 6,
              height: 6,
              decoration: BoxDecoration(color: c, shape: BoxShape.circle)),
          const SizedBox(width: 6),
          Text(_status,
              style: TextStyle(
                  color: c, fontSize: 12, fontWeight: FontWeight.w500)),
          const SizedBox(width: 4),
          Icon(LucideIcons.chevronDown, size: 12, color: c),
        ]),
      ),
    );
  }

  // ═══════════════════════════════════════════════════════════════════════════
  // Canvas — manual pan/zoom (no InteractiveViewer, no gesture conflicts)
  // ═══════════════════════════════════════════════════════════════════════════

  Widget _canvas() {
    return Listener(
      behavior: HitTestBehavior.opaque,
      onPointerDown: _onPointerDown,
      onPointerMove: _onPointerMove,
      onPointerUp: _onPointerUp,
      onPointerSignal: _onPointerSignal,
      child: ClipRect(
        child: CustomPaint(
          painter: _CanvasPainter(
            pan: _pan,
            zoom: _zoom,
            nodes: _nodes,
            edges: _edges,
            selected: _selected,
            connFrom: _connFrom != null ? _byId(_connFrom!) : null,
            connCursor: _connCursor,
            rectStart: _rectStart,
            rectEnd: _rectEnd,
            execStatus: _execStatus,
            execCounts: _execCounts,
            stickyNotes: _stickyNotes,
          ),
          child: const SizedBox.expand(),
        ),
      ),
    );
  }

  // ── Pointer event handlers ──

  void _onPointerDown(PointerDownEvent e) {
    final canvasPos = _toCanvas(e.localPosition);

    // Right-click → context menu
    if (e.buttons == kSecondaryMouseButton) {
      _showContextMenu(e.position, canvasPos);
      return;
    }

    // Check output handle hit (start connection)
    final handleIdx = _hitOutputHandle(canvasPos);
    if (handleIdx != null) {
      setState(() {
        _mode = _Mode.dragConn;
        _connFrom = _nodes[handleIdx]['id'] as String;
        _connCursor = canvasPos;
      });
      return;
    }

    // Check node hit
    final nodeIdx = _hitNode(canvasPos);
    if (nodeIdx != null) {
      final id = _nodes[nodeIdx]['id'] as String;
      final shift = HardwareKeyboard.instance.logicalKeysPressed
          .contains(LogicalKeyboardKey.shiftLeft) || HardwareKeyboard.instance.logicalKeysPressed
          .contains(LogicalKeyboardKey.shiftRight);

      if (shift) {
        // Shift-click: toggle selection
        setState(() {
          if (_selected.contains(id)) {
            _selected.remove(id);
          } else {
            _selected.add(id);
          }
        });
        return;
      }

      if (!_selected.contains(id)) {
        _selected
          ..clear()
          ..add(id);
      }

      // Store start positions for all selected nodes
      _dragStartPositions = {};
      for (final sid in _selected) {
        final n = _byId(sid);
        if (n != null) _dragStartPositions![sid] = _nodePos(n);
      }

      setState(() {
        _mode = _Mode.dragNode;
        _dragIdx = nodeIdx;
        _dragStart = canvasPos;
      });
      return;
    }

    // Empty space: start canvas pan or rubber-band select
    final shiftHeld = HardwareKeyboard.instance.logicalKeysPressed
        .contains(LogicalKeyboardKey.shiftLeft) || HardwareKeyboard.instance.logicalKeysPressed
        .contains(LogicalKeyboardKey.shiftRight);

    if (shiftHeld) {
      setState(() {
        _mode = _Mode.selectRect;
        _rectStart = canvasPos;
        _rectEnd = canvasPos;
      });
    } else {
      setState(() {
        _mode = _Mode.panCanvas;
        _dragStart = e.localPosition;
        _selected.clear();
        _configNodeId = null;
        _showPalette = false;
      });
    }
  }

  void _onPointerMove(PointerMoveEvent e) {
    switch (_mode) {
      case _Mode.panCanvas:
        setState(() {
          _pan += e.localPosition - _dragStart;
          _dragStart = e.localPosition;
        });
      case _Mode.dragNode:
        final canvasPos = _toCanvas(e.localPosition);
        final delta = canvasPos - _dragStart;
        setState(() {
          for (final sid in _selected) {
            final startPos = _dragStartPositions?[sid];
            if (startPos == null) continue;
            final idx = _nodes.indexWhere((n) => n['id'] == sid);
            if (idx >= 0) {
              _nodes[idx]['position'] = {
                'x': (startPos.dx + delta.dx / 1).roundToDouble(),
                'y': (startPos.dy + delta.dy / 1).roundToDouble(),
              };
            }
          }
        });
      case _Mode.dragConn:
        setState(() => _connCursor = _toCanvas(e.localPosition));
      case _Mode.selectRect:
        setState(() => _rectEnd = _toCanvas(e.localPosition));
      case _Mode.idle:
        break;
    }
  }

  void _onPointerUp(PointerUpEvent e) {
    switch (_mode) {
      case _Mode.dragNode:
        // Snap to grid
        _pushUndo();
        for (final sid in _selected) {
          final idx = _nodes.indexWhere((n) => n['id'] == sid);
          if (idx >= 0) {
            final p = _nodePos(_nodes[idx]);
            _setPos(idx, p);
          }
        }
        _dirty = true;
        // Double-click detection: if barely moved, open config
        final canvasPos = _toCanvas(e.localPosition);
        if ((_dragStart - canvasPos).distance < 3 && _dragIdx != null) {
          _configNodeId = _nodes[_dragIdx!]['id'] as String;
          _showPalette = false;
        }
      case _Mode.dragConn:
        // Check if released on a node input handle
        if (_connCursor != null && _connFrom != null) {
          bool connected = false;
          for (final n in _nodes) {
            final id = n['id'] as String;
            if (id == _connFrom) continue;
            if ((_connCursor! - _inHandle(n)).distance < 24) {
              _addEdge(_connFrom!, id);
              connected = true;
              break;
            }
          }
          // Released on empty space → open node picker
          if (!connected) {
            _showPalette = true;
            _palettePos = _connCursor;
            _paletteConnFrom = _connFrom;
          }
        }
        _connFrom = null;
        _connCursor = null;
      case _Mode.selectRect:
        // Select all nodes inside the rectangle
        if (_rectStart != null && _rectEnd != null) {
          final r = Rect.fromPoints(_rectStart!, _rectEnd!);
          _selected.clear();
          for (final n in _nodes) {
            final p = _nodePos(n);
            final nodeRect = Rect.fromLTWH(p.dx, p.dy, _nodeW, _nodeH);
            if (r.overlaps(nodeRect)) {
              _selected.add(n['id'] as String);
            }
          }
        }
        _rectStart = null;
        _rectEnd = null;
      default:
        break;
    }
    setState(() => _mode = _Mode.idle);
  }

  void _onPointerSignal(PointerSignalEvent e) {
    if (e is PointerScrollEvent) {
      // Zoom centered on cursor
      final delta = -e.scrollDelta.dy * 0.001;
      final newZoom = (_zoom + delta * _zoom).clamp(0.15, 3.0);
      final focalScreen = e.localPosition;
      final focalCanvas = _toCanvas(focalScreen);
      setState(() {
        _zoom = newZoom;
        _pan = focalScreen - focalCanvas * _zoom;
      });
    }
  }

  // ── Context menu ──
  void _showContextMenu(Offset screenPos, Offset canvasPos) {
    final nodeIdx = _hitNode(canvasPos);
    showMenu<String>(
      context: context,
      position: RelativeRect.fromLTRB(
          screenPos.dx, screenPos.dy, screenPos.dx, screenPos.dy),
      color: const Color(0xFF1A1A22),
      shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(8),
          side: BorderSide(color: Colors.white.withOpacity(0.08))),
      items: nodeIdx != null
          ? [
              _ctxItem('open', 'Open settings', LucideIcons.settings),
              _ctxItem('duplicate', 'Duplicate', LucideIcons.copy),
              _ctxItem('disable', 'Toggle disable', LucideIcons.ban),
              const PopupMenuDivider(),
              _ctxItem('delete', 'Delete', LucideIcons.trash2,
                  color: _red),
            ]
          : [
              _ctxItem('add', 'Add node', LucideIcons.plus),
              _ctxItem('selectAll', 'Select all', LucideIcons.checkSquare),
              _ctxItem('fitView', 'Fit to view', LucideIcons.maximize2),
            ],
    ).then((action) {
      if (action == null) return;
      setState(() {
        switch (action) {
          case 'open':
            if (nodeIdx != null) {
              _configNodeId = _nodes[nodeIdx]['id'] as String;
              _selected
                ..clear()
                ..add(_configNodeId!);
            }
          case 'duplicate':
            if (nodeIdx != null) {
              _selected
                ..clear()
                ..add(_nodes[nodeIdx]['id'] as String);
              _duplicateSelected();
            }
          case 'disable':
            if (nodeIdx != null) {
              _selected
                ..clear()
                ..add(_nodes[nodeIdx]['id'] as String);
              _toggleDisable();
            }
          case 'delete':
            if (nodeIdx != null) {
              _selected
                ..clear()
                ..add(_nodes[nodeIdx]['id'] as String);
              _deleteSelected();
            }
          case 'add':
            _showPalette = true;
            _palettePos = canvasPos;
            _paletteConnFrom = null;
          case 'selectAll':
            _selected.clear();
            for (final n in _nodes) {
              _selected.add(n['id'] as String);
            }
          case 'fitView':
            _fitToView();
        }
      });
    });
  }

  PopupMenuItem<String> _ctxItem(String val, String label, IconData icon,
      {Color color = _dimText}) {
    return PopupMenuItem(
      value: val,
      child: Row(children: [
        Icon(icon, size: 15, color: color),
        const SizedBox(width: 10),
        Text(label, style: TextStyle(color: color, fontSize: 13)),
      ]),
    );
  }

  // ── Fit to view ──
  void _fitToView() {
    if (_nodes.isEmpty) return;
    double minX = double.infinity, minY = double.infinity;
    double maxX = double.negativeInfinity, maxY = double.negativeInfinity;
    for (final n in _nodes) {
      final p = _nodePos(n);
      minX = min(minX, p.dx);
      minY = min(minY, p.dy);
      maxX = max(maxX, p.dx + _nodeW);
      maxY = max(maxY, p.dy + _nodeH);
    }
    final size = (context.findRenderObject() as RenderBox?)?.size ??
        const Size(800, 600);
    final contentW = maxX - minX + 100;
    final contentH = maxY - minY + 100;
    final z = min(size.width / contentW, (size.height - 52) / contentH)
        .clamp(0.2, 1.5);
    setState(() {
      _zoom = z;
      _pan = Offset(
        (size.width - contentW * z) / 2 - minX * z + 50 * z,
        ((size.height - 52) - contentH * z) / 2 - minY * z + 50 * z,
      );
    });
  }

  // ═══════════════════════════════════════════════════════════════════════════
  // Zoom controls (bottom-left)
  // ═══════════════════════════════════════════════════════════════════════════

  Widget _zoomControls() {
    return Positioned(
      left: 16,
      bottom: (_showLogs ? 180 : 0) + 16,
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 4),
        decoration: BoxDecoration(
          color: _surface,
          borderRadius: BorderRadius.circular(8),
          border: Border.all(color: _border),
        ),
        child: Row(mainAxisSize: MainAxisSize.min, children: [
          _zBtn(LucideIcons.maximize2, _fitToView),
          _zBtn(LucideIcons.minus, () {
            setState(() => _zoom = (_zoom * 0.8).clamp(0.15, 3.0));
          }),
          GestureDetector(
            onTap: () => setState(() => _zoom = 1.0),
            child: SizedBox(
              width: 48,
              child: Center(
                child: Text('${(_zoom * 100).round()}%',
                    style: const TextStyle(color: _dimText, fontSize: 11)),
              ),
            ),
          ),
          _zBtn(LucideIcons.plus, () {
            setState(() => _zoom = (_zoom * 1.25).clamp(0.15, 3.0));
          }),
          Container(width: 1, height: 18, color: _border),
          _zBtn(
              _showMinimap ? LucideIcons.map : LucideIcons.mapPin,
              () => setState(() => _showMinimap = !_showMinimap)),
          _zBtn(LucideIcons.stickyNote, _addStickyNote),
          Container(width: 1, height: 18, color: _border),
          GestureDetector(
            onTap: () => setState(() => _showLogs = !_showLogs),
            child: Padding(
              padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 6),
              child: Text('Logs',
                  style: TextStyle(
                      color: _showLogs ? _accent : _dimText, fontSize: 11)),
            ),
          ),
        ]),
      ),
    );
  }

  void _addStickyNote() {
    final pos = _toCanvas(Offset(
        MediaQuery.of(context).size.width / 2,
        MediaQuery.of(context).size.height / 2));
    setState(() {
      _stickyNotes.add({
        'id': 'note_${DateTime.now().millisecondsSinceEpoch}',
        'text': 'Note',
        'x': pos.dx,
        'y': pos.dy,
        'w': 200.0,
        'h': 100.0,
      });
    });
  }

  Widget _zBtn(IconData ic, VoidCallback onTap) => InkWell(
        onTap: onTap,
        borderRadius: BorderRadius.circular(4),
        child: Padding(
          padding: const EdgeInsets.all(6),
          child: Icon(ic, size: 14, color: _dimText),
        ),
      );

  // ═══════════════════════════════════════════════════════════════════════════
  // Node palette — n8n-style categorized with search & drill-down
  // ═══════════════════════════════════════════════════════════════════════════

  String? _palCategory; // null = top-level, non-null = drilled into category
  final _palSearchCtrl = TextEditingController();

  Widget _palette() {
    final query = _palSearchCtrl.text.trim().toLowerCase();
    final isSearch = query.isNotEmpty;

    return Positioned(
      right: 0,
      top: 0,
      bottom: 0,
      child: Container(
        width: 300,
        decoration: BoxDecoration(
            color: _surface,
            border: Border(
                left: BorderSide(color: Colors.white.withOpacity(0.06)))),
        child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
          // Header with back / title / close
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 16, 8, 0),
            child: Row(children: [
              if (_palCategory != null && !isSearch)
                GestureDetector(
                  onTap: () => setState(() => _palCategory = null),
                  child: const Padding(
                    padding: EdgeInsets.only(right: 8),
                    child: Icon(LucideIcons.arrowLeft,
                        size: 16, color: _dimText),
                  ),
                ),
              if (_palCategory != null && !isSearch) ...[
                Icon(_categories
                        .firstWhere((c) => c.name == _palCategory,
                            orElse: () => _categories[0])
                        .icon,
                    size: 15, color: _dimText),
                const SizedBox(width: 8),
              ],
              Expanded(
                child: Text(
                  isSearch
                      ? 'Search results'
                      : _palCategory ?? 'What happens next?',
                  style: const TextStyle(
                      color: Colors.white,
                      fontSize: 14,
                      fontWeight: FontWeight.w600),
                ),
              ),
              IconButton(
                  onPressed: () => setState(() {
                        _showPalette = false;
                        _palCategory = null;
                        _palSearchCtrl.clear();
                      }),
                  icon: const Icon(LucideIcons.x, size: 16, color: _dimText)),
            ]),
          ),

          // Search bar
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 12, 16, 8),
            child: TextField(
              controller: _palSearchCtrl,
              autofocus: true,
              style: const TextStyle(color: Colors.white, fontSize: 13),
              onChanged: (_) => setState(() {}),
              decoration: InputDecoration(
                hintText: 'Search nodes...',
                hintStyle: const TextStyle(color: _subtleText, fontSize: 13),
                prefixIcon: const Icon(LucideIcons.search,
                    size: 15, color: _subtleText),
                filled: true,
                fillColor: _bg,
                contentPadding: const EdgeInsets.symmetric(vertical: 10),
                border: OutlineInputBorder(
                    borderRadius: BorderRadius.circular(8),
                    borderSide: const BorderSide(color: _border)),
                enabledBorder: OutlineInputBorder(
                    borderRadius: BorderRadius.circular(8),
                    borderSide: const BorderSide(color: _border)),
                focusedBorder: OutlineInputBorder(
                    borderRadius: BorderRadius.circular(8),
                    borderSide: const BorderSide(color: _accent)),
              ),
            ),
          ),
          Container(height: 1, color: _border),

          // Content
          Expanded(child: isSearch ? _palSearch(query) : _palContent()),
        ]),
      ),
    );
  }

  /// Top-level categories or drilled-in node list
  Widget _palContent() {
    if (_palCategory == null) {
      // Top-level category list
      return ListView(padding: const EdgeInsets.all(8), children: [
        ..._categories.map((c) => _palCategoryItem(c)),
        const SizedBox(height: 8),
        Padding(
          padding: const EdgeInsets.symmetric(horizontal: 12),
          child: GestureDetector(
            onTap: () => setState(() => _palCategory = 'Triggers'),
            child: Container(
              padding: const EdgeInsets.symmetric(vertical: 10, horizontal: 12),
              decoration: BoxDecoration(
                  color: Colors.white.withOpacity(0.03),
                  borderRadius: BorderRadius.circular(8)),
              child: Row(children: [
                const Icon(LucideIcons.zap, size: 15, color: _dimText),
                const SizedBox(width: 10),
                const Expanded(
                  child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text('Add another trigger',
                            style:
                                TextStyle(color: Colors.white, fontSize: 13)),
                        Text('Workflows can have multiple triggers',
                            style:
                                TextStyle(color: _subtleText, fontSize: 11)),
                      ]),
                ),
                const Icon(LucideIcons.chevronRight,
                    size: 14, color: _subtleText),
              ]),
            ),
          ),
        ),
      ]);
    }

    // Drilled into a category
    final items = _allNodeDefs
        .where((d) => d.category == _palCategory)
        .toList();

    // Group into subcategories for Data transformation
    if (_palCategory == 'Data transformation') {
      return ListView(padding: const EdgeInsets.all(8), children: [
        _palSection('Popular'),
        ..._palFilterList(items, ['edit_fields', 'code', 'date_time', 'aggregate']),
        _palSection('Add or remove items'),
        ..._palFilterList(items, ['filter', 'limit', 'remove_duplicates', 'split_out']),
        _palSection('Combine items'),
        ..._palFilterList(items, ['aggregate', 'merge', 'summarize']),
        _palSection('Convert data'),
        ..._palFilterList(items, [
          'convert_to_json', 'extract_from_json', 'crypto', 'html_parse'
        ]),
      ]);
    }

    if (_palCategory == 'Flow') {
      return ListView(padding: const EdgeInsets.all(8), children: [
        _palSection('Popular'),
        ..._palFilterList(items, ['if_condition', 'switch', 'merge', 'loop']),
        _palSection('Error handling'),
        ..._palFilterList(items, ['try_catch', 'stop_and_error']),
        _palSection('Other'),
        ..._palFilterList(items, [
          'wait', 'filter', 'no_operation', 'execute_sub_workflow'
        ]),
      ]);
    }

    if (_palCategory == 'Integrations') {
      return ListView(padding: const EdgeInsets.all(8), children: [
        _palSection('Communication'),
        ..._palFilterList(items, ['slack', 'discord', 'telegram', 'sendgrid', 'twilio_sms']),
        _palSection('Developer tools'),
        ..._palFilterList(items, ['github', 'jira', 'notion']),
        _palSection('Databases'),
        ..._palFilterList(items, ['postgres_query', 'mysql_query', 'redis_command']),
        _palSection('Cloud & payments'),
        ..._palFilterList(items, ['s3', 'stripe', 'google_sheets']),
      ]);
    }

    if (_palCategory == 'AI') {
      return ListView(padding: const EdgeInsets.all(8), children: [
        _palSection('Transform & generate'),
        ..._palFilterList(items, ['ai_transform', 'ai_summarize']),
        _palSection('Agents'),
        ..._palFilterList(items, ['ai_agent']),
      ]);
    }

    return ListView(
      padding: const EdgeInsets.all(8),
      children: items.map((d) => _palNodeItem(d)).toList(),
    );
  }

  /// Search results
  Widget _palSearch(String query) {
    final matches = _allNodeDefs
        .where((d) =>
            d.label.toLowerCase().contains(query) ||
            d.description.toLowerCase().contains(query) ||
            d.type.toLowerCase().contains(query) ||
            d.category.toLowerCase().contains(query))
        .toList();
    if (matches.isEmpty) {
      return const Center(
          child:
              Text('No nodes found', style: TextStyle(color: _subtleText)));
    }
    return ListView(
      padding: const EdgeInsets.all(8),
      children: matches.map((d) => _palNodeItem(d)).toList(),
    );
  }

  List<Widget> _palFilterList(
      List<_NodeDef> all, List<String> types) {
    return types
        .map((t) => all.where((d) => d.type == t).firstOrNull)
        .whereType<_NodeDef>()
        .map((d) => _palNodeItem(d))
        .toList();
  }

  Widget _palSection(String title) => Padding(
        padding: const EdgeInsets.fromLTRB(12, 12, 12, 4),
        child: Text(title,
            style: TextStyle(
                color: Colors.white.withOpacity(0.35),
                fontSize: 11,
                fontWeight: FontWeight.w600)),
      );

  Widget _palCategoryItem(_Category c) => Padding(
        padding: const EdgeInsets.only(bottom: 2),
        child: Material(
          color: Colors.transparent,
          child: InkWell(
            borderRadius: BorderRadius.circular(8),
            onTap: () => setState(() => _palCategory = c.name),
            child: Padding(
              padding:
                  const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
              child: Row(children: [
                Icon(c.icon, size: 16, color: _dimText),
                const SizedBox(width: 12),
                Expanded(
                  child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(c.name,
                            style: const TextStyle(
                                color: Colors.white, fontSize: 13)),
                        Text(c.description,
                            style: const TextStyle(
                                color: _subtleText, fontSize: 11)),
                      ]),
                ),
                const Icon(LucideIcons.chevronRight,
                    size: 14, color: _subtleText),
              ]),
            ),
          ),
        ),
      );

  Widget _palNodeItem(_NodeDef d) => Padding(
        padding: const EdgeInsets.only(bottom: 2),
        child: Material(
          color: Colors.transparent,
          child: InkWell(
            borderRadius: BorderRadius.circular(8),
            onTap: () {
              setState(() {
                final pos = _palettePos ?? const Offset(400, 300);
                _addNode(d.type, pos, connectFrom: _paletteConnFrom);
                _showPalette = false;
                _palCategory = null;
                _palSearchCtrl.clear();
              });
            },
            child: Padding(
              padding:
                  const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
              child: Row(children: [
                Container(
                  width: 32,
                  height: 32,
                  decoration: BoxDecoration(
                      color: d.color.withOpacity(0.12),
                      borderRadius: BorderRadius.circular(7)),
                  child: Icon(d.icon, size: 15, color: d.color),
                ),
                const SizedBox(width: 12),
                Expanded(
                  child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(d.label,
                            style: const TextStyle(
                                color: Colors.white, fontSize: 13)),
                        Text(d.description,
                            style: const TextStyle(
                                color: _subtleText, fontSize: 11)),
                      ]),
                ),
                const Icon(LucideIcons.plus, size: 14, color: _subtleText),
              ]),
            ),
          ),
        ),
      );

  // ═══════════════════════════════════════════════════════════════════════════
  // Config panel (right side)
  // ═══════════════════════════════════════════════════════════════════════════

  Widget _configPanel() {
    final node = _byId(_configNodeId!);
    if (node == null) return const SizedBox.shrink();
    final d = _def(node['type'] ?? 'http_request');
    final isTrigger = node['type'] == 'trigger';
    final disabled = node['disabled'] == true;
    final config = Map<String, dynamic>.from(node['config'] ?? {});

    return Positioned(
      right: _showPalette ? 280 : 0,
      top: 0,
      bottom: 0,
      child: Container(
        width: 340,
        decoration: BoxDecoration(
            color: _surface,
            border: Border(
                left: BorderSide(color: Colors.white.withOpacity(0.06)))),
        child: Column(children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(20, 16, 12, 12),
            child: Row(children: [
              Container(
                width: 30,
                height: 30,
                decoration: BoxDecoration(
                    color: d.color.withOpacity(0.12),
                    borderRadius: BorderRadius.circular(7)),
                child: Icon(d.icon, size: 14, color: d.color),
              ),
              const SizedBox(width: 10),
              Expanded(
                  child: Text(d.label,
                      style: const TextStyle(
                          color: Colors.white,
                          fontSize: 14,
                          fontWeight: FontWeight.w600))),
              if (!isTrigger) ...[
                IconButton(
                  onPressed: () => setState(() {
                    _selected
                      ..clear()
                      ..add(node['id'] as String);
                    _toggleDisable();
                  }),
                  icon: Icon(
                      disabled ? LucideIcons.toggleLeft : LucideIcons.toggleRight,
                      size: 16,
                      color: disabled ? _red : _green),
                  tooltip: disabled ? 'Enable' : 'Disable',
                ),
                IconButton(
                  onPressed: () => setState(() {
                    _selected
                      ..clear()
                      ..add(node['id'] as String);
                    _deleteSelected();
                    _configNodeId = null;
                  }),
                  icon: const Icon(LucideIcons.trash2,
                      size: 15, color: Colors.redAccent),
                  tooltip: 'Delete',
                ),
              ],
              IconButton(
                  onPressed: () => setState(() => _configNodeId = null),
                  icon: const Icon(LucideIcons.x, size: 16, color: _dimText)),
            ]),
          ),
          // Settings / Input / Output tabs
          Container(
            decoration: BoxDecoration(
                border: Border(
                    bottom:
                        BorderSide(color: Colors.white.withOpacity(0.06)))),
            child: Row(children: [
              _cfgTabBtn('Settings', 0),
              _cfgTabBtn('Input', 1),
              _cfgTabBtn('Output', 2),
            ]),
          ),
          Expanded(
            child: _configTab == 0
                ? ListView(padding: const EdgeInsets.all(20), children: [
                    _cfgLabel('Label'),
                    const SizedBox(height: 6),
                    _cfgField(
                      value: node['label'] ?? '',
                      hint: 'Node label',
                      onChanged: (v) => setState(() {
                        node['label'] = v;
                        _dirty = true;
                      }),
                    ),
                    const SizedBox(height: 16),
                    if (!isTrigger) ..._typeFields(node, config),
                    const SizedBox(height: 20),
                    _cfgLabel('Connections'),
                    const SizedBox(height: 8),
                    ..._connList(node['id'] as String),
                  ])
                : _configTab == 1
                    ? _dataPreview(node['id'] as String, true)
                    : _dataPreview(node['id'] as String, false),
          ),
        ]),
      ),
    );
  }

  Widget _cfgTabBtn(String label, int idx) {
    final active = _configTab == idx;
    return Expanded(
      child: GestureDetector(
        onTap: () => setState(() => _configTab = idx),
        child: Container(
          padding: const EdgeInsets.symmetric(vertical: 10),
          decoration: BoxDecoration(
            border: Border(
              bottom: BorderSide(
                color: active ? _accent : Colors.transparent,
                width: 2,
              ),
            ),
          ),
          child: Center(
            child: Text(label,
                style: TextStyle(
                    color: active ? Colors.white : _subtleText,
                    fontSize: 12,
                    fontWeight: active ? FontWeight.w500 : FontWeight.w400)),
          ),
        ),
      ),
    );
  }

  Widget _dataPreview(String nodeId, bool isInput) {
    final Map<String, dynamic>? data = _lastExecData != null
        ? (_lastExecData![nodeId] as Map<String, dynamic>?)
        : null;
    final pinned = _pinnedData != null ? _pinnedData![nodeId] : null;
    final dynamic display = isInput
        ? (data != null ? data['input'] : null)
        : (pinned ?? (data != null ? data['output'] : null));

    if (display == null) {
      return Center(
        child: Padding(
          padding: const EdgeInsets.all(32),
          child: Column(mainAxisSize: MainAxisSize.min, children: [
            Icon(isInput ? LucideIcons.arrowDownToLine : LucideIcons.arrowUpFromLine,
                size: 24, color: Colors.white.withOpacity(0.1)),
            const SizedBox(height: 12),
            Text(
              'Execute the workflow to see ${isInput ? "input" : "output"} data',
              textAlign: TextAlign.center,
              style: TextStyle(
                  color: Colors.white.withOpacity(0.3), fontSize: 12),
            ),
          ]),
        ),
      );
    }

    final jsonStr = _prettyJson(display);

    return Column(children: [
      if (!isInput && data?['output'] != null)
        Padding(
          padding: const EdgeInsets.fromLTRB(12, 8, 12, 0),
          child: Row(children: [
            if (pinned != null)
              Container(
                padding:
                    const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
                decoration: BoxDecoration(
                    color: _orange.withOpacity(0.15),
                    borderRadius: BorderRadius.circular(4)),
                child: const Text('PINNED',
                    style: TextStyle(
                        color: _orange,
                        fontSize: 9,
                        fontWeight: FontWeight.w600)),
              ),
            const Spacer(),
            GestureDetector(
              onTap: () => setState(() {
                _pinnedData ??= {};
                if (_pinnedData!.containsKey(nodeId)) {
                  _pinnedData!.remove(nodeId);
                } else {
                  _pinnedData![nodeId] = data?['output'];
                }
              }),
              child: Icon(
                  pinned != null ? LucideIcons.pinOff : LucideIcons.pin,
                  size: 14,
                  color: pinned != null ? _orange : _subtleText),
            ),
          ]),
        ),
      Expanded(
        child: SingleChildScrollView(
          padding: const EdgeInsets.all(12),
          child: SelectableText(
            jsonStr,
            style: TextStyle(
              color: Colors.white.withOpacity(0.7),
              fontSize: 11,
              fontFamily: 'monospace',
            ),
          ),
        ),
      ),
    ]);
  }

  String _prettyJson(dynamic data) {
    try {
      const encoder = JsonEncoder.withIndent('  ');
      return encoder.convert(data);
    } catch (_) {
      return '$data';
    }
  }

  List<Widget> _typeFields(
      Map<String, dynamic> node, Map<String, dynamic> config) {
    final t = node['type'] as String? ?? '';
    final i = _nodes.indexOf(node);
    if (i < 0) return [];
    switch (t) {
      case 'http_request':
        return [
          _tf(i, config, 'url', 'URL', hint: 'https://api.example.com'),
          _tf(i, config, 'method', 'Method', hint: 'GET'),
          _tf(i, config, 'body', 'Body (JSON)', hint: '{}', lines: 3),
        ];
      case 'send_email':
        return [
          _tf(i, config, 'to', 'To'),
          _tf(i, config, 'subject', 'Subject'),
          _tf(i, config, 'body', 'Body', lines: 3),
        ];
      case 'set_variable':
        return [
          _tf(i, config, 'key', 'Key'),
          _tf(i, config, 'value', 'Value'),
        ];
      case 'code':
        return [
          _tf(i, config, 'expression', 'Expression',
              hint: '{{.trigger.name}}', lines: 4),
        ];
      case 'if_condition':
        return [
          _tf(i, config, 'field', 'Field', hint: 'trigger.status'),
          _tf(i, config, 'operator', 'Operator', hint: 'eq, neq, contains'),
          _tf(i, config, 'value', 'Value'),
        ];
      case 'delay':
        return [
          _tf(i, config, 'durationMs', 'Duration (ms)', hint: '1000'),
        ];
      case 'switch':
        return [
          _tf(i, config, 'field', 'Field', hint: 'trigger.status'),
          _tf(i, config, 'case1', 'Case 1 value'),
          _tf(i, config, 'case2', 'Case 2 value'),
          _tf(i, config, 'defaultTarget', 'Default target node ID'),
        ];
      case 'merge':
        return [];
      case 'loop':
        return [
          _tf(i, config, 'items', 'Items field', hint: 'trigger.items'),
          _tf(i, config, 'loopVariable', 'Loop variable name', hint: 'item'),
        ];
      case 'wait':
        return [
          _tf(i, config, 'seconds', 'Wait seconds', hint: '5'),
        ];
      case 'no_operation':
        return [];
      case 'execute_sub_workflow':
        return [
          _tf(i, config, 'workflowId', 'Sub-workflow ID'),
        ];
      case 'filter':
        return [
          _tf(i, config, 'field', 'Field', hint: 'trigger.status'),
          _tf(i, config, 'operator', 'Operator', hint: 'eq, neq, contains'),
          _tf(i, config, 'value', 'Value'),
        ];
      case 'edit_fields':
        return [
          _tf(i, config, 'fields', 'Fields (JSON)',
              hint: '{"key": "value"}', lines: 4),
        ];
      case 'aggregate':
        return [
          _tf(i, config, 'field', 'Array field', hint: 'trigger.items'),
          _tf(i, config, 'operation', 'Operation',
              hint: 'count, sum, min, max, avg'),
        ];
      case 'summarize':
        return [
          _tf(i, config, 'field', 'Field', hint: 'trigger.items'),
          _tf(i, config, 'groupBy', 'Group by', hint: 'status'),
        ];
      case 'limit':
        return [
          _tf(i, config, 'count', 'Max items', hint: '10'),
        ];
      case 'split_out':
        return [
          _tf(i, config, 'field', 'Array field', hint: 'trigger.items'),
        ];
      case 'remove_duplicates':
        return [
          _tf(i, config, 'field', 'Dedup by field', hint: 'email'),
        ];
      case 'date_time':
        return [
          _tf(i, config, 'operation', 'Operation',
              hint: 'now, format, parse, add'),
          _tf(i, config, 'format', 'Format', hint: '2006-01-02T15:04:05Z'),
          _tf(i, config, 'value', 'Value'),
          _tf(i, config, 'duration', 'Duration', hint: '24h'),
        ];
      case 'convert_to_json':
        return [
          _tf(i, config, 'data', 'Data field', hint: 'trigger.payload'),
        ];
      case 'extract_from_json':
        return [
          _tf(i, config, 'json', 'JSON string field'),
          _tf(i, config, 'path', 'JSON path', hint: 'data.name'),
        ];
      case 'html_parse':
        return [
          _tf(i, config, 'html', 'HTML content'),
          _tf(i, config, 'selector', 'Selector'),
        ];
      case 'crypto':
        return [
          _tf(i, config, 'operation', 'Operation',
              hint: 'md5, sha256, base64_encode, base64_decode'),
          _tf(i, config, 'input', 'Input'),
        ];
      case 'slack':
        return [
          _tf(i, config, 'webhookUrl', 'Webhook URL'),
          _tf(i, config, 'message', 'Message', lines: 3),
        ];
      case 'discord':
        return [
          _tf(i, config, 'webhookUrl', 'Webhook URL'),
          _tf(i, config, 'message', 'Message', lines: 3),
        ];
      case 'telegram':
        return [
          _tf(i, config, 'botToken', 'Bot Token'),
          _tf(i, config, 'chatId', 'Chat ID'),
          _tf(i, config, 'message', 'Message', lines: 3),
        ];
      case 'github':
        return [
          _tf(i, config, 'token', 'Personal Access Token'),
          _tf(i, config, 'owner', 'Owner'),
          _tf(i, config, 'repo', 'Repository'),
          _tf(i, config, 'action', 'Action', hint: 'create_issue'),
          _tf(i, config, 'title', 'Title'),
          _tf(i, config, 'body', 'Body', lines: 3),
        ];
      case 'javascript':
        return [
          _tf(i, config, 'code', 'Code',
              hint: '{{json_stringify .trigger}}', lines: 8),
        ];
      // ── Error handling ──
      case 'try_catch':
        return [
          _tf(i, config, 'tryNodes', 'Try node IDs (comma-separated)'),
          _tf(i, config, 'catchTarget', 'Catch target node ID'),
        ];
      case 'stop_and_error':
        return [
          _tf(i, config, 'message', 'Error message', hint: 'Something went wrong'),
        ];
      // ── AI ──
      case 'ai_transform':
        return [
          _tf(i, config, 'model', 'Model', hint: 'claude-sonnet-4-20250514'),
          _tf(i, config, 'prompt', 'Prompt', hint: 'Transform this data...', lines: 4),
          _tf(i, config, 'apiKey', 'API Key'),
        ];
      case 'ai_agent':
        return [
          _tf(i, config, 'model', 'Model', hint: 'claude-sonnet-4-20250514'),
          _tf(i, config, 'systemPrompt', 'System prompt', lines: 3),
          _tf(i, config, 'userMessage', 'User message', hint: '{{.trigger.body.message}}'),
          _tf(i, config, 'tools', 'Tools (JSON)', hint: '[]', lines: 3),
          _tf(i, config, 'apiKey', 'API Key'),
          _tf(i, config, 'maxSteps', 'Max steps', hint: '5'),
        ];
      case 'ai_summarize':
        return [
          _tf(i, config, 'model', 'Model', hint: 'claude-sonnet-4-20250514'),
          _tf(i, config, 'text', 'Text to summarize', lines: 3),
          _tf(i, config, 'maxLength', 'Max length', hint: '200'),
          _tf(i, config, 'apiKey', 'API Key'),
        ];
      // ── New integrations ──
      case 'google_sheets':
        return [
          _tf(i, config, 'accessToken', 'Access Token'),
          _tf(i, config, 'spreadsheetId', 'Spreadsheet ID'),
          _tf(i, config, 'range', 'Range', hint: 'Sheet1!A1:D10'),
          _tf(i, config, 'action', 'Action', hint: 'read or append'),
          _tf(i, config, 'values', 'Values (JSON)', hint: '[[]]', lines: 2),
        ];
      case 'notion':
        return [
          _tf(i, config, 'apiKey', 'Integration Token'),
          _tf(i, config, 'action', 'Action', hint: 'query_database or create_page'),
          _tf(i, config, 'databaseId', 'Database ID'),
          _tf(i, config, 'properties', 'Properties (JSON)', hint: '{}', lines: 3),
        ];
      case 'stripe':
        return [
          _tf(i, config, 'apiKey', 'Secret Key'),
          _tf(i, config, 'action', 'Action', hint: 'create_charge'),
          _tf(i, config, 'amount', 'Amount (cents)'),
          _tf(i, config, 'currency', 'Currency', hint: 'usd'),
          _tf(i, config, 'email', 'Customer email'),
        ];
      case 'twilio_sms':
        return [
          _tf(i, config, 'accountSid', 'Account SID'),
          _tf(i, config, 'authToken', 'Auth Token'),
          _tf(i, config, 'from', 'From number'),
          _tf(i, config, 'to', 'To number'),
          _tf(i, config, 'body', 'Message', lines: 2),
        ];
      case 'postgres_query':
      case 'mysql_query':
        return [
          _tf(i, config, 'connectionUrl', 'Connection URL'),
          _tf(i, config, 'query', 'SQL Query', lines: 4),
        ];
      case 'redis_command':
        return [
          _tf(i, config, 'connectionUrl', 'Connection URL'),
          _tf(i, config, 'command', 'Command', hint: 'GET mykey'),
        ];
      case 's3':
        return [
          _tf(i, config, 'accessKeyId', 'Access Key ID'),
          _tf(i, config, 'secretAccessKey', 'Secret Access Key'),
          _tf(i, config, 'region', 'Region', hint: 'us-east-1'),
          _tf(i, config, 'bucket', 'Bucket'),
          _tf(i, config, 'key', 'Object Key'),
          _tf(i, config, 'action', 'Action', hint: 'get, put, or list'),
        ];
      case 'sendgrid':
        return [
          _tf(i, config, 'apiKey', 'API Key'),
          _tf(i, config, 'to', 'To'),
          _tf(i, config, 'from', 'From'),
          _tf(i, config, 'subject', 'Subject'),
          _tf(i, config, 'body', 'Body', lines: 3),
        ];
      case 'jira':
        return [
          _tf(i, config, 'domain', 'Domain', hint: 'yourcompany'),
          _tf(i, config, 'email', 'Email'),
          _tf(i, config, 'apiToken', 'API Token'),
          _tf(i, config, 'action', 'Action', hint: 'create_issue'),
          _tf(i, config, 'projectKey', 'Project Key'),
          _tf(i, config, 'summary', 'Summary'),
          _tf(i, config, 'description', 'Description', lines: 3),
        ];
      // ── Applad-native ──
      case 'applad_auth':
        return [
          _tf(i, config, 'action', 'Action', hint: 'create_user, get_user, list_users, update_user, delete_user'),
          _tf(i, config, 'email', 'Email'),
          _tf(i, config, 'password', 'Password'),
          _tf(i, config, 'name', 'Name'),
          _tf(i, config, 'userId', 'User ID'),
        ];
      case 'applad_database':
        return [
          _tf(i, config, 'action', 'Action', hint: 'create_document, get_document, list_documents, update_document, delete_document'),
          _tf(i, config, 'databaseId', 'Database ID'),
          _tf(i, config, 'collectionId', 'Collection ID'),
          _tf(i, config, 'documentId', 'Document ID'),
          _tf(i, config, 'data', 'Data (JSON)', hint: '{}', lines: 4),
        ];
      case 'applad_storage':
        return [
          _tf(i, config, 'action', 'Action', hint: 'list_files, get_file, delete_file'),
          _tf(i, config, 'bucketId', 'Bucket ID'),
          _tf(i, config, 'fileId', 'File ID'),
        ];
      case 'applad_functions':
        return [
          _tf(i, config, 'action', 'Action', hint: 'invoke, list_executions'),
          _tf(i, config, 'targetId', 'Target ID'),
          _tf(i, config, 'data', 'Request Data (JSON)', hint: '{}', lines: 3),
        ];
      case 'applad_messaging':
        return [
          _tf(i, config, 'action', 'Action', hint: 'send_email, send_sms, send_push'),
          _tf(i, config, 'to', 'To'),
          _tf(i, config, 'subject', 'Subject'),
          _tf(i, config, 'body', 'Body', lines: 3),
          _tf(i, config, 'title', 'Title (push only)'),
        ];
      // ── Additional flow ──
      case 'sort':
        return [
          _tf(i, config, 'items', 'Items field', hint: 'trigger.items'),
          _tf(i, config, 'field', 'Sort by field'),
          _tf(i, config, 'order', 'Order', hint: 'asc or desc'),
        ];
      case 'rename_keys':
        return [
          _tf(i, config, 'mapping', 'Mapping (JSON)', hint: '{"oldKey": "newKey"}', lines: 3),
        ];
      case 'compare_datasets':
        return [
          _tf(i, config, 'input1', 'Input 1 field', hint: 'trigger.oldData'),
          _tf(i, config, 'input2', 'Input 2 field', hint: 'trigger.newData'),
          _tf(i, config, 'keyField', 'Key field', hint: 'id'),
        ];
      default:
        return [];
    }
  }

  Widget _tf(int ni, Map<String, dynamic> cfg, String key, String label,
      {String? hint, int lines = 1}) {
    return Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
      _cfgLabel(label),
      const SizedBox(height: 6),
      _cfgField(
        value: '${cfg[key] ?? ''}',
        hint: hint ?? '',
        maxLines: lines,
        onChanged: (v) => setState(() {
          final c = Map<String, dynamic>.from(_nodes[ni]['config'] ?? {});
          c[key] = v;
          _nodes[ni]['config'] = c;
          _dirty = true;
        }),
      ),
      const SizedBox(height: 16),
    ]);
  }

  Widget _cfgLabel(String t) => Text(t,
      style: TextStyle(
          color: Colors.white.withOpacity(0.5),
          fontSize: 12,
          fontWeight: FontWeight.w500));

  Widget _cfgField(
      {required String value,
      required String hint,
      int maxLines = 1,
      required ValueChanged<String> onChanged}) {
    return TextField(
      controller: TextEditingController(text: value),
      maxLines: maxLines,
      style: const TextStyle(color: Colors.white, fontSize: 13),
      decoration: InputDecoration(
        hintText: hint,
        hintStyle: const TextStyle(color: _subtleText, fontSize: 13),
        filled: true,
        fillColor: _bg,
        contentPadding:
            const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
        border: OutlineInputBorder(
            borderRadius: BorderRadius.circular(6),
            borderSide: const BorderSide(color: _border)),
        enabledBorder: OutlineInputBorder(
            borderRadius: BorderRadius.circular(6),
            borderSide: const BorderSide(color: _border)),
        focusedBorder: OutlineInputBorder(
            borderRadius: BorderRadius.circular(6),
            borderSide: const BorderSide(color: _accent)),
      ),
      onChanged: onChanged,
    );
  }

  List<Widget> _connList(String id) {
    final ins = _edges.where((e) => e['target'] == id).toList();
    final outs = _edges.where((e) => e['source'] == id).toList();
    if (ins.isEmpty && outs.isEmpty) {
      return [
        const Text('No connections',
            style: TextStyle(color: _subtleText, fontSize: 12))
      ];
    }
    return [
      ...ins.map((e) {
        final src = _byId(e['source'] as String);
        return _connChip('From: ${src?['label'] ?? '?'}', LucideIcons.arrowLeft,
            () => setState(() {
              _pushUndo();
              _edges.removeWhere((x) => x['id'] == e['id']);
              _dirty = true;
            }));
      }),
      ...outs.map((e) {
        final tgt = _byId(e['target'] as String);
        return _connChip('To: ${tgt?['label'] ?? '?'}', LucideIcons.arrowRight,
            () => setState(() {
              _pushUndo();
              _edges.removeWhere((x) => x['id'] == e['id']);
              _dirty = true;
            }));
      }),
    ];
  }

  Widget _connChip(String label, IconData icon, VoidCallback onDel) => Padding(
        padding: const EdgeInsets.only(bottom: 6),
        child: Container(
          padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
          decoration: BoxDecoration(
              color: _bg,
              borderRadius: BorderRadius.circular(6),
              border: Border.all(color: _border)),
          child: Row(children: [
            Icon(icon, size: 13, color: _subtleText),
            const SizedBox(width: 8),
            Expanded(
                child: Text(label,
                    style: const TextStyle(color: _dimText, fontSize: 12))),
            GestureDetector(
                onTap: onDel,
                child: const Icon(LucideIcons.x, size: 12, color: _subtleText)),
          ]),
        ),
      );

  // ═══════════════════════════════════════════════════════════════════════════
  // Top tabs (Editor / Executions)
  // ═══════════════════════════════════════════════════════════════════════════

  Widget _topTabs() {
    return Container(
      height: 40,
      decoration: BoxDecoration(
          color: _bg,
          border: Border(
              bottom: BorderSide(color: Colors.white.withOpacity(0.06)))),
      child: Row(children: [
        // Tags (left side)
        const SizedBox(width: 16),
        if (_tags.isNotEmpty)
          ..._tags.map((t) => Padding(
                padding: const EdgeInsets.only(right: 6),
                child: Container(
                  padding:
                      const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
                  decoration: BoxDecoration(
                      color: _accent.withOpacity(0.12),
                      borderRadius: BorderRadius.circular(4)),
                  child: Row(mainAxisSize: MainAxisSize.min, children: [
                    Text(t,
                        style:
                            const TextStyle(color: _accent, fontSize: 11)),
                    const SizedBox(width: 4),
                    GestureDetector(
                      onTap: () => setState(() => _tags.remove(t)),
                      child: const Icon(LucideIcons.x,
                          size: 10, color: _accent),
                    ),
                  ]),
                ),
              )),
        GestureDetector(
          onTap: _addTag,
          child: const Text('+ Add tag',
              style: TextStyle(color: _subtleText, fontSize: 11)),
        ),
        const Spacer(),
        const SizedBox(width: 16),
      ]),
    );
  }


  void _addTag() {
    final ctrl = TextEditingController();
    showAppDialog(
      context: context,
      title: 'Add tag',
      width: 340,
      content: AppDialogField(
        controller: ctrl,
        hint: 'Tag name',
        autofocus: true,
      ),
      actions: [
        const AppDialogCancel(),
        AppDialogAction(
          label: 'Add',
          onTap: () {
            if (ctrl.text.trim().isNotEmpty) {
              setState(() => _tags.add(ctrl.text.trim()));
            }
            Navigator.pop(context);
          },
        ),
      ],
    );
  }

  // ═══════════════════════════════════════════════════════════════════════════
  // Logs panel (bottom)
  // ═══════════════════════════════════════════════════════════════════════════

  Widget _logsPanel() {
    return Container(
      height: 180,
      decoration: BoxDecoration(
          color: _surface,
          border: Border(
              top: BorderSide(color: Colors.white.withOpacity(0.06)))),
      child: Column(children: [
        Padding(
          padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
          child: Row(children: [
            const Icon(LucideIcons.terminal, size: 14, color: _dimText),
            const SizedBox(width: 8),
            const Text('Logs',
                style: TextStyle(
                    color: Colors.white,
                    fontSize: 13,
                    fontWeight: FontWeight.w600)),
            const Spacer(),
            IconButton(
                onPressed: () => setState(() => _showLogs = false),
                icon: const Icon(LucideIcons.x, size: 14, color: _dimText)),
          ]),
        ),
        Container(height: 1, color: _border),
        Expanded(
          child: _execStatus == null
              ? const Center(
                  child: Text('Execute workflow to see logs',
                      style: TextStyle(color: _subtleText, fontSize: 12)))
              : ListView(
                  padding: const EdgeInsets.all(12),
                  children: _execStatus!.entries.map((e) {
                    final node = _byId(e.key);
                    return Padding(
                      padding: const EdgeInsets.only(bottom: 4),
                      child: Row(children: [
                        Icon(
                            e.value == 'completed'
                                ? LucideIcons.checkCircle2
                                : e.value == 'failed'
                                    ? LucideIcons.xCircle
                                    : LucideIcons.loader2,
                            size: 13,
                            color: e.value == 'completed'
                                ? _green
                                : e.value == 'failed'
                                    ? _red
                                    : _accent),
                        const SizedBox(width: 8),
                        Text('${node?['label'] ?? e.key}: ${e.value}',
                            style: const TextStyle(
                                color: _dimText,
                                fontSize: 12,
                                fontFamily: 'monospace')),
                      ]),
                    );
                  }).toList(),
                ),
        ),
      ]),
    );
  }

  // ═══════════════════════════════════════════════════════════════════════════
  // Minimap (bottom-right)
  // ═══════════════════════════════════════════════════════════════════════════

  Widget _minimap() {
    return Positioned(
      right: (_showPalette ? 300 : 0) + (_configNodeId != null ? 340 : 0) + 16,
      bottom: (_showLogs ? 180 : 0) + 16,
      child: Container(
        width: 160,
        height: 100,
        decoration: BoxDecoration(
          color: _surface.withOpacity(0.9),
          borderRadius: BorderRadius.circular(8),
          border: Border.all(color: _border),
        ),
        child: ClipRRect(
          borderRadius: BorderRadius.circular(8),
          child: CustomPaint(
            painter: _MinimapPainter(
              nodes: _nodes,
              pan: _pan,
              zoom: _zoom,
              viewSize: MediaQuery.of(context).size,
            ),
          ),
        ),
      ),
    );
  }

  // ═══════════════════════════════════════════════════════════════════════════
  // Executions tab (full panel, not dialog)
  // ═══════════════════════════════════════════════════════════════════════════

  Widget _executionsTab() {
    return FutureBuilder<dynamic>(
      future: ref
          .read(apiClientProvider)
          .get('/workflows/$_wfId/executions')
          .then((r) => r.data),
      builder: (ctx, snap) {
        if (snap.connectionState == ConnectionState.waiting) {
          return const Center(
              child: CircularProgressIndicator(color: _accent));
        }
        if (snap.hasError) {
          return Center(
              child: Text('${snap.error}',
                  style: const TextStyle(color: _dimText)));
        }
        final execs = List<Map<String, dynamic>>.from(
            (snap.data as Map)['executions'] ?? []);
        if (execs.isEmpty) {
          return const Center(
              child: Text('No executions yet',
                  style: TextStyle(color: _dimText, fontSize: 14)));
        }
        return ListView.separated(
          padding: const EdgeInsets.all(24),
          itemCount: execs.length,
          separatorBuilder: (_, __) => const SizedBox(height: 8),
          itemBuilder: (ctx, i) {
            final e = execs[i];
            final st = e['status'] ?? 'pending';
            final dur = e['durationMs'] ?? 0;
            final logs =
                List<Map<String, dynamic>>.from(e['logs'] ?? []);
            final idStr = (e['\$id'] ?? '').toString();
            final sc = st == 'completed'
                ? _green
                : st == 'failed'
                    ? _red
                    : _accent;
            return Container(
              decoration: BoxDecoration(
                  color: _surface,
                  borderRadius: BorderRadius.circular(8),
                  border: Border.all(color: _border)),
              child: ExpansionTile(
                leading: Icon(
                    st == 'completed'
                        ? LucideIcons.checkCircle2
                        : st == 'failed'
                            ? LucideIcons.xCircle
                            : LucideIcons.clock,
                    size: 18,
                    color: sc),
                title: Text(
                    idStr.length > 12 ? idStr.substring(0, 12) : idStr,
                    style: const TextStyle(
                        color: Colors.white, fontSize: 13)),
                subtitle: Text('$st  •  ${dur}ms',
                    style:
                        const TextStyle(color: _subtleText, fontSize: 11)),
                children: logs.map((l) {
                  return ListTile(
                    dense: true,
                    leading: Icon(
                        l['status'] == 'completed'
                            ? LucideIcons.check
                            : l['status'] == 'skipped'
                                ? LucideIcons.skipForward
                                : LucideIcons.x,
                        size: 14,
                        color: l['status'] == 'completed'
                            ? _green
                            : l['status'] == 'skipped'
                                ? _dimText
                                : _red),
                    title: Text(
                        '${l['label']} (${l['nodeType']})',
                        style: const TextStyle(
                            color: Colors.white, fontSize: 12)),
                    subtitle: Text('${l['durationMs']}ms',
                        style: const TextStyle(
                            color: _subtleText, fontSize: 11)),
                  );
                }).toList(),
              ),
            );
          },
        );
      },
    );
  }

  // ═══════════════════════════════════════════════════════════════════════════
  // Executions dialog (kept for toolbar History button)
  // ═══════════════════════════════════════════════════════════════════════════

  void _showExecs() {
    showDialog(
      context: context,
      builder: (_) => Dialog(
        backgroundColor: _surface,
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(8)),
        child: SizedBox(
          width: 560,
          height: 480,
          child: Column(children: [
            Container(
              padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 14),
              decoration: BoxDecoration(
                  border: Border(
                      bottom: BorderSide(
                          color: Colors.white.withOpacity(0.06)))),
              child: Row(children: [
                const Text('Execution History',
                    style: TextStyle(
                        color: Colors.white,
                        fontSize: 15,
                        fontWeight: FontWeight.w600)),
                const Spacer(),
                IconButton(
                    onPressed: () => Navigator.pop(context),
                    icon: const Icon(LucideIcons.x,
                        size: 16, color: _dimText)),
              ]),
            ),
            Expanded(child: _executionsTab()),
          ]),
        ),
      ),
    );
  }
}

// ═════════════════════════════════════════════════════════════════════════════
// Canvas painter — draws everything on a single CustomPaint
// ═════════════════════════════════════════════════════════════════════════════

class _CanvasPainter extends CustomPainter {
  final Offset pan;
  final double zoom;
  final List<Map<String, dynamic>> nodes, edges;
  final Set<String> selected;
  final Map<String, dynamic>? connFrom;
  final Offset? connCursor;
  final Offset? rectStart, rectEnd;
  final Map<String, String>? execStatus;
  final Map<String, int>? execCounts;
  final List<Map<String, dynamic>> stickyNotes;

  _CanvasPainter({
    required this.pan,
    required this.zoom,
    required this.nodes,
    required this.edges,
    required this.selected,
    this.connFrom,
    this.connCursor,
    this.rectStart,
    this.rectEnd,
    this.execStatus,
    this.execCounts,
    this.stickyNotes = const [],
  });

  Offset _t(Offset o) => o * zoom + pan; // canvas → screen
  double _s(double v) => v * zoom;

  Offset _nodePos(Map<String, dynamic> n) {
    final p = n['position'] as Map<String, dynamic>?;
    return Offset(
        (p?['x'] as num?)?.toDouble() ?? 0, (p?['y'] as num?)?.toDouble() ?? 0);
  }

  Offset _outH(Map<String, dynamic> n, [int oi = 0]) {
    final d = _def(n['type'] ?? '');
    if (d.outputs <= 1) return _nodePos(n) + const Offset(_nodeW, _nodeH / 2);
    final spacing = _nodeH / (d.outputs + 1);
    return _nodePos(n) + Offset(_nodeW, spacing * (oi + 1));
  }

  Offset _inH(Map<String, dynamic> n, [int ii = 0]) {
    final d = _def(n['type'] ?? '');
    if (d.inputs <= 1) return _nodePos(n) + const Offset(0, _nodeH / 2);
    final spacing = _nodeH / (d.inputs + 1);
    return _nodePos(n) + Offset(0, spacing * (ii + 1));
  }

  Map<String, dynamic>? _byId(String id) =>
      nodes.where((n) => n['id'] == id).firstOrNull;

  @override
  void paint(Canvas canvas, Size size) {
    canvas.save();

    // Grid dots
    _drawGrid(canvas, size);

    // Sticky notes (behind everything)
    for (final note in stickyNotes) {
      final x = (note['x'] as num?)?.toDouble() ?? 0;
      final y = (note['y'] as num?)?.toDouble() ?? 0;
      final w = (note['w'] as num?)?.toDouble() ?? 200;
      final h = (note['h'] as num?)?.toDouble() ?? 100;
      final sp = _t(Offset(x, y));
      final rect = RRect.fromRectAndRadius(
          Rect.fromLTWH(sp.dx, sp.dy, _s(w), _s(h)),
          Radius.circular(_s(6)));
      canvas.drawRRect(rect, Paint()..color = const Color(0x30F59E0B));
      canvas.drawRRect(
          rect,
          Paint()
            ..color = const Color(0x50F59E0B)
            ..style = PaintingStyle.stroke
            ..strokeWidth = _s(1));
      final tp = TextPainter(
        text: TextSpan(
            text: note['text'] ?? '',
            style: TextStyle(
                color: const Color(0xAAF59E0B), fontSize: _s(12))),
        textDirection: TextDirection.ltr,
      )..layout(maxWidth: _s(w - 16));
      tp.paint(canvas, Offset(sp.dx + _s(8), sp.dy + _s(8)));
    }

    // Edges
    _drawEdges(canvas);

    // Dragging connection
    if (connFrom != null && connCursor != null) {
      _drawBezier(canvas, _t(_outH(connFrom!)), _t(connCursor!),
          Paint()
            ..color = _accent.withOpacity(0.6)
            ..strokeWidth = 2 * zoom
            ..style = PaintingStyle.stroke);
    }

    // Nodes
    for (final n in nodes) {
      _drawNode(canvas, n);
    }

    // Selection rectangle
    if (rectStart != null && rectEnd != null) {
      final r = Rect.fromPoints(_t(rectStart!), _t(rectEnd!));
      canvas.drawRect(
          r,
          Paint()
            ..color = _accent.withOpacity(0.08)
            ..style = PaintingStyle.fill);
      canvas.drawRect(
          r,
          Paint()
            ..color = _accent.withOpacity(0.4)
            ..style = PaintingStyle.stroke
            ..strokeWidth = 1);
    }

    canvas.restore();
  }

  void _drawGrid(Canvas canvas, Size size) {
    final p = Paint()..color = _gridDot;
    final spacing = _s(24);
    if (spacing < 6) return; // Too zoomed out
    final offX = pan.dx % spacing;
    final offY = pan.dy % spacing;
    for (double x = offX; x < size.width; x += spacing) {
      for (double y = offY; y < size.height; y += spacing) {
        canvas.drawCircle(Offset(x, y), zoom.clamp(0.5, 1.5), p);
      }
    }
  }

  void _drawEdges(Canvas canvas) {
    final paint = Paint()
      ..color = const Color(0x50FFFFFF)
      ..strokeWidth = _s(2)
      ..style = PaintingStyle.stroke;

    for (final e in edges) {
      final src = _byId(e['source'] as String);
      final tgt = _byId(e['target'] as String);
      if (src == null || tgt == null) continue;
      _drawBezier(canvas, _t(_outH(src)), _t(_inH(tgt)), paint);

      // Item count badge after execution
      if (execCounts != null) {
        final count = execCounts![e['id']];
        if (count != null) {
          final mid = (_t(_outH(src)) + _t(_inH(tgt))) / 2;
          _drawBadge(canvas, mid, '$count items');
        }
      }
    }
  }

  void _drawBezier(Canvas canvas, Offset from, Offset to, Paint p) {
    final dx = (to.dx - from.dx).abs() * 0.5;
    final backward = to.dx < from.dx;
    final path = Path()..moveTo(from.dx, from.dy);
    if (backward) {
      final spread = max(dx, _s(80));
      path.cubicTo(from.dx + spread, from.dy, to.dx - spread, to.dy, to.dx, to.dy);
    } else {
      path.cubicTo(from.dx + dx, from.dy, to.dx - dx, to.dy, to.dx, to.dy);
    }
    canvas.drawPath(path, p);
  }

  void _drawBadge(Canvas canvas, Offset pos, String text) {
    final tp = TextPainter(
      text: TextSpan(
          text: text,
          style: TextStyle(color: Colors.white, fontSize: _s(9))),
      textDirection: TextDirection.ltr,
    )..layout();
    final r = RRect.fromRectAndRadius(
        Rect.fromCenter(
            center: pos,
            width: tp.width + _s(12),
            height: _s(18)),
        Radius.circular(_s(9)));
    canvas.drawRRect(r, Paint()..color = _accent);
    tp.paint(canvas, Offset(pos.dx - tp.width / 2, pos.dy - tp.height / 2));
  }

  void _drawNode(Canvas canvas, Map<String, dynamic> n) {
    final id = n['id'] as String;
    final pos = _nodePos(n);
    final d = _def(n['type'] ?? 'http_request');
    final sel = selected.contains(id);
    final disabled = n['disabled'] == true;
    final screenPos = _t(pos);
    final w = _s(_nodeW);
    final h = _s(_nodeH);

    final rect = RRect.fromRectAndRadius(
        Rect.fromLTWH(screenPos.dx, screenPos.dy, w, h),
        Radius.circular(_s(10)));

    // Shadow
    canvas.drawRRect(
        rect.shift(Offset(0, _s(4))),
        Paint()
          ..color = sel
              ? d.color.withOpacity(0.08)
              : Colors.black.withOpacity(0.25)
          ..maskFilter = MaskFilter.blur(BlurStyle.normal, _s(8)));

    // Body
    canvas.drawRRect(
        rect,
        Paint()
          ..color = disabled ? const Color(0xFF111114) : _surface);

    // Border
    canvas.drawRRect(
        rect,
        Paint()
          ..color = sel
              ? d.color
              : disabled
                  ? Colors.white.withOpacity(0.04)
                  : _border
          ..style = PaintingStyle.stroke
          ..strokeWidth = sel ? _s(1.5) : _s(1));

    // Left color bar
    final barRect = RRect.fromRectAndCorners(
        Rect.fromLTWH(screenPos.dx, screenPos.dy, _s(4), h),
        topLeft: Radius.circular(_s(10)),
        bottomLeft: Radius.circular(_s(10)));
    canvas.drawRRect(
        barRect,
        Paint()..color = disabled ? const Color(0xFF333) : d.color);

    // Icon bg
    final iconX = screenPos.dx + _s(16);
    final iconY = screenPos.dy + h / 2 - _s(16);
    final iconRect = RRect.fromRectAndRadius(
        Rect.fromLTWH(iconX, iconY, _s(32), _s(32)),
        Radius.circular(_s(7)));
    canvas.drawRRect(iconRect, Paint()..color = d.color.withOpacity(0.12));

    // Label text
    final labelTp = TextPainter(
      text: TextSpan(
        text: n['label'] ?? d.label,
        style: TextStyle(
          color: disabled ? _subtleText : Colors.white,
          fontSize: _s(13),
          fontWeight: FontWeight.w500,
        ),
      ),
      textDirection: TextDirection.ltr,
      maxLines: 1,
      ellipsis: '…',
    )..layout(maxWidth: w - _s(70));
    labelTp.paint(canvas,
        Offset(iconX + _s(42), screenPos.dy + h / 2 - _s(14)));

    // Type subtitle
    final typeTp = TextPainter(
      text: TextSpan(
          text: n['type'] ?? '',
          style: TextStyle(color: _subtleText, fontSize: _s(10))),
      textDirection: TextDirection.ltr,
    )..layout();
    typeTp.paint(
        canvas, Offset(iconX + _s(42), screenPos.dy + h / 2 + _s(4)));

    // Input handles (left)
    for (var ii = 0; ii < d.inputs; ii++) {
      final inP = _t(_inH(n, ii));
      canvas.drawCircle(inP, _s(_handleR), Paint()..color = _surface);
      canvas.drawCircle(
          inP,
          _s(_handleR),
          Paint()
            ..color = _dimText
            ..style = PaintingStyle.stroke
            ..strokeWidth = _s(1.5));
    }

    // Output handles (right) with labels
    for (var oi = 0; oi < d.outputs; oi++) {
      final outP = _t(_outH(n, oi));
      canvas.drawCircle(outP, _s(_handleR), Paint()..color = _accent);
      canvas.drawCircle(
          outP,
          _s(_handleR),
          Paint()
            ..color = Colors.white.withOpacity(0.3)
            ..style = PaintingStyle.stroke
            ..strokeWidth = _s(1.5));
      // Label for multi-output
      if (d.outputs > 1 && d.outputLabels != null && oi < d.outputLabels!.length) {
        final labelTp2 = TextPainter(
          text: TextSpan(
              text: d.outputLabels![oi],
              style: TextStyle(
                  color: _subtleText, fontSize: _s(8))),
          textDirection: TextDirection.ltr,
        )..layout();
        labelTp2.paint(canvas,
            Offset(outP.dx + _s(10), outP.dy - labelTp2.height / 2));
      }
    }

    // Execution status badge
    if (execStatus != null) {
      final st = execStatus![id];
      if (st != null) {
        final badgePos = Offset(screenPos.dx + w - _s(8), screenPos.dy - _s(4));
        Color bc;
        switch (st) {
          case 'completed':
            bc = _green;
          case 'failed':
            bc = _red;
          case 'skipped':
            bc = const Color(0xFF64748B);
          default:
            bc = _accent;
        }
        canvas.drawCircle(badgePos, _s(8), Paint()..color = _surface);
        canvas.drawCircle(badgePos, _s(6), Paint()..color = bc);
      }
    }

    // Disabled overlay strikethrough
    if (disabled) {
      canvas.drawLine(
          Offset(screenPos.dx, screenPos.dy + h / 2),
          Offset(screenPos.dx + w, screenPos.dy + h / 2),
          Paint()
            ..color = _red.withOpacity(0.3)
            ..strokeWidth = _s(1.5));
    }
  }

  @override
  bool shouldRepaint(covariant _CanvasPainter old) => true;
}

// ═════════════════════════════════════════════════════════════════════════════
// Executions dialog
// ═════════════════════════════════════════════════════════════════════════════

class _ExecDialog extends StatelessWidget {
  final String wfId;
  final WidgetRef ref;
  const _ExecDialog({required this.wfId, required this.ref});

  @override
  Widget build(BuildContext context) {
    return Dialog(
      backgroundColor: _surface,
      shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(12),
          side: const BorderSide(color: _border)),
      child: SizedBox(
        width: 640,
        height: 480,
        child: Column(children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(24, 20, 16, 16),
            child: Row(children: [
              const Icon(LucideIcons.history, size: 18, color: _dimText),
              const SizedBox(width: 10),
              const Text('Execution History',
                  style: TextStyle(
                      color: Colors.white,
                      fontSize: 16,
                      fontWeight: FontWeight.w600)),
              const Spacer(),
              IconButton(
                  onPressed: () => Navigator.pop(context),
                  icon: const Icon(LucideIcons.x, size: 16, color: _dimText)),
            ]),
          ),
          Container(height: 1, color: _border),
          Expanded(
            child: FutureBuilder<dynamic>(
              future: ref
                  .read(apiClientProvider)
                  .get('/workflows/$wfId/executions')
                  .then((r) => r.data),
              builder: (ctx, snap) {
                if (snap.connectionState == ConnectionState.waiting) {
                  return const Center(
                      child: CircularProgressIndicator(color: _accent));
                }
                if (snap.hasError) {
                  return Center(
                      child: Text('${snap.error}',
                          style: const TextStyle(color: _dimText)));
                }
                final execs = List<Map<String, dynamic>>.from(
                    (snap.data as Map)['executions'] ?? []);
                if (execs.isEmpty) {
                  return const Center(
                      child: Text('No executions yet',
                          style: TextStyle(color: _dimText)));
                }
                return ListView.separated(
                  padding: const EdgeInsets.all(16),
                  itemCount: execs.length,
                  separatorBuilder: (_, __) => const SizedBox(height: 8),
                  itemBuilder: (ctx, i) {
                    final e = execs[i];
                    final st = e['status'] ?? 'pending';
                    final dur = e['durationMs'] ?? 0;
                    final logs =
                        List<Map<String, dynamic>>.from(e['logs'] ?? []);
                    final idStr = (e['\$id'] ?? '').toString();

                    return Container(
                      decoration: BoxDecoration(
                          color: _bg,
                          borderRadius: BorderRadius.circular(8),
                          border: Border.all(color: _border)),
                      child: ExpansionTile(
                        leading: _icon(st),
                        title: Text(
                            idStr.length > 8 ? idStr.substring(0, 8) : idStr,
                            style: const TextStyle(
                                color: Colors.white, fontSize: 13)),
                        subtitle: Text('$st  •  ${dur}ms',
                            style: const TextStyle(
                                color: _subtleText, fontSize: 11)),
                        children: [
                          if (e['error'] != null &&
                              (e['error'] as String).isNotEmpty)
                            Padding(
                              padding: const EdgeInsets.symmetric(
                                  horizontal: 16, vertical: 8),
                              child: Text('Error: ${e['error']}',
                                  style: const TextStyle(
                                      color: Colors.redAccent, fontSize: 12)),
                            ),
                          ...logs.map((l) => ListTile(
                                dense: true,
                                leading: _icon(l['status'] ?? ''),
                                title: Text(
                                    '${l['label']} (${l['nodeType']})',
                                    style: const TextStyle(
                                        color: Colors.white, fontSize: 12)),
                                subtitle: Text('${l['durationMs']}ms',
                                    style: const TextStyle(
                                        color: _subtleText, fontSize: 11)),
                              )),
                        ],
                      ),
                    );
                  },
                );
              },
            ),
          ),
        ]),
      ),
    );
  }

  Widget _icon(String s) {
    switch (s) {
      case 'completed':
        return const Icon(LucideIcons.checkCircle2, size: 18, color: _green);
      case 'running':
        return const Icon(LucideIcons.loader2, size: 18, color: _accent);
      case 'failed':
        return const Icon(LucideIcons.xCircle, size: 18, color: _red);
      case 'skipped':
        return const Icon(LucideIcons.skipForward, size: 18, color: _dimText);
      default:
        return const Icon(LucideIcons.clock, size: 18, color: _dimText);
    }
  }
}

// ═════════════════════════════════════════════════════════════════════════════
// Minimap painter
// ═════════════════════════════════════════════════════════════════════════════

class _MinimapPainter extends CustomPainter {
  final List<Map<String, dynamic>> nodes;
  final Offset pan;
  final double zoom;
  final Size viewSize;

  _MinimapPainter(
      {required this.nodes,
      required this.pan,
      required this.zoom,
      required this.viewSize});

  @override
  void paint(Canvas canvas, Size size) {
    if (nodes.isEmpty) return;

    double minX = double.infinity, minY = double.infinity;
    double maxX = double.negativeInfinity, maxY = double.negativeInfinity;
    for (final n in nodes) {
      final p = n['position'] as Map<String, dynamic>?;
      final x = (p?['x'] as num?)?.toDouble() ?? 0;
      final y = (p?['y'] as num?)?.toDouble() ?? 0;
      minX = min(minX, x);
      minY = min(minY, y);
      maxX = max(maxX, x + _nodeW);
      maxY = max(maxY, y + _nodeH);
    }

    final contentW = maxX - minX + 200;
    final contentH = maxY - minY + 200;
    final scale = min(size.width / contentW, size.height / contentH);

    // Draw nodes as small rectangles
    for (final n in nodes) {
      final p = n['position'] as Map<String, dynamic>?;
      final x = ((p?['x'] as num?)?.toDouble() ?? 0) - minX + 100;
      final y = ((p?['y'] as num?)?.toDouble() ?? 0) - minY + 100;
      final d = _def(n['type'] ?? '');
      canvas.drawRRect(
        RRect.fromRectAndRadius(
            Rect.fromLTWH(x * scale, y * scale, _nodeW * scale, _nodeH * scale),
            const Radius.circular(2)),
        Paint()..color = d.color.withOpacity(0.5),
      );
    }

    // Draw viewport rectangle
    final vpLeft = (-pan.dx / zoom - minX + 100) * scale;
    final vpTop = (-pan.dy / zoom - minY + 100) * scale;
    final vpW = (viewSize.width / zoom) * scale;
    final vpH = (viewSize.height / zoom) * scale;
    canvas.drawRect(
      Rect.fromLTWH(vpLeft, vpTop, vpW, vpH),
      Paint()
        ..color = Colors.white.withOpacity(0.15)
        ..style = PaintingStyle.stroke
        ..strokeWidth = 1,
    );
  }

  @override
  bool shouldRepaint(covariant _MinimapPainter old) => true;
}

// ═════════════════════════════════════════════════════════════════════════════
// Intent types for keyboard shortcuts
// ═════════════════════════════════════════════════════════════════════════════

class _UndoIntent extends Intent {
  const _UndoIntent();
}

class _RedoIntent extends Intent {
  const _RedoIntent();
}

class _CopyIntent extends Intent {
  const _CopyIntent();
}

class _PasteIntent extends Intent {
  const _PasteIntent();
}

class _DuplicateIntent extends Intent {
  const _DuplicateIntent();
}

class _SelectAllIntent extends Intent {
  const _SelectAllIntent();
}

class _SaveIntent extends Intent {
  const _SaveIntent();
}

class _DeleteIntent extends Intent {
  const _DeleteIntent();
}

class _DisableIntent extends Intent {
  const _DisableIntent();
}

class _AddNodeIntent extends Intent {
  const _AddNodeIntent();
}

class _EscapeIntent extends Intent {
  const _EscapeIntent();
}
