import 'package:args/command_runner.dart';
import '../utils/output.dart';

final class InstructCommand extends Command<void> {
  InstructCommand() {
    argParser.addFlag(
      'dry-run',
      help: 'Show what would be changed without applying it.',
      negatable: false,
    );
    argParser.addOption(
      'context',
      abbr: 'c',
      help: 'Additional context file to include with the instruction.',
    );
    argParser.addOption(
      'model',
      help: 'AI model to use (default: configured in applad.yaml).',
    );
  }

  @override
  String get name => 'instruct';
  @override
  String get description =>
      'Use natural language to instruct Applad to make infrastructure changes. (Phase 5)';

  @override
  String get invocation => 'applad instruct "<instruction>"';

  @override
  Future<void> run() async {
    final dryRun = argResults!['dry-run'] as bool;
    final context = argResults!['context'] as String?;

    Output.info('applad instruct — coming in Phase 5 (AI assistant)');
    if (dryRun) Output.info('  (--dry-run flag registered)');
    if (context != null) Output.info('  (--context: $context)');
  }
}
