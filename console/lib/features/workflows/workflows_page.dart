import 'package:flutter/material.dart';

// Phase 1: embeds n8n in a WebView.
class WorkflowsPage extends StatelessWidget {
  const WorkflowsPage({super.key});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Workflows')),
      body: const Center(child: Text('Workflows')),
    );
  }
}
