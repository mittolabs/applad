import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import '../../core/api/client.dart';
import '../../core/providers/auth_provider.dart';
import '../../core/providers/project_provider.dart';

/// Onboarding stepper shown after first login when no projects exist.
/// If projects already exist, redirects straight to /databases.
class OnboardingPage extends ConsumerStatefulWidget {
  const OnboardingPage({super.key});

  @override
  ConsumerState<OnboardingPage> createState() => _OnboardingPageState();
}

class _OnboardingPageState extends ConsumerState<OnboardingPage> {
  int _step = 0;
  final _projectNameCtrl = TextEditingController(text: 'My Project');
  final _projectDescCtrl = TextEditingController();
  final _keyNameCtrl = TextEditingController(text: 'Default Key');

  String? _createdProjectId;
  String? _createdApiKey;
  bool _loading = false;

  @override
  Widget build(BuildContext context) {
    final auth = ref.watch(consoleAuthProvider);
    final user = auth.valueOrNull;

    // Not logged in → redirect to login
    if (user == null && !auth.isLoading) {
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (mounted) context.go('/login');
      });
      return const SizedBox.shrink();
    }

    // Check if projects already exist
    final projectsAsync = ref.watch(projectsProvider);
    return projectsAsync.when(
      loading: () =>
          const Scaffold(body: Center(child: CircularProgressIndicator())),
      error: (e, _) => Scaffold(
        body: Center(child: Text('Error loading projects: $e')),
      ),
      data: (projects) {
        // If projects exist, skip onboarding
        if (projects.isNotEmpty && _createdProjectId == null) {
          // Set the first project as current and go to databases
          WidgetsBinding.instance.addPostFrameCallback((_) {
            if (mounted) {
              final id = projects.first['\$id'] as String;
              ref.read(currentProjectProvider.notifier).state = id;
              ref.read(apiClientProvider).setProject(id);
              context.go('/projects');
            }
          });
          return const SizedBox.shrink();
        }

        return Scaffold(
          body: Center(
            child: SingleChildScrollView(
              child: SizedBox(
                width: 560,
                child: Card(
                  elevation: 4,
                  child: Padding(
                    padding: const EdgeInsets.all(32),
                    child: Column(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        // Header
                        const Icon(Icons.cloud, size: 48),
                        const SizedBox(height: 8),
                        Text('Welcome to Applad',
                            style: Theme.of(context)
                                .textTheme
                                .headlineSmall),
                        if (user != null)
                          Text('Hello, ${user.name.isEmpty ? user.email : user.name}',
                              style: Theme.of(context)
                                  .textTheme
                                  .bodyMedium),
                        const SizedBox(height: 24),

                        // Stepper
                        Stepper(
                          currentStep: _step,
                          controlsBuilder: (_, details) =>
                              const SizedBox.shrink(),
                          steps: [
                            Step(
                              title: const Text('Create your first project'),
                              subtitle: _step > 0
                                  ? Text(_createdProjectId ?? '')
                                  : null,
                              isActive: _step >= 0,
                              state: _step > 0
                                  ? StepState.complete
                                  : StepState.indexed,
                              content: _buildCreateProject(),
                            ),
                            Step(
                              title: const Text('Generate an API key'),
                              subtitle: _step > 1
                                  ? const Text('Key created')
                                  : null,
                              isActive: _step >= 1,
                              state: _step > 1
                                  ? StepState.complete
                                  : StepState.indexed,
                              content: _buildCreateKey(),
                            ),
                            Step(
                              title: const Text('Connect your app'),
                              isActive: _step >= 2,
                              state: _step > 2
                                  ? StepState.complete
                                  : StepState.indexed,
                              content: _buildSdkSnippets(),
                            ),
                          ],
                        ),
                      ],
                    ),
                  ),
                ),
              ),
            ),
          ),
        );
      },
    );
  }

  Widget _buildCreateProject() {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        const Text('Every app starts with a project. Give it a name.'),
        const SizedBox(height: 12),
        TextField(
          controller: _projectNameCtrl,
          decoration: const InputDecoration(labelText: 'Project Name'),
        ),
        const SizedBox(height: 8),
        TextField(
          controller: _projectDescCtrl,
          decoration: const InputDecoration(labelText: 'Description (optional)'),
        ),
        const SizedBox(height: 16),
        FilledButton(
          onPressed: _loading ? null : _createProject,
          child: _loading
              ? const SizedBox(
                  width: 20, height: 20,
                  child: CircularProgressIndicator(strokeWidth: 2))
              : const Text('Create Project'),
        ),
      ],
    );
  }

  Widget _buildCreateKey() {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        const Text(
            'API keys let your app communicate with Applad. '
            'Keep this key secret.'),
        const SizedBox(height: 12),
        TextField(
          controller: _keyNameCtrl,
          decoration: const InputDecoration(labelText: 'Key Name'),
        ),
        const SizedBox(height: 16),
        FilledButton(
          onPressed: _loading ? null : _createKey,
          child: _loading
              ? const SizedBox(
                  width: 20, height: 20,
                  child: CircularProgressIndicator(strokeWidth: 2))
              : const Text('Generate Key'),
        ),
        if (_createdApiKey != null) ...[
          const SizedBox(height: 16),
          Container(
            padding: const EdgeInsets.all(12),
            decoration: BoxDecoration(
              color: Theme.of(context)
                  .colorScheme
                  .surfaceContainerHighest,
              borderRadius: BorderRadius.circular(8),
            ),
            child: Row(
              children: [
                Expanded(
                  child: SelectableText(
                    _createdApiKey!,
                    style: const TextStyle(fontFamily: 'monospace', fontSize: 13),
                  ),
                ),
                IconButton(
                  icon: const Icon(Icons.copy),
                  onPressed: () {
                    Clipboard.setData(
                        ClipboardData(text: _createdApiKey!));
                    ScaffoldMessenger.of(context).showSnackBar(
                      const SnackBar(
                          content: Text('API key copied to clipboard')),
                    );
                  },
                ),
              ],
            ),
          ),
          const SizedBox(height: 8),
          const Text(
            'Save this key now. You won\'t be able to see it again.',
            style: TextStyle(fontWeight: FontWeight.w500, color: Colors.orange),
          ),
          const SizedBox(height: 12),
          FilledButton(
            onPressed: () => setState(() => _step = 2),
            child: const Text('Next'),
          ),
        ],
      ],
    );
  }

  Widget _buildSdkSnippets() {
    final projectId = _createdProjectId ?? 'YOUR_PROJECT_ID';
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        const Text('Install an SDK and start building.'),
        const SizedBox(height: 16),
        _SdkTab(
          tabs: const ['Dart / Flutter', 'JavaScript'],
          contents: [
            '''
# pubspec.yaml
dependencies:
  applad: ^0.1.0

# main.dart
import 'package:applad/applad.dart';

final client = Applad(
  endpoint: 'http://localhost/v1',
  projectId: '$projectId',
);
client.setKey('YOUR_API_KEY');
''',
            '''
npm install @mittolabs/applad

import { Applad } from '@mittolabs/applad';

const client = new Applad({
  endpoint: 'http://localhost',
  projectId: '$projectId',
});
client.setKey('YOUR_API_KEY');
''',
          ],
        ),
        const SizedBox(height: 24),
        SizedBox(
          width: double.infinity,
          child: FilledButton.icon(
            onPressed: _goToDashboard,
            icon: const Icon(Icons.arrow_forward),
            label: const Text('Go to Dashboard'),
          ),
        ),
      ],
    );
  }

  Future<void> _createProject() async {
    setState(() => _loading = true);
    try {
      final api = ref.read(apiClientProvider);
      final res = await api.post('/projects', data: {
        'name': _projectNameCtrl.text,
        'description': _projectDescCtrl.text,
      });
      final data = res.data as Map<String, dynamic>;
      final id = data['\$id'] as String;
      setState(() {
        _createdProjectId = id;
        _step = 1;
      });
      ref.read(currentProjectProvider.notifier).state = id;
      api.setProject(id);
      ref.invalidate(projectsProvider);
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('Error: $e')),
        );
      }
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  Future<void> _createKey() async {
    if (_createdProjectId == null) return;
    setState(() => _loading = true);
    try {
      final api = ref.read(apiClientProvider);
      final res = await api.post(
        '/projects/$_createdProjectId/keys',
        data: {'name': _keyNameCtrl.text, 'scopes': <String>[]},
      );
      final data = res.data as Map<String, dynamic>;
      setState(() {
        _createdApiKey = data['secret'] as String?;
      });
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('Error: $e')),
        );
      }
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  void _goToDashboard() {
    context.go('/projects');
  }
}

// Simple tab widget for SDK snippets
class _SdkTab extends StatefulWidget {
  final List<String> tabs;
  final List<String> contents;
  const _SdkTab({required this.tabs, required this.contents});

  @override
  State<_SdkTab> createState() => _SdkTabState();
}

class _SdkTabState extends State<_SdkTab> {
  int _selected = 0;

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          children: List.generate(widget.tabs.length, (i) {
            return Padding(
              padding: const EdgeInsets.only(right: 8),
              child: ChoiceChip(
                label: Text(widget.tabs[i]),
                selected: _selected == i,
                onSelected: (_) => setState(() => _selected = i),
              ),
            );
          }),
        ),
        const SizedBox(height: 8),
        Container(
          width: double.infinity,
          padding: const EdgeInsets.all(16),
          decoration: BoxDecoration(
            color:
                Theme.of(context).colorScheme.surfaceContainerHighest,
            borderRadius: BorderRadius.circular(8),
          ),
          child: Stack(
            children: [
              SelectableText(
                widget.contents[_selected].trim(),
                style: const TextStyle(
                    fontFamily: 'monospace', fontSize: 13),
              ),
              Positioned(
                top: 0,
                right: 0,
                child: IconButton(
                  icon: const Icon(Icons.copy, size: 18),
                  onPressed: () {
                    Clipboard.setData(ClipboardData(
                        text: widget.contents[_selected].trim()));
                    ScaffoldMessenger.of(context).showSnackBar(
                      const SnackBar(content: Text('Copied!')),
                    );
                  },
                ),
              ),
            ],
          ),
        ),
      ],
    );
  }
}
