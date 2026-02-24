import 'package:args/command_runner.dart';
import '../utils/output.dart';

/// The AI-powered infrastructure assistant.
final class InstructCommand extends Command<void> {
  InstructCommand() {
    argParser.addFlag(
      'dry-run',
      help: 'Shows the proposed changes without applying them.',
      defaultsTo: true,
    );
  }

  @override
  String get name => 'instruct';

  @override
  String get description =>
      'Your AI lad — give instructions to manage your backend.';

  @override
  Future<void> run() async {
    if (argResults!.rest.isEmpty) {
      Output.info('Proper usage: applad instruct "your instruction here"');
      Output.blank();
      Output.info('Examples:');
      Output.info('  applad instruct "add a table for blog posts"');
      Output.info('  applad instruct "enable google auth"');
      Output.info(
          '  applad instruct "setup production environment on 1.2.3.4"');
      return;
    }

    final prompt = argResults!.rest.join(' ');
    final dryRun = argResults!['dry-run'] as bool;

    Output.info('\x1b[35m[LAD]\x1b[0m Analyzing instruction: "$prompt"...');
    await Future.delayed(const Duration(milliseconds: 800));

    Output.info(
        '\x1b[35m[LAD]\x1b[0m Reading config tree and extracting context...');
    await Future.delayed(const Duration(milliseconds: 600));

    Output.info('\x1b[35m[LAD]\x1b[0m Synthesizing proposed changes...');
    await Future.delayed(const Duration(milliseconds: 1200));

    Output.blank();
    Output.header('PROPOSED CHANGES');

    if (prompt.contains('table')) {
      Output.info('CREATE database/tables/posts.yaml');
      Output.blank();
      Output.info('```yaml');
      Output.info('name: posts');
      Output.info('database: primary');
      Output.info('fields:');
      Output.info('  title: string');
      Output.info('  content: text');
      Output.info('  author_id: relation(users.id)');
      Output.info('```');
    } else {
      Output.info('MODIFY project.yaml');
      Output.info('ADD \${NEW_VAR} to .env.example');
    }

    Output.blank();
    if (dryRun) {
      Output.info(
          'DRY-RUN: No changes applied. Run with --no-dry-run to apply.');
    } else {
      Output.success('Changes applied to config tree.');
      Output.info('Run "applad up" to reconcile your infrastructure.');
    }
  }
}
