import 'package:args/command_runner.dart';
import '../utils/output.dart';

final class HostingCommand extends Command<void> {
  HostingCommand() {
    addSubcommand(_HostSubcommand('deploy', 'Deploy static hosting.'));
    addSubcommand(_HostSubcommand('status', 'Show hosting status.'));
    addSubcommand(_HostSubcommand('rollback', 'Roll back hosting deployment.'));
  }

  @override
  String get name => 'hosting';
  @override
  String get description => 'Manage static hosting. (Phase 4)';
}

final class _HostSubcommand extends Command<void> {
  _HostSubcommand(this._name, this._description);
  final String _name;
  final String _description;
  @override
  String get name => _name;
  @override
  String get description => '$_description (Phase 4)';
  @override
  Future<void> run() async =>
      Output.info('applad hosting $name — coming in Phase 4');
}
