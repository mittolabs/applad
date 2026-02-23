import 'package:flutter/material.dart';
import '../../widgets/empty_state.dart';

class InstructScreen extends StatelessWidget {
  const InstructScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('AI Assistant')),
      body: const ComingSoonState(feature: 'AI Instruct', phase: 'Phase 5'),
    );
  }
}
