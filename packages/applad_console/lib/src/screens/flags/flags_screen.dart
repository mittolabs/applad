import 'package:flutter/material.dart';
import '../../widgets/empty_state.dart';

class FlagsScreen extends StatelessWidget {
  const FlagsScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Feature Flags')),
      body: const ComingSoonState(feature: 'Feature Flags', phase: 'Phase 4'),
    );
  }
}
