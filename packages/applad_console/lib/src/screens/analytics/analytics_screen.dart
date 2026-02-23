import 'package:flutter/material.dart';
import '../../widgets/empty_state.dart';

class AnalyticsScreen extends StatelessWidget {
  const AnalyticsScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Analytics')),
      body: const ComingSoonState(feature: 'Analytics', phase: 'Phase 4'),
    );
  }
}
