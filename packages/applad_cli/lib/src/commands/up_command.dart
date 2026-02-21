import 'package:args/command_runner.dart';
import '../utils/output.dart';

final class UpCommand extends Command<void> {
  @override
  String get name => 'up';
  @override
  String get description => 'Start the Applad server. (Phase 3)';

  @override
  Future<void> run() async {
    Output.info('applad up — coming in Phase 3');
    Output.info(
        'In the meantime, run the server with: cd packages/applad_server && dart_frog dev');
  }
}
