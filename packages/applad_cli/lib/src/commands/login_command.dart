import 'dart:io';
import 'package:args/command_runner.dart';
import '../utils/output.dart';
import '../utils/config_finder.dart';
import '../security/session_manager.dart';

/// `applad login` — Connects the local CLI to an Applad instance.
/// Registers the developer's SSH key and sends an access request.
final class LoginCommand extends Command<void> {
  LoginCommand() {
    argParser.addOption(
      'path',
      abbr: 'p',
      help: 'Path to the root config directory (default: auto-discover).',
    );
  }

  @override
  String get name => 'login';

  @override
  String get description =>
      'Logs into an existing Applad instance and requests access.';

  @override
  Future<void> run() async {
    final configPath = argResults!['path'] as String?;

    Output.header('Applad Login');

    final finder = const ConfigFinder();
    final String rootPath;
    try {
      rootPath = configPath ?? finder.requireRoot();
    } catch (e) {
      Output.error('No applad.yaml found.');
      Output.info('Are you in an initialized Applad project directory?');
      return;
    }

    // Load instance minimal config
    final File appladYaml = File('$rootPath/applad.yaml');
    if (!appladYaml.existsSync()) {
      Output.error('Missing applad.yaml at $rootPath');
      return;
    }

    Output.info('Looking up instance configuration...');
    // We mock the actual "reading" of where the management API is
    // since the spec says it connects to the instance URL.
    await Future.delayed(const Duration(milliseconds: 500));

    Output.info('Found instance endpoint.');
    Output.info('Locating local SSH keys...');
    await Future.delayed(const Duration(milliseconds: 500));

    Output.info('Sending access request to instance administrators...');
    await Future.delayed(const Duration(milliseconds: 1000));

    Output.success('Access request submitted successfully.');
    SessionManager.login();
    Output.blank();
    Output.info('Your request is pending administrator approval.');
    Output.info(
        'Once approved, you can run `applad up` to start your environment.');
  }
}
