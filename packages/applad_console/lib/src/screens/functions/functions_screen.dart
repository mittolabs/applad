import 'package:flutter/material.dart';
import '../../widgets/empty_state.dart';

class FunctionsScreen extends StatelessWidget {
  const FunctionsScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Functions')),
      body: const ComingSoonState(feature: 'Functions', phase: 'Phase 3'),
    );
  }
}
