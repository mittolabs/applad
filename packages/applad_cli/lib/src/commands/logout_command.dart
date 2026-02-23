import 'package:args/command_runner.dart';
import '../utils/output.dart';
import '../security/session_manager.dart';

/// `applad logout` — Removes local session / credentials for the instance.
final class LogoutCommand extends Command<void> {
  LogoutCommand();

  @override
  String get name => 'logout';

  @override
  String get description =>
      'Logs out from the current Applad instance and clears local credentials.';

  @override
  Future<void> run() async {
    Output.header('Applad Logout');

    Output.info('Clearing local session and credentials...');
    await Future.delayed(const Duration(milliseconds: 600));

    Output.success('Logged out successfully.');
    SessionManager.logout();
    Output.info('Run `applad login` to reconnect to an instance.');
  }
}
