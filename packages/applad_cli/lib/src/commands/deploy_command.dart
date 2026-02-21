import 'dart:io';
import 'package:args/command_runner.dart';
import 'package:applad_core/applad_core.dart';
import 'package:path/path.dart' as p;
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

    if (config.hosting.isEmpty && config.deployments.isEmpty) {
      Output.info('No deployment pipelines found in this project.');
      return;
    }

    Output.header('Deployment Pipelines');
    for (final host in config.hosting) {
      Output.info('- ${host.name} (type: web)');
    }
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
    final user = envConfig.user ?? 'root';
    if (host == null) {
      Output.error(
          'No "host" specified in the infrastructure configuration for $envName.');
      return;
    }

    // Resolve pipeline
    HostingConfig? targetHost;
    try {
      targetHost = config.hosting.firstWhere((h) => h.name == pipelineName);
    } catch (_) {}

    if (targetHost != null) {
      await _deployWeb(targetHost, rootPath, user, host);
      return;
    }

    Output.error(
        'Deployment pipeline "$pipelineName" not found. Check applad deploy list.');
  }

  Future<void> _deployWeb(
      HostingConfig hosting, String rootPath, String user, String host) async {
    if (hosting.buildCommand != null) {
      Output.info(
          'Building ${hosting.name} frontend: ${hosting.buildCommand} ...');
      final buildProc = await Process.start('sh', ['-c', hosting.buildCommand!],
          workingDirectory: rootPath, mode: ProcessStartMode.inheritStdio);
      if (await buildProc.exitCode != 0) {
        return Output.error('Frontend build failed.');
      }
    }

    Output.info('Syncing build artifacts to remote host...');
    final outDir = hosting.outputDirectory;
    final targetDest = '/opt/applad/${hosting.name}_web';

    final mkdirProc = await Process.start(
        'ssh',
        [
          '-o',
          'StrictHostKeyChecking=no',
          '$user@$host',
          'mkdir -p $targetDest'
        ],
        mode: ProcessStartMode.inheritStdio);
    await mkdirProc.exitCode;

    await _rsync(p.join(rootPath, outDir), '$user@$host:$targetDest');

    Output.success('Deployed ${hosting.name} successfully!');
  }

  Future<void> _rsync(String source, String destination,
      {List<String> excludes = const [], bool isFile = false}) async {
    final args = ['-avz', '--delete', '-e', 'ssh -o StrictHostKeyChecking=no'];
    args.addAll(excludes);
    args.add(isFile ? source : '$source/');
    args.add(destination);

    final proc = await Process.start('rsync', args);
    final exit = await proc.exitCode;
    if (exit != 0) {
      throw Exception(
          'Rsync failed for \$source -> \$destination with exit code \$exit');
    }
  }
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
