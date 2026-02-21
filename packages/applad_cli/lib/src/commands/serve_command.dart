import 'package:args/command_runner.dart';
import '../utils/output.dart';

final class ServeCommand extends Command<void> {
  @override
  String get name => 'serve';
  @override
  String get description => 'Alias for `applad up` — start the server.';
  @override
  Future<void> run() async {
    Output.info('applad serve — alias for `applad up`');
    Output.info(
        'In the meantime, run: cd packages/applad_server && dart_frog dev');
  }
}
