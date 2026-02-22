import 'dart:io';
import 'package:args/command_runner.dart';
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

    final projectDir = ConfigFinder.findProjectRoot();
    if (projectDir == null) {
      Output.error('No Applad project found (missing project.yaml).');
      return;
    }

    final functionsDir = Directory('${projectDir.path}/functions');
    if (!functionsDir.existsSync()) {
      Output.info('Functions namespace is not enabled. Creating directory...');
      functionsDir.createSync(recursive: true);
    }

    final name = Output.prompt('Function name', defaultValue: 'my-function');
    final runtime =
        Output.prompt('Runtime (dart/node/python)', defaultValue: 'dart');

    final yamlFile = File('${functionsDir.path}/$name.yaml');
    if (yamlFile.existsSync()) {
      Output.error('Function "$name" already exists.');
      return;
    }

    final memory = Output.prompt('Memory limit (128mb/256mb/512mb)',
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

# Container security — each function runs isolated
container:
  image: "applad/runtime-$runtime:latest"
  read_only_root: true
''';

    yamlFile.writeAsStringSync(content);

    Output.blank();
    Output.success('Created function config: ${yamlFile.path}');

    // Create source stub if local
    final sourcePath = '${functionsDir.path}/$name.${_getExtension(runtime)}';
    final sourceFile = File(sourcePath);
    if (!sourceFile.existsSync()) {
      sourceFile.writeAsStringSync(_getStub(runtime));
      Output.success('Created source stub: $sourcePath');
    }

    Output.blank();
    Output.nextSteps([
      'Edit ${yamlFile.path} to configure triggers or environment vars.',
      'Edit $sourcePath to implement your logic.',
      'Run `applad up` to deploy.'
    ]);
  }

  String _getExtension(String runtime) {
    return switch (runtime) {
      'dart' => 'dart',
      'node' => 'js',
      'python' => 'py',
      _ => 'txt',
    };
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
