import 'package:flutter/material.dart';
import '../../widgets/empty_state.dart';

class HostingScreen extends StatelessWidget {
  const HostingScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Hosting')),
      body: const ComingSoonState(feature: 'Hosting', phase: 'Phase 4'),
    );
  }
}
