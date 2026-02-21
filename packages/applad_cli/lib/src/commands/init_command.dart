import 'dart:io';
import 'package:args/command_runner.dart';
import 'package:mason/mason.dart';
import '../utils/output.dart';
import '../templates/applad_init_brick_bundle.dart';

/// `applad init` — scaffolds a new Applad project config tree using Mason.
final class InitCommand extends Command<void> {
  InitCommand() {
    argParser.addFlag(
      'yes',
      abbr: 'y',
      help: 'Accept all defaults without prompting.',
      negatable: false,
    );
    argParser.addOption(
      'org',
      help: 'Organization name.',
    );
    argParser.addOption(
      'project',
      help: 'Project name.',
    );
  }

  @override
  String get name => 'init';

  @override
  String get description =>
      'Scaffold a new Applad project config tree in the current directory.';

  @override
  Future<void> run() async {
    Output.header('Applad Init');
    Output.info('Setting up a new Applad project...');
    Output.blank();

    final useDefaults = argResults!['yes'] as bool;

    // Gather project details
    final orgName = (argResults!['org'] as String?) ??
        (useDefaults
            ? 'my-org'
            : Output.prompt('Organization name', defaultValue: 'my-org'));

    final projectName = (argResults!['project'] as String?) ??
        (useDefaults
            ? 'my-project'
            : Output.prompt('Project name', defaultValue: 'my-project'));

    final dbAdapter = useDefaults
        ? 'sqlite'
        : Output.prompt('Database adapter (sqlite/postgres/mysql)',
            defaultValue: 'sqlite');

    final orgId = _toId(orgName);
    final projectId = _toId(projectName);

    Output.blank();
    Output.info('Generating project structure...');

    final generator = await MasonGenerator.fromBundle(appladInitBrickBundle);
    final target = DirectoryGeneratorTarget(Directory.current);

    await generator.generate(target, vars: <String, dynamic>{
      'org_id': orgId,
      'org_name': orgName,
      'project_id': projectId,
      'project_name': projectName,
      'db_adapter': dbAdapter,
      'is_sqlite': dbAdapter == 'sqlite',
    });

    Output.blank();
    Output.success('Project initialized successfully!');
    Output.blank();
    Output.kv('Org', '$orgName ($orgId)');
    Output.kv('Project', '$projectName ($projectId)');
    Output.kv('Database', dbAdapter);

    Output.nextSteps([
      'Review your config in orgs/$orgId/projects/$projectId/',
      'Run `applad config validate` to check your config',
      'Run `applad up` to start the server',
    ]);
  }

  String _toId(String name) {
    return name
        .toLowerCase()
        .replaceAll(RegExp(r'[^a-z0-9]+'), '-')
        .replaceAll(RegExp(r'^-+|-+$'), '');
  }
}
