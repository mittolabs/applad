import 'package:flutter/material.dart';
import '../../widgets/empty_state.dart';

class AuthScreen extends StatelessWidget {
  const AuthScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Authentication')),
      body: const ComingSoonState(feature: 'Authentication', phase: 'Phase 2'),
    );
  }
}
