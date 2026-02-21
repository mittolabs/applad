import 'package:args/command_runner.dart';
import '../utils/output.dart';

final class FlagsCommand extends Command<void> {
  FlagsCommand() {
    addSubcommand(_FlagSubcommand('list', 'List all feature flags.'));
    addSubcommand(_FlagSubcommand('enable', 'Enable a feature flag.'));
    addSubcommand(_FlagSubcommand('disable', 'Disable a feature flag.'));
  }

  @override
  String get name => 'flags';
  @override
  String get description => 'Manage feature flags. (Phase 4)';
}

final class _FlagSubcommand extends Command<void> {
  _FlagSubcommand(this._name, this._description);
  final String _name;
  final String _description;
  @override
  String get name => _name;
  @override
  String get description => '$_description (Phase 4)';
  @override
  Future<void> run() async =>
      Output.info('applad flags $name — coming in Phase 4');
}
