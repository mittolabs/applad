import 'package:args/command_runner.dart';
import '../utils/version.dart';
import '../utils/output.dart';

/// `applad version` — Prints the currently installed Applad version.
final class VersionCommand extends Command<void> {
  @override
  String get name => 'version';

  @override
  String get description => 'Prints the currently installed Applad version.';

  @override
  void run() {
    Output.info('applad v$kApplAdVersion');
  }
}
