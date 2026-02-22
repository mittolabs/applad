import 'dart:io';
import 'package:args/command_runner.dart';
import 'package:path/path.dart' as p;
import '../utils/output.dart';
import '../utils/config_finder.dart';

/// `applad flags` — group for feature flag management.
final class FlagsCommand extends Command<void> {
  FlagsCommand() {
    addSubcommand(FlagsCreateCommand());
  }

  @override
  String get name => 'flags';
  @override
  String get description => 'Feature flag management.';
}

/// `applad flags create` — guided flag creation.
final class FlagsCreateCommand extends Command<void> {
  @override
  String get name => 'create';
  @override
  String get description => 'Guided creation of a new feature flag.';

  @override
  Future<void> run() async {
    Output.header('Create Feature Flag');

    final projectDir = ConfigFinder.discoverProjectRoot();
    if (projectDir == null) {
      Output.error('No Applad project found.');
      return;
    }

    final projectName = p.basename(projectDir.path);
    Output.info('Selected project: $projectName');

    final flagsDir = Directory('${projectDir.path}/flags');
    if (!flagsDir.existsSync()) {
      Output.info('Flags namespace is not enabled. Creating directory...');
      flagsDir.createSync(recursive: true);
    }

    final name = Output.prompt('Flag name', defaultValue: 'new-onboarding');
    final description = Output.prompt('Description',
        defaultValue: 'Use the new onboarding flow.');

    final yamlFile = File('${flagsDir.path}/$name.yaml');
    if (yamlFile.existsSync()) {
      Output.error('Flag "$name" already exists.');
      return;
    }

    final content = '''
# ============================================================
# FEATURE FLAG: $name
# Generated via applad flags create
# ============================================================

name: "$name"
description: "$description"
enabled: false

# Rules for rollout
rules:
  - id: "beta-users"
    condition: "user.role == 'beta'"
    enabled: true
  - id: "percentage-rollout"
    percentage: 10
    enabled: true
''';

    yamlFile.writeAsStringSync(content);

    Output.blank();
    Output.success('Created flag config: ${yamlFile.path}');
    Output.blank();
    Output.nextSteps([
      'Edit ${yamlFile.path} to define rollout rules.',
      'Run `applad up` to deploy.'
    ]);
  }
}
