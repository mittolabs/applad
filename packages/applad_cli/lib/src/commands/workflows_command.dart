import 'package:args/command_runner.dart';
import '../utils/output.dart';

final class WorkflowsCommand extends Command<void> {
  WorkflowsCommand() {
    addSubcommand(_WfSubcommand('run', 'Trigger a workflow.'));
    addSubcommand(_WfSubcommand('status', 'Show workflow run status.'));
    addSubcommand(_WfSubcommand('pause', 'Pause a running workflow.'));
  }

  @override
  String get name => 'workflows';
  @override
  String get description => 'Manage workflows. (Phase 3)';
}

final class _WfSubcommand extends Command<void> {
  _WfSubcommand(this._name, this._description);
  final String _name;
  final String _description;
  @override
  String get name => _name;
  @override
  String get description => '$_description (Phase 3)';
  @override
  Future<void> run() async =>
      Output.info('applad workflows $name — coming in Phase 3');
}
