import 'package:flutter/material.dart';
import '../../widgets/empty_state.dart';

class WorkflowsScreen extends StatelessWidget {
  const WorkflowsScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Workflows')),
      body: const ComingSoonState(feature: 'Workflows', phase: 'Phase 3'),
    );
  }
}
