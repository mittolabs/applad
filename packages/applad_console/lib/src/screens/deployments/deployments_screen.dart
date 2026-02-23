import 'package:flutter/material.dart';
import '../../widgets/empty_state.dart';

class DeploymentsScreen extends StatelessWidget {
  const DeploymentsScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Deployments')),
      body: const ComingSoonState(feature: 'Deployments', phase: 'Phase 4'),
    );
  }
}
