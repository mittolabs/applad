import 'dart:io';
import 'package:args/command_runner.dart';
import '../utils/output.dart';
import '../utils/config_finder.dart';

/// `applad messaging` — group for messaging management.
final class MessagingCommand extends Command<void> {
  MessagingCommand() {
    addSubcommand(MessagingTemplatesCommand());
  }

  @override
  String get name => 'messaging';
  @override
  String get description => 'Unified messaging management.';
}

/// `applad messaging templates` — group for template management.
final class MessagingTemplatesCommand extends Command<void> {
  MessagingTemplatesCommand() {
    addSubcommand(MessagingTemplatesCreateCommand());
  }

  @override
  String get name => 'templates';
  @override
  String get description => 'Message template management.';
}

/// `applad messaging templates create` — guided template creation.
final class MessagingTemplatesCreateCommand extends Command<void> {
  @override
  String get name => 'create';
  @override
  String get description => 'Guided creation of a new message template.';

  @override
  Future<void> run() async {
    Output.header('Create Message Template');

    final projectDir = ConfigFinder.findProjectRoot();
    if (projectDir == null) {
      Output.error('No Applad project found (missing project.yaml).');
      return;
    }

    final messagingDir = Directory('${projectDir.path}/messaging/templates');
    if (!messagingDir.existsSync()) {
      Output.info('Messaging namespace is not enabled. Creating directory...');
      messagingDir.createSync(recursive: true);
    }

    final name = Output.prompt('Template name', defaultValue: 'welcome-email');
    final type = Output.prompt('Type (email/sms/push)', defaultValue: 'email');

    final yamlFile = File('${messagingDir.path}/$name.yaml');
    if (yamlFile.existsSync()) {
      Output.error('Template "$name" already exists.');
      return;
    }

    final content = '''
# ============================================================
# MESSAGE TEMPLATE: $name
# Generated via applad messaging templates create
# ============================================================

name: "$name"
type: "$type"

# Template content (supports Liquid/Handlebars based on engine)
body: |
  Hi {{user.name}},
  Welcome to our app!
''';

    yamlFile.writeAsStringSync(content);

    Output.blank();
    Output.success('Created template config: ${yamlFile.path}');
    Output.blank();
    Output.nextSteps([
      'Edit ${yamlFile.path} to customize the message body.',
      'Run `applad up` to deploy.'
    ]);
  }
}
