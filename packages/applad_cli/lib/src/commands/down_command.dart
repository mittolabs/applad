import 'dart:io';
import 'package:args/command_runner.dart';
import 'package:applad_core/applad_core.dart';
import 'package:path/path.dart' as p;
import '../utils/output.dart';
import '../utils/config_finder.dart';

final class DownCommand extends Command<void> {
  DownCommand() {
    argParser.addOption(
      'env',
      abbr: 'e',
      help: 'Environment to tear down (e.g., production)',
      defaultsTo: 'local',
    );
    argParser.addOption(
      'path',
      abbr: 'p',
      help: 'Path to the root config directory (default: auto-discover).',
    );
    argParser.addFlag(
      'dry-run',
      help:
          'Shows exactly what applad down would do without actually doing anything.',
      negatable: false,
    );
  }

  @override
  String get name => 'down';

  @override
  String get description =>
      'Tears down all infrastructure for a specific environment.';

  @override
  Future<void> run() async {
    final envName = argResults!['env'] as String;
    final dryRun = argResults!['dry-run'] as bool;
    final configPath = argResults!['path'] as String?;

    Output.header('Tearing down environment: $envName');

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

    if (envConfig.infraTarget == 'local') {
      await _runLocal(rootPath, dryRun);
    } else if (envConfig.infraTarget == 'vps') {
      await _runVps(envConfig, config, rootPath, dryRun);
    } else {
      Output.error('Unsupported infraTarget: ${envConfig.infraTarget}');
    }
  }

  Future<void> _runLocal(String workspaceRoot, bool dryRun) async {
    final composeFile =
        File(p.join(workspaceRoot, '.applad', 'local', 'docker-compose.yml'));

    if (dryRun) {
      Output.info(
          'DRY-RUN: Would run `docker compose down` for local server at ${composeFile.path}');
      return;
    }

    if (!composeFile.existsSync()) {
      Output.error('Could not find local deployment at: ${composeFile.path}');
      Output.info('Is the server running? Use `applad up` to start it.');
      return;
    }

    Output.info('Stopping local Applad Core Server via Docker Compose...');

    try {
      final process = await Process.start(
        'docker',
        ['compose', '-f', composeFile.path, 'down'],
        mode: ProcessStartMode.inheritStdio,
      );
      final exitCode = await process.exitCode;
      if (exitCode == 0) {
        Output.success('Local infrastructure torn down successfully.');
      } else {
        Output.error('Docker Compose failed with exit code $exitCode');
      }
    } catch (e) {
      Output.error('Failed to launch Docker Compose.');
      Output.error(e.toString());
    }
  }

  Future<void> _runVps(ProjectEnvironmentConfig envConfig, ApplAdConfig config,
      String rootPath, bool dryRun) async {
    final host = envConfig.host;
    final user = envConfig.user ?? 'root';

    if (host == null) {
      Output.error('No "host" specified in the infrastructure configuration.');
      return;
    }

    if (dryRun) {
      Output.info(
          'DRY-RUN: Would SSH into $user@$host and stop Docker containers in /opt/applad');
      return;
    }

    Output.info('Connecting to $user@$host to tear down infrastructure...');

    try {
      final process = await Process.start(
          'ssh',
          [
            '-o',
            'StrictHostKeyChecking=no',
            '$user@$host',
            'cd /opt/applad && docker compose down'
          ],
          mode: ProcessStartMode.inheritStdio);

      final exitCode = await process.exitCode;
      if (exitCode == 0) {
        Output.success(
            'Remote infrastructure at $host torn down successfully.');
      } else {
        Output.error(
            'Remote teardown failed with exit code $exitCode. Is Docker Compose installed on the host?');
      }
    } catch (e) {
      Output.error('Failed to establish SSH connection to $host');
      Output.error(e.toString());
    }
  }
}
