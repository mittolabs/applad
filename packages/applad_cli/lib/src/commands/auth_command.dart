import 'package:args/command_runner.dart';
import '../utils/output.dart';

final class AuthCommand extends Command<void> {
  @override
  String get name => 'auth';
  @override
  String get description =>
      'Manage authentication providers and users. (Phase 2)';
  @override
  Future<void> run() async => Output.info('applad auth — coming in Phase 2');
}
