import 'package:args/command_runner.dart';
import '../utils/output.dart';

final class UsersCommand extends Command<void> {
  UsersCommand() {
    addSubcommand(_UserSubcommand('list', 'List all users.'));
    addSubcommand(
        _UserSubcommand('purge', 'Delete a user and all their data.'));
    addSubcommand(_UserSubcommand('export', 'Export user data.'));
  }

  @override
  String get name => 'users';
  @override
  String get description => 'Manage users. (Phase 2)';
}

final class _UserSubcommand extends Command<void> {
  _UserSubcommand(this._name, this._description);
  final String _name;
  final String _description;
  @override
  String get name => _name;
  @override
  String get description => '$_description (Phase 2)';
  @override
  Future<void> run() async =>
      Output.info('applad users $name — coming in Phase 2');
}
