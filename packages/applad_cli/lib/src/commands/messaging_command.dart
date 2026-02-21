import 'package:args/command_runner.dart';
import '../utils/output.dart';

final class MessagingCommand extends Command<void> {
  MessagingCommand() {
    addSubcommand(_ChannelsSubcommand());
    addSubcommand(_TestSubcommand());
    addSubcommand(_LogsSubcommand());
    addSubcommand(_TemplatesSubcommand());
  }

  @override
  String get name => 'messaging';
  @override
  String get description => 'Manage messaging channels, templates, and logs.';
}

final class _ChannelsSubcommand extends Command<void> {
  _ChannelsSubcommand() {
    addSubcommand(_ChannelsListSubcommand());
  }
  @override
  String get name => 'channels';
  @override
  String get description => 'Manage messaging channels.';
}

final class _ChannelsListSubcommand extends Command<void> {
  @override
  String get name => 'list';
  @override
  String get description => 'List configured channels.';
  @override
  Future<void> run() async =>
      Output.info('applad messaging channels list — coming in Phase 4');
}

final class _TestSubcommand extends Command<void> {
  _TestSubcommand() {
    argParser.addOption('to', help: 'Recipient address/number/token.');
    argParser.addOption('template', help: 'Template key to test.');
  }
  @override
  String get name => 'test';
  @override
  String get description =>
      'Test a messaging channel (e.g., email, sms, push, slack).';
  @override
  String get invocation =>
      'applad messaging test <channel> --to <recipient> --template <key>';
  @override
  Future<void> run() async =>
      Output.info('applad messaging test — coming in Phase 4');
}

final class _LogsSubcommand extends Command<void> {
  _LogsSubcommand() {
    argParser.addOption('channel', help: 'Filter by channel.');
    argParser.addOption('template', help: 'Filter by template.');
  }
  @override
  String get name => 'logs';
  @override
  String get description => 'Show delivery logs across all channels.';
  @override
  Future<void> run() async =>
      Output.info('applad messaging logs — coming in Phase 4');
}

final class _TemplatesSubcommand extends Command<void> {
  _TemplatesSubcommand() {
    addSubcommand(_TemplatesListSubcommand());
    addSubcommand(_TemplatesValidateSubcommand());
  }
  @override
  String get name => 'templates';
  @override
  String get description => 'Manage messaging templates.';
}

final class _TemplatesListSubcommand extends Command<void> {
  @override
  String get name => 'list';
  @override
  String get description => 'List template references from config files.';
  @override
  Future<void> run() async =>
      Output.info('applad messaging templates list — coming in Phase 4');
}

final class _TemplatesValidateSubcommand extends Command<void> {
  @override
  String get name => 'validate';
  @override
  String get description =>
      'Validate template references against admin database.';
  @override
  Future<void> run() async =>
      Output.info('applad messaging templates validate — coming in Phase 4');
}
