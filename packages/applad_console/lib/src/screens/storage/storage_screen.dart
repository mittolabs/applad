import 'package:flutter/material.dart';
import '../../widgets/empty_state.dart';

class StorageScreen extends StatelessWidget {
  const StorageScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Storage')),
      body: const ComingSoonState(feature: 'Storage', phase: 'Phase 2'),
    );
  }
}
