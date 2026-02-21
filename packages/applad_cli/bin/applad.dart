import 'dart:io';
import 'package:applad_cli/src/commands/applad_command_runner.dart';

Future<void> main(List<String> args) async {
  final runner = ApplAdCommandRunner();
  try {
    await runner.run(args);
  } catch (e) {
    stderr.writeln('Error: $e');
    exitCode = 1;
  }
}
