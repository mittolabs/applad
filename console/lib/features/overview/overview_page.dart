import 'dart:ui';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import '../../core/api/client.dart';
import '../../core/providers/project_provider.dart';

// -- Providers that load real project resources --

final _projectResourcesProvider =
    FutureProvider.family<List<_ResourceNode>, String>((ref, projectId) async {
  final api = ref.read(apiClientProvider);
  final nodes = <_ResourceNode>[];
  double x = 100, y = 120;

  // Central project node
  nodes.add(_ResourceNode(
    id: 'project',
    type: 'project',
    name: 'Project',
    subtitle: projectId.length > 12 ? projectId.substring(0, 12) : projectId,
    icon: Icons.cloud,
    color: const Color(0xFF3472A4),
    status: 'active',
    position: Offset(x, 300),
    targetIds: [],
    route: '/settings',
  ));
  x += 320;

  // Databases → tables
  try {
    final dbRes = await api.get('/databases');
    final dbs = List<Map<String, dynamic>>.from(dbRes.data['databases'] ?? []);
    for (final db in dbs) {
      final dbId = db['\$id'] as String;
      final dbNodeId = 'db_$dbId';
      nodes[0].targetIds.add(dbNodeId);
      nodes.add(_ResourceNode(
        id: dbNodeId, type: 'database', name: db['name'] ?? 'Database',
        subtitle: dbId, icon: Icons.table_chart, color: Colors.indigo,
        status: 'active', position: Offset(x, y), targetIds: [], route: '/databases',
      ));

      // Load tables for this database
      try {
        final tRes = await api.get('/databases/$dbId/tables');
        final tables = List<Map<String, dynamic>>.from(tRes.data['tables'] ?? []);
        double tx = x + 300;
        for (final t in tables) {
          final tId = 'tbl_${t['\$id']}';
          nodes.last.targetIds.add(tId);
          nodes.add(_ResourceNode(
            id: tId, type: 'table', name: t['name'] ?? 'Table',
            subtitle: '${t['columns']?.length ?? 0} columns',
            icon: Icons.grid_on, color: Colors.deepPurple,
            status: 'active', position: Offset(tx, y), targetIds: [], route: '/databases',
          ));
          y += 90;
        }
      } catch (_) {}
      y += 30;
    }
  } catch (_) {}

  // Functions
  y = 120;
  x += 300;
  try {
    final fnRes = await api.get('/functions');
    final fns = List<Map<String, dynamic>>.from(fnRes.data['functions'] ?? []);
    if (fns.isNotEmpty) {
      for (final fn in fns) {
        final fnId = 'fn_${fn['\$id']}';
        nodes[0].targetIds.add(fnId);
        final status = fn['status'] ?? 'active';
        nodes.add(_ResourceNode(
          id: fnId, type: 'function', name: fn['name'] ?? 'Function',
          subtitle: '${fn['runtime'] ?? 'unknown'} runtime',
          icon: Icons.functions, color: Colors.pink,
          status: status, position: Offset(x, y), targetIds: [], route: '/functions',
        ));
        y += 100;
      }
    }
  } catch (_) {}

  // Deployments (web apps, containers)
  y = 120;
  x += 300;
  try {
    final depRes = await api.get('/deploy');
    final deps = List<Map<String, dynamic>>.from(depRes.data['deployments'] ?? []);
    for (final dep in deps) {
      final depId = 'dep_${dep['\$id']}';
      nodes[0].targetIds.add(depId);
      final depType = dep['type'] ?? 'web';
      IconData icon;
      Color color;
      switch (depType) {
        case 'container':
          icon = Icons.inventory_2;
          color = Colors.teal;
        case 'function':
          icon = Icons.functions;
          color = Colors.pink;
        case 'mobile':
          icon = Icons.phone_android;
          color = Colors.orange;
        default:
          icon = Icons.web;
          color = Colors.cyan;
      }
      nodes.add(_ResourceNode(
        id: depId, type: depType, name: dep['name'] ?? 'Deployment',
        subtitle: '${dep['status'] ?? 'pending'} — $depType',
        icon: icon, color: color,
        status: dep['status'] ?? 'pending', position: Offset(x, y),
        targetIds: [], route: '/deploy',
      ));
      y += 100;
    }
  } catch (_) {}

  // Workflows
  y = 120;
  x += 300;
  try {
    final wfRes = await api.get('/workflows');
    final wfs = List<Map<String, dynamic>>.from(wfRes.data['workflows'] ?? []);
    for (final wf in wfs) {
      final wfId = 'wf_${wf['\$id']}';
      nodes[0].targetIds.add(wfId);
      final nodeCount = (wf['nodes'] as List?)?.length ?? 0;
      nodes.add(_ResourceNode(
        id: wfId, type: 'workflow', name: wf['name'] ?? 'Workflow',
        subtitle: '${wf['triggerType'] ?? 'manual'} — $nodeCount nodes',
        icon: Icons.account_tree, color: Colors.amber,
        status: wf['status'] ?? 'draft', position: Offset(x, y),
        targetIds: [], route: '/workflows',
      ));
      y += 100;
    }
  } catch (_) {}

  // Storage buckets
  y = 120;
  x += 300;
  try {
    final stRes = await api.get('/storage/buckets');
    final buckets = List<Map<String, dynamic>>.from(stRes.data['buckets'] ?? []);
    for (final b in buckets) {
      final bId = 'bkt_${b['\$id']}';
      nodes[0].targetIds.add(bId);
      nodes.add(_ResourceNode(
        id: bId, type: 'bucket', name: b['name'] ?? 'Bucket',
        subtitle: b['\$id'] ?? '',
        icon: Icons.folder, color: Colors.green,
        status: 'active', position: Offset(x, y),
        targetIds: [], route: '/storage',
      ));
      y += 100;
    }
  } catch (_) {}

  // If nothing besides the project node, show a hint
  if (nodes.length == 1) {
    nodes.add(_ResourceNode(
      id: 'empty', type: 'hint', name: 'Get started',
      subtitle: 'Create a database, function, or deployment',
      icon: Icons.add_circle_outline, color: Colors.white24,
      status: 'hint', position: Offset(x - 900, 300),
      targetIds: [], route: '/databases',
    ));
    nodes[0].targetIds.add('empty');
  }

  return nodes;
});

// -- Node model --

class _ResourceNode {
  final String id;
  final String type;
  final String name;
  final String subtitle;
  final IconData icon;
  final Color color;
  final String status;
  Offset position;
  final List<String> targetIds;
  final String route;

  _ResourceNode({
    required this.id, required this.type, required this.name,
    required this.subtitle, required this.icon, required this.color,
    required this.status, required this.position, required this.targetIds,
    required this.route,
  });
}

// -- Main page --

class OverviewPage extends ConsumerStatefulWidget {
  const OverviewPage({super.key});

  @override
  ConsumerState<OverviewPage> createState() => _OverviewPageState();
}

class _OverviewPageState extends ConsumerState<OverviewPage> {
  // Track dragged positions (overrides from provider)
  final Map<String, Offset> _dragOffsets = {};

  @override
  Widget build(BuildContext context) {
    final projectId = ref.watch(currentProjectProvider);

    if (projectId == null) {
      return Scaffold(
        backgroundColor: const Color(0xFF0B0B0F),
        body: Center(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Icon(Icons.cloud_outlined, size: 64, color: Colors.white.withOpacity(0.2)),
              const SizedBox(height: 16),
              Text('Select a project in Settings', style: TextStyle(color: Colors.white.withOpacity(0.5))),
            ],
          ),
        ),
      );
    }

    final resourcesAsync = ref.watch(_projectResourcesProvider(projectId));

    return Scaffold(
      backgroundColor: const Color(0xFF0B0B0F),
      body: resourcesAsync.when(
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (e, _) => Center(child: Text('Error: $e', style: const TextStyle(color: Colors.red))),
        data: (nodes) => _buildCanvas(context, nodes, projectId),
      ),
    );
  }

  Widget _buildCanvas(BuildContext context, List<_ResourceNode> nodes, String projectId) {
    // Apply drag offsets
    for (final node in nodes) {
      if (_dragOffsets.containsKey(node.id)) {
        node.position = _dragOffsets[node.id]!;
      }
    }

    return Stack(
      children: [
        // Zoomable canvas
        InteractiveViewer(
          boundaryMargin: const EdgeInsets.all(2000),
          minScale: 0.2,
          maxScale: 2.0,
          child: SizedBox(
            width: 2400,
            height: 1200,
            child: Stack(
              children: [
                const Positioned.fill(child: _DottedGrid()),

                // Connection lines
                CustomPaint(
                  size: const Size(2400, 1200),
                  painter: _ConnectionPainter(nodes: nodes),
                ),

                // Resource nodes
                ...nodes.map((node) => Positioned(
                  left: node.position.dx,
                  top: node.position.dy,
                  child: GestureDetector(
                    onPanUpdate: (d) {
                      setState(() {
                        node.position += d.delta;
                        _dragOffsets[node.id] = node.position;
                      });
                    },
                    onDoubleTap: () => context.go(node.route),
                    child: _ResourceCard(node: node),
                  ),
                )),
              ],
            ),
          ),
        ),

        // Top bar
        Positioned(
          top: 0, left: 0, right: 0,
          child: Container(
            height: 52,
            padding: const EdgeInsets.symmetric(horizontal: 20),
            decoration: BoxDecoration(
              color: const Color(0xFF0B0B0F).withOpacity(0.92),
              border: Border(bottom: BorderSide(color: Colors.white.withOpacity(0.05))),
            ),
            child: Row(
              children: [
                const Icon(Icons.cloud, color: Colors.white, size: 22),
                const SizedBox(width: 10),
                const Text('Applad', style: TextStyle(color: Colors.white, fontWeight: FontWeight.bold, fontSize: 15)),
                Text(' / ', style: TextStyle(color: Colors.white.withOpacity(0.15))),
                Text(
                  projectId.length > 12 ? '${projectId.substring(0, 12)}...' : projectId,
                  style: TextStyle(color: Colors.white.withOpacity(0.5), fontSize: 13),
                ),
                const Spacer(),
                _TopBadge(Icons.view_in_ar, '${nodes.length - 1}', 'resources'),
                const SizedBox(width: 12),
                Text('Double-click a node to open',
                  style: TextStyle(color: Colors.white.withOpacity(0.2), fontSize: 11)),
              ],
            ),
          ),
        ),

        // Bottom controls
        Positioned(
          left: 16, bottom: 16,
          child: Column(
            children: [
              _CanvasBtn(Icons.refresh, onTap: () {
                _dragOffsets.clear();
                ref.invalidate(_projectResourcesProvider(projectId));
              }),
              const SizedBox(height: 8),
              _CanvasBtn(Icons.auto_fix_high, onTap: () {
                // Auto-layout: reset positions
                _dragOffsets.clear();
                ref.invalidate(_projectResourcesProvider(projectId));
              }),
            ],
          ),
        ),
      ],
    );
  }
}

// -- Top bar badge --

class _TopBadge extends StatelessWidget {
  final IconData icon;
  final String value;
  final String label;
  const _TopBadge(this.icon, this.value, this.label);

  @override
  Widget build(BuildContext context) {
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        Icon(icon, color: Colors.white24, size: 14),
        const SizedBox(width: 4),
        Text(value, style: const TextStyle(color: Colors.white60, fontSize: 12, fontWeight: FontWeight.bold)),
        const SizedBox(width: 3),
        Text(label, style: TextStyle(color: Colors.white.withOpacity(0.25), fontSize: 11)),
      ],
    );
  }
}

// -- Resource card node --

class _ResourceCard extends StatelessWidget {
  final _ResourceNode node;
  const _ResourceCard({required this.node});

  @override
  Widget build(BuildContext context) {
    return Container(
      width: 220,
      decoration: BoxDecoration(
        color: const Color(0xFF16171B),
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: node.color.withOpacity(0.15)),
        boxShadow: [BoxShadow(color: Colors.black.withOpacity(0.5), blurRadius: 20)],
      ),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Padding(
            padding: const EdgeInsets.all(12),
            child: Row(
              children: [
                Container(
                  padding: const EdgeInsets.all(6),
                  decoration: BoxDecoration(
                    color: node.color.withOpacity(0.12),
                    borderRadius: BorderRadius.circular(8),
                  ),
                  child: Icon(node.icon, color: node.color, size: 18),
                ),
                const SizedBox(width: 10),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(node.name,
                        style: const TextStyle(color: Colors.white, fontWeight: FontWeight.bold, fontSize: 13),
                        overflow: TextOverflow.ellipsis),
                      Text(_typeLabel(node.type),
                        style: TextStyle(color: Colors.white.withOpacity(0.3), fontSize: 10)),
                    ],
                  ),
                ),
              ],
            ),
          ),
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 2),
            child: Row(
              children: [
                _StatusDot(status: node.status),
                const SizedBox(width: 8),
                Text(node.status,
                  style: TextStyle(color: Colors.white.withOpacity(0.35), fontSize: 11)),
              ],
            ),
          ),
          const SizedBox(height: 4),
          Container(
            width: double.infinity,
            padding: const EdgeInsets.all(10),
            decoration: BoxDecoration(
              color: Colors.white.withOpacity(0.02),
              borderRadius: const BorderRadius.vertical(bottom: Radius.circular(12)),
              border: Border(top: BorderSide(color: Colors.white.withOpacity(0.04))),
            ),
            child: Text(node.subtitle,
              style: TextStyle(color: Colors.white.withOpacity(0.4), fontSize: 11),
              overflow: TextOverflow.ellipsis),
          ),
        ],
      ),
    );
  }

  String _typeLabel(String type) {
    switch (type) {
      case 'project': return 'PROJECT';
      case 'database': return 'DATABASE';
      case 'table': return 'TABLE';
      case 'function': return 'FUNCTION';
      case 'web': return 'WEB APP';
      case 'mobile': return 'MOBILE APP';
      case 'container': return 'CONTAINER';
      case 'workflow': return 'WORKFLOW';
      case 'bucket': return 'STORAGE BUCKET';
      case 'hint': return '';
      default: return type.toUpperCase();
    }
  }
}

class _StatusDot extends StatelessWidget {
  final String status;
  const _StatusDot({required this.status});

  @override
  Widget build(BuildContext context) {
    Color color;
    switch (status) {
      case 'active':
      case 'Running':
      case 'Healthy':
        color = Colors.green;
      case 'building':
        color = Colors.orange;
      case 'failed':
        color = Colors.red;
      case 'paused':
      case 'draft':
        color = Colors.grey;
      case 'hint':
        color = Colors.white24;
      default:
        color = Colors.yellow;
    }
    return Container(
      width: 7, height: 7,
      decoration: BoxDecoration(shape: BoxShape.circle, color: color),
    );
  }
}

// -- Bézier connection lines --

class _ConnectionPainter extends CustomPainter {
  final List<_ResourceNode> nodes;
  _ConnectionPainter({required this.nodes});

  @override
  void paint(Canvas canvas, Size size) {
    final linePaint = Paint()
      ..color = Colors.white.withOpacity(0.08)
      ..strokeWidth = 1.5
      ..style = PaintingStyle.stroke;

    final dotPaint = Paint()
      ..color = Colors.white.withOpacity(0.2)
      ..style = PaintingStyle.fill;

    final nodeMap = {for (var n in nodes) n.id: n};

    for (var node in nodes) {
      for (var targetId in node.targetIds) {
        final target = nodeMap[targetId];
        if (target == null) continue;

        final start = node.position + const Offset(220, 45);
        final end = target.position + const Offset(0, 45);

        final path = Path();
        path.moveTo(start.dx, start.dy);

        final dx = (end.dx - start.dx).abs() / 2;
        path.cubicTo(
          start.dx + dx, start.dy,
          end.dx - dx, end.dy,
          end.dx, end.dy,
        );
        canvas.drawPath(path, linePaint);

        canvas.drawCircle(start, 3, dotPaint);
        canvas.drawCircle(end, 3, dotPaint);
      }
    }
  }

  @override
  bool shouldRepaint(covariant CustomPainter oldDelegate) => true;
}

// -- Dotted grid --

class _DottedGrid extends StatelessWidget {
  const _DottedGrid();

  @override
  Widget build(BuildContext context) {
    return CustomPaint(painter: _GridPainter());
  }
}

class _GridPainter extends CustomPainter {
  @override
  void paint(Canvas canvas, Size size) {
    final paint = Paint()
      ..color = Colors.white.withOpacity(0.03)
      ..strokeCap = StrokeCap.round
      ..strokeWidth = 1.5;
    for (double x = 0; x < size.width; x += 30) {
      for (double y = 0; y < size.height; y += 30) {
        canvas.drawPoints(PointMode.points, [Offset(x, y)], paint);
      }
    }
  }

  @override
  bool shouldRepaint(covariant CustomPainter oldDelegate) => false;
}

// -- Canvas button --

class _CanvasBtn extends StatelessWidget {
  final IconData icon;
  final VoidCallback? onTap;
  const _CanvasBtn(this.icon, {this.onTap});

  @override
  Widget build(BuildContext context) {
    return GestureDetector(
      onTap: onTap,
      child: Container(
        padding: const EdgeInsets.all(10),
        decoration: BoxDecoration(
          color: const Color(0xFF16171B),
          borderRadius: BorderRadius.circular(8),
          border: Border.all(color: Colors.white.withOpacity(0.05)),
        ),
        child: Icon(icon, color: Colors.white38, size: 18),
      ),
    );
  }
}
