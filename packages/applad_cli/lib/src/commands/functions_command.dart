import 'dart:io';
import 'package:args/command_runner.dart';
import 'package:path/path.dart' as p;
import '../utils/output.dart';
import '../utils/config_finder.dart';

/// `applad functions` — group for function management.
final class FunctionsCommand extends Command<void> {
  FunctionsCommand() {
    addSubcommand(FunctionsCreateCommand());
  }

  @override
  String get name => 'functions';

  @override
  String get description => 'Serverless functions management.';
}

/// `applad functions create` — guided function creation.
final class FunctionsCreateCommand extends Command<void> {
  @override
  String get name => 'create';

  @override
  String get description => 'Guided creation of a new serverless function.';

  @override
  Future<void> run() async {
    Output.header('Create Function');

    final projectDir = ConfigFinder.discoverProjectRoot();
    if (projectDir == null) {
      Output.error('No Applad project found.');
      return;
    }

    final projectName = p.basename(projectDir.path);
    Output.info('Selected project: $projectName');

    final functionsDir = Directory('${projectDir.path}/functions');
    if (!functionsDir.existsSync()) {
      Output.info('Creating functions/ directory...');
      functionsDir.createSync(recursive: true);
    }

    final name = Output.prompt('Function name', defaultValue: 'my-function');
    final yamlFile = File('${functionsDir.path}/$name.yaml');

    if (yamlFile.existsSync()) {
      Output.error('Function config "$name.yaml" already exists.');
      return;
    }

    final sourceType = Output.prompt('Source type (local/github/registry)',
        defaultValue: 'local');

    String sourceYaml = '';
    String? localSourcePath;

    if (sourceType == 'local') {
      localSourcePath = Output.prompt('Source path (relative to project root)',
          defaultValue: './src/functions/$name/main.dart');
      sourceYaml = '''
source:
  type: "local"
  path: "$localSourcePath"''';
    } else if (sourceType == 'github') {
      final repo = Output.prompt('GitHub Repo (org/repo)');
      final branch = Output.prompt('Branch', defaultValue: 'main');
      final path = Output.prompt('Path within repo');
      sourceYaml = '''
source:
  type: "github"
  repo: "$repo"
  branch: "$branch"
  path: "$path"
  ssh_key: "ci-github-actions"''';
    } else if (sourceType == 'registry') {
      final image = Output.prompt('Container Image');
      sourceYaml = '''
source:
  type: "registry"
  image: "$image"
  credentials: "registry-credentials"''';
    }

    final runtime =
        Output.prompt('Runtime (dart/node/python)', defaultValue: 'dart');
    final memory = Output.prompt('Memory limit (128mb/256mb/512mb/1gb)',
        defaultValue: '256mb');
    final timeout =
        int.tryParse(Output.prompt('Timeout in seconds', defaultValue: '30')) ??
            30;

    final content = '''
# ============================================================
# FUNCTION: $name
# Generated via applad functions create
# ============================================================

name: "$name"
runtime: "$runtime"
timeout: $timeout
memory: "$memory"

$sourceYaml

container:
  readonly_filesystem: true
  no_new_privileges: true
''';

    yamlFile.writeAsStringSync(content);
    Output.success('Created function config: ${yamlFile.path}');

    // If local, try to create source stub
    if (sourceType == 'local' && localSourcePath != null) {
      final absoluteSourcePath =
          p.normalize(p.join(projectDir.path, localSourcePath));
      final sourceFile = File(absoluteSourcePath);

      if (!sourceFile.existsSync()) {
        try {
          sourceFile.parent.createSync(recursive: true);
          sourceFile.writeAsStringSync(_getStub(runtime));
          Output.success(
              'Created source stub: ${p.relative(absoluteSourcePath)}');
        } catch (e) {
          Output.warning(
              'Could not create source stub at $absoluteSourcePath: $e');
        }
      }
    }

    Output.blank();
    Output.nextSteps([
      'Edit ${yamlFile.path} to refine your configuration.',
      if (sourceType == 'local') 'Implement your logic in $localSourcePath',
      'Run `applad up` to deploy.'
    ]);
  }

  String _getStub(String runtime) {
    return switch (runtime) {
      'dart' =>
        "import 'package:applad_function/applad_function.dart';\n\nFuture<AppladResponse> handler(AppladRequest request) async {\n  return AppladResponse.json({'message': 'Hello from Applad!'});\n}\n",
      'node' =>
        "exports.handler = async (event) => {\n  return { statusCode: 200, body: JSON.stringify({ message: 'Hello from Applad!' }) };\n};\n",
      'python' =>
        "def handler(event, context):\n    return { 'statusCode': 200, 'body': 'Hello from Applad!' }\n",
      _ => '',
    };
  }
}
