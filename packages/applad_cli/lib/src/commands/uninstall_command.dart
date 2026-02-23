import 'dart:io';
import 'package:args/command_runner.dart';
import 'package:path/path.dart' as p;
import '../utils/output.dart';

/// Command to uninstall Applad by cleaning up configuration and session data.
final class UninstallCommand extends Command<void> {
  @override
  String get name => 'uninstall';

  @override
  String get description =>
      'Permanently removes Applad\'s local and global configuration data.';

  @override
  Future<void> run() async {
    Output.header('Uninstalling Applad CLI');
    Output.warning(
        'This will permanently delete your session, trust state, and global configuration.');
    Output.blank();

    final confirm = Output.confirm(
      'Are you absolutely sure you want to proceed?',
      defaultValue: false,
    );

    if (!confirm) {
      Output.info('Uninstall aborted.');
      return;
    }

    // 1. Clean up global config (~/.applad)
    final home = Platform.environment['HOME'];
    if (home != null) {
      final globalConfigDir = Directory(p.join(home, '.applad'));
      if (globalConfigDir.existsSync()) {
        try {
          globalConfigDir.deleteSync(recursive: true);
          Output.success('Global configuration removed (trust, session, etc.)');
        } catch (e) {
          Output.error('Failed to remove global configuration: $e');
        }
      } else {
        Output.info('No global configuration found (~/.applad).');
      }
    }

    // 2. Clean up local config (./.applad)
    final localConfigDir = Directory(p.join(Directory.current.path, '.applad'));
    if (localConfigDir.existsSync()) {
      final deleteLocal = Output.confirm(
        'Found local workspace configuration (./.applad). Delete it as well?',
        defaultValue: false,
      );

      if (deleteLocal) {
        try {
          localConfigDir.deleteSync(recursive: true);
          Output.success('Local workspace configuration removed.');
        } catch (e) {
          Output.error('Failed to remove local configuration: $e');
        }
      }
    }

    Output.blank();
    Output.success('Cleanup complete!');
    Output.blank();
    Output.header('Next steps to complete uninstallation:');
    Output.info(
        'The Applad binary itself is still on your system. To remove it:');
    Output.blank();
    Output.step(
        1, 'If installed via pub: run `dart pub global deactivate applad`');
    Output.step(
        2, 'If installed manually: delete the `applad` binary from your PATH');
    Output.blank();
  }
}
