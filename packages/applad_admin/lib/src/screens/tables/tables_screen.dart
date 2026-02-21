import 'package:flutter/material.dart';
import '../../widgets/empty_state.dart';

class TablesScreen extends StatelessWidget {
  const TablesScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Tables')),
      body: const ComingSoonState(feature: 'Tables', phase: 'Phase 2'),
    );
  }
}
