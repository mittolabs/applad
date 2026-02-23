import 'package:flutter/material.dart';

class SettingsScreen extends StatelessWidget {
  const SettingsScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Settings')),
      body: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text('Settings', style: Theme.of(context).textTheme.headlineSmall),
            const SizedBox(height: 24),
            ListTile(
              leading: const Icon(Icons.info_outline),
              title: const Text('Version'),
              subtitle: const Text('0.1.0'),
            ),
            ListTile(
              leading: const Icon(Icons.code_outlined),
              title: const Text('Source Code'),
              subtitle: const Text('github.com/mittolabs/applad'),
              onTap: () {},
            ),
          ],
        ),
      ),
    );
  }
}
