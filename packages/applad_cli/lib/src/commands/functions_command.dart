import 'package:args/command_runner.dart';
import '../utils/output.dart';

final class FunctionsCommand extends Command<void> {
  FunctionsCommand() {
    addSubcommand(_FnSubcommand('deploy', 'Deploy a function.'));
    addSubcommand(_FnSubcommand('invoke', 'Invoke a function locally.'));
    addSubcommand(_FnSubcommand('logs', 'Stream function logs.'));
  }

  @override
  String get name => 'functions';
  @override
  String get description => 'Manage serverless functions. (Phase 3)';
}

final class _FnSubcommand extends Command<void> {
  _FnSubcommand(this._name, this._description);
  final String _name;
  final String _description;

  @override
  String get name => _name;
  @override
  String get description => '$_description (Phase 3)';

  @override
  Future<void> run() async =>
      Output.info('applad functions $name — coming in Phase 3');
}
