import 'package:flutter/material.dart';
import '../../widgets/empty_state.dart';

class ObservabilityScreen extends StatelessWidget {
  const ObservabilityScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Observability')),
      body: const ComingSoonState(feature: 'Observability', phase: 'Phase 3'),
    );
  }
}
