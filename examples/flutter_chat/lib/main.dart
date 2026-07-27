import 'package:flutter/material.dart';

import 'applad_service.dart';
import 'config.dart';
import 'screens/channels_screen.dart';
import 'screens/login_screen.dart';

void main() {
  runApp(const ChatApp());
}

class ChatApp extends StatelessWidget {
  const ChatApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Applad Chat',
      debugShowCheckedModeBanner: false,
      theme: ThemeData(
        useMaterial3: true,
        colorSchemeSeed: const Color(0xFF6C47FF),
        brightness: Brightness.dark,
        scaffoldBackgroundColor: const Color(0xFF0B0B0F),
      ),
      home: const _Gate(),
    );
  }
}

/// Decides the first screen: a misconfiguration notice, the channel list for a
/// recovered session, or the login screen.
class _Gate extends StatefulWidget {
  const _Gate();

  @override
  State<_Gate> createState() => _GateState();
}

class _GateState extends State<_Gate> {
  late final Future<bool> _restore;

  @override
  void initState() {
    super.initState();
    _restore = Config.isConfigured
        ? AppladService.instance.restore()
        : Future.value(false);
  }

  @override
  Widget build(BuildContext context) {
    if (!Config.isConfigured) {
      return const _Misconfigured();
    }
    return FutureBuilder<bool>(
      future: _restore,
      builder: (context, snap) {
        if (snap.connectionState != ConnectionState.done) {
          return const Scaffold(body: Center(child: CircularProgressIndicator()));
        }
        return snap.data == true ? const ChannelsScreen() : const LoginScreen();
      },
    );
  }
}

class _Misconfigured extends StatelessWidget {
  const _Misconfigured();

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: Center(
        child: Padding(
          padding: const EdgeInsets.all(32),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              const Icon(Icons.settings_outlined, size: 40),
              const SizedBox(height: 16),
              Text('No project configured',
                  style: Theme.of(context).textTheme.titleLarge),
              const SizedBox(height: 8),
              const Text(
                'Pass --dart-define=APPLAD_PROJECT=<id> (and APPLAD_ENDPOINT). '
                'Create the project in the console, then run tool/bootstrap.dart '
                'once to create the chat schema.',
                textAlign: TextAlign.center,
              ),
            ],
          ),
        ),
      ),
    );
  }
}
