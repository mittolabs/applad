import 'package:args/command_runner.dart';
import '../utils/output.dart';

final class KeysCommand extends Command<void> {
  KeysCommand() {
    addSubcommand(_KeySubcommand('add', 'Register an SSH key.'));
    addSubcommand(_KeySubcommand('list', 'List registered SSH keys.'));
    addSubcommand(_KeySubcommand('revoke', 'Revoke an SSH key.'));
  }

  @override
  String get name => 'keys';
  @override
  String get description => 'Manage SSH keys for audit identity. (Phase 2)';
}

final class _KeySubcommand extends Command<void> {
  _KeySubcommand(this._name, this._description);
  final String _name;
  final String _description;
  @override
  String get name => _name;
  @override
  String get description => '$_description (Phase 2)';
  @override
  Future<void> run() async =>
      Output.info('applad keys $name — coming in Phase 2');
}
