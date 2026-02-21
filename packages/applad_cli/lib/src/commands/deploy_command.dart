import 'package:args/command_runner.dart';
import '../utils/output.dart';

final class DeployCommand extends Command<void> {
  DeployCommand() {
    addSubcommand(_DeploySubcommand('run', 'Run a deployment.'));
    addSubcommand(_DeploySubcommand('status', 'Show deployment status.'));
    addSubcommand(
        _DeploySubcommand('rollback', 'Roll back to previous deployment.'));
  }

  @override
  String get name => 'deploy';
  @override
  String get description => 'Manage deployments. (Phase 4)';
}

final class _DeploySubcommand extends Command<void> {
  _DeploySubcommand(this._name, this._description);
  final String _name;
  final String _description;
  @override
  String get name => _name;
  @override
  String get description => '$_description (Phase 4)';
  @override
  Future<void> run() async =>
      Output.info('applad deploy $name — coming in Phase 4');
}
