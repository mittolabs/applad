import 'package:args/command_runner.dart';
import '../utils/output.dart';

final class SecretsCommand extends Command<void> {
  SecretsCommand() {
    addSubcommand(_SecretSubcommand('set', 'Set a secret value.'));
    addSubcommand(_SecretSubcommand('rotate', 'Rotate a secret.'));
    addSubcommand(_SecretSubcommand('list', 'List secret names (not values).'));
  }

  @override
  String get name => 'secrets';
  @override
  String get description => 'Manage secrets. (Phase 2)';
}

final class _SecretSubcommand extends Command<void> {
  _SecretSubcommand(this._name, this._description);
  final String _name;
  final String _description;
  @override
  String get name => _name;
  @override
  String get description => '$_description (Phase 2)';
  @override
  Future<void> run() async =>
      Output.info('applad secrets $name — coming in Phase 2');
}
