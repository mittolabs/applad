import 'dart:io';
import 'package:args/command_runner.dart';
import 'package:path/path.dart' as p;
import '../utils/output.dart';

final class UpCommand extends Command<void> {
  @override
  String get name => 'up';

  @override
  String get description => 'Start the Applad configuration server natively.';

  @override
  Future<void> run() async {
    Output.info('\x1b[32mBooting Applad Core Server...\x1b[0m');

    // We pass APPLAD_WORKSPACE_ROOT so backend ConfigLoader knows to parse YAMLs precisely
    // from the user's current CLI execution point, securely decoupling it from the server source.
    final workspaceRoot = Directory.current.path;

    // For MVP phase natively, we look up the applad_server package relatively from the CLI script if possible
    // or assume we are spawned via melos inside the monorepo.
    // In production releases this will be a statically bundled artifact block!
    var serverPath =
        p.join(Directory.current.path, 'packages', 'applad_server');
    if (!Directory(serverPath).existsSync()) {
      serverPath =
          p.join(Directory.current.parent.path, 'packages', 'applad_server');
    }

    try {
      final process = await Process.start(
        'dart_frog',
        ['dev'],
        workingDirectory: serverPath,
        environment: {'APPLAD_WORKSPACE_ROOT': workspaceRoot},
        mode: ProcessStartMode.inheritStdio,
      );
      await process.exitCode;
    } catch (e) {
      Output.error(
          'Failed to spawn the Applad server. Ensure dart_frog_cli is installed.');
      Output.error(e.toString());
    }
  }
}
