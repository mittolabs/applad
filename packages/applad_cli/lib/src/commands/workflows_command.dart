import 'dart:io';
import 'package:args/command_runner.dart';
import 'package:path/path.dart' as p;
import '../utils/output.dart';
import '../utils/config_finder.dart';

/// `applad workflows` — group for workflow management.
final class WorkflowsCommand extends Command<void> {
  WorkflowsCommand() {
    addSubcommand(WorkflowsCreateCommand());
  }

  @override
  String get name => 'workflows';
  @override
  String get description => 'Automation workflows management.';
}

/// `applad workflows create` — guided workflow creation.
final class WorkflowsCreateCommand extends Command<void> {
  @override
  String get name => 'create';
  @override
  String get description => 'Guided creation of a new automation workflow.';

  @override
  Future<void> run() async {
    Output.header('Create Workflow');

    final projectDir = ConfigFinder.discoverProjectRoot();
    if (projectDir == null) {
      Output.error('No Applad project found.');
      return;
    }

    final projectName = p.basename(projectDir.path);
    Output.info('Selected project: $projectName');

    final workflowsDir = Directory('${projectDir.path}/workflows');
    if (!workflowsDir.existsSync()) {
      Output.info('Workflows namespace is not enabled. Creating directory...');
      workflowsDir.createSync(recursive: true);
    }

    final name =
        Output.prompt('Workflow name', defaultValue: 'user-signup-flow');
    final trigger = Output.prompt('Trigger type (event/schedule/manual)',
        defaultValue: 'event');

    final yamlFile = File('${workflowsDir.path}/$name.yaml');
    if (yamlFile.existsSync()) {
      Output.error('Workflow "$name" already exists.');
      return;
    }

    final content = '''
# ============================================================
# WORKFLOW: $name
# Generated via applad workflows create
# ============================================================

name: "$name"
enabled: true

trigger:
  type: "$trigger"
  ${trigger == 'event' ? 'event: "user.created"' : trigger == 'schedule' ? 'cron: "0 0 * * *"' : ''}

# Sequence of steps to execute
steps:
  - id: "send-welcome"
    type: "messaging.send"
    template: "welcome-email"
    to: "\${event.user.email}"
  - id: "log-event"
    type: "functions.call"
    function: "log-analytics"
''';

    yamlFile.writeAsStringSync(content);

    Output.blank();
    Output.success('Created workflow config: ${yamlFile.path}');
    Output.blank();
    Output.nextSteps([
      'Edit ${yamlFile.path} to define the workflow steps.',
      'Run `applad up` to deploy.'
    ]);
  }
}
