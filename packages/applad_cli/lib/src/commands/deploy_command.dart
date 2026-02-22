import 'package:args/command_runner.dart';
import 'package:applad_core/applad_core.dart';
import '../utils/output.dart';
import '../utils/config_finder.dart';

/// `applad deploy` — Orchestrates a complete deployment pipeline for artifacts.
final class DeployCommand extends Command<void> {
  DeployCommand() {
    addSubcommand(DeployListCommand());
    addSubcommand(DeployRunCommand());
    addSubcommand(DeployStatusCommand());
    addSubcommand(DeployLogsCommand());
  }

  @override
  String get name => 'deploy';

  @override
  String get description =>
      'Manages all deployment pipelines for apps and sites.';
}

final class DeployListCommand extends Command<void> {
  @override
  String get name => 'list';

  @override
  String get description =>
      'Lists all deployment pipelines defined in the active project.';

  @override
  Future<void> run() async {
    const finder = ConfigFinder();
    final String rootPath;
    try {
      rootPath = finder.requireRoot();
    } catch (e) {
      Output.error(e.toString());
      return;
    }

    final merger = ConfigMerger();
    final config = merger.merge(rootPath);

    if (config.deployments.isEmpty) {
      Output.info('No deployment pipelines found in this project.');
      return;
    }

    Output.header('Deployment Pipelines');
    for (final dep in config.deployments) {
      Output.info('- ${dep.name} (platform: ${dep.platform})');
    }
  }
}

final class DeployRunCommand extends Command<void> {
  DeployRunCommand() {
    argParser.addOption(
      'env',
      abbr: 'e',
      help: 'Environment to deploy to (e.g., production)',
      defaultsTo: 'production',
    );
    argParser.addOption(
      'path',
      abbr: 'p',
      help: 'Path to the root config directory (default: auto-discover).',
    );
  }

  @override
  String get name => 'run';

  @override
  String get description => 'Triggers a deployment pipeline by name.';

  @override
  Future<void> run() async {
    if (argResults!.rest.isEmpty) {
      Output.error(
          'Please specify a pipeline name to run. e.g applad deploy run web');
      return;
    }
    final pipelineName = argResults!.rest.first;
    final envName = argResults!['env'] as String;

    Output.header('Running deployment "$pipelineName" to $envName');

    final configPath = argResults!['path'] as String?;
    final finder = const ConfigFinder();

    final String rootPath;
    try {
      rootPath = configPath ?? finder.requireRoot();
    } catch (e) {
      Output.error(e.toString());
      return;
    }

    final merger = ConfigMerger();
    final config = merger.merge(rootPath);

    final targetEnv = Environment.fromString(envName);
    final envConfig = config.project.environments[targetEnv];

    if (envConfig == null) {
      Output.error('Environment "$envName" is not configured in project.yaml');
      return;
    }

    final host = envConfig.host;
    if (host == null) {
      Output.error(
          'No "host" specified in the infrastructure configuration for $envName.');
      return;
    }

    // Resolve pipeline
    DeploymentConfig? targetDep;
    try {
      targetDep = config.deployments.firstWhere((d) => d.name == pipelineName);
    } catch (_) {}

    if (targetDep == null) {
      Output.error('No deployment pipeline found named "$pipelineName".');
      return;
    }
  }

  // Temporarily disabled since the infrastructure handling now branches from `targetDep`
  // Future deployment target support comes in next phase.
}

final class DeployStatusCommand extends Command<void> {
  @override
  String get name => 'status';
  @override
  String get description =>
      'Shows the current status of the most recent deployment.';

  @override
  Future<void> run() async {
    Output.info('Deployments tracking currently unimplemented natively.');
  }
}

final class DeployLogsCommand extends Command<void> {
  @override
  String get name => 'logs';
  @override
  String get description => 'Shows the deployment log for a specific pipeline.';

  @override
  Future<void> run() async {
    Output.info('Deployment streams currently unimplemented natively.');
  }
}
