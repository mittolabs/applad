import 'package:flutter/material.dart';
import '../../widgets/empty_state.dart';

class MessagingScreen extends StatelessWidget {
  const MessagingScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Messaging')),
      body: const ComingSoonState(feature: 'Messaging', phase: 'Phase 4'),
    );
  }
}
