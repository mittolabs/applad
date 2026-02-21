import 'dart:io';
import 'package:args/command_runner.dart';
import 'package:applad_core/applad_core.dart';
import 'package:path/path.dart' as p;
import '../utils/output.dart';
import '../utils/config_finder.dart';

final class UpCommand extends Command<void> {
  UpCommand() {
    argParser.addOption(
      'env',
      abbr: 'e',
      help: 'Environment to reconcile (e.g., production)',
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
          'Shows exactly what applad up would do without actually doing anything.',
      negatable: false,
    );
    argParser.addFlag(
      'watch',
      help: 'Watches config files for changes and re-runs applad up.',
      negatable: false,
    );
  }

  @override
  String get name => 'up';

  @override
  String get description =>
      'Applies your configuration to the active environment natively.';

  @override
  Future<void> run() async {
    final envName = argResults!['env'] as String;
    final dryRun = argResults!['dry-run'] as bool;
    final configPath = argResults!['path'] as String?;

    Output.header('Reconciling environment: $envName');

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
    Output.info('\x1b[32mBooting Applad Core Server Locally...\x1b[0m');

    if (dryRun) {
      Output.info(
          'DRY-RUN: Would spawn dart_frog dev in workspace $workspaceRoot');
      return;
    }

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

  Future<void> _runVps(ProjectEnvironmentConfig envConfig, ApplAdConfig config,
      String rootPath, bool dryRun) async {
    final host = envConfig.host;
    final user = envConfig.user ?? 'root';

    if (host == null) {
      Output.error('No "host" specified in the infrastructure configuration.');
      return;
    }

    if (dryRun) {
      Output.info('DRY-RUN: Would provision VPS at $user@$host');
      Output.info('DRY-RUN: Would sync Applad config tree and server source');
      Output.info(
          'DRY-RUN: Would trigger native Docker orchestration for the API server');
      return;
    }

    final deployStagingDir = Directory(p.join(rootPath, '.applad', 'deploy'));
    if (deployStagingDir.existsSync()) {
      deployStagingDir.deleteSync(recursive: true);
    }
    deployStagingDir.createSync(recursive: true);

    var appladRepoRoot = Directory.current.path;
    if (!Directory(p.join(appladRepoRoot, 'packages', 'applad_core'))
        .existsSync()) {
      appladRepoRoot = Directory.current.parent.path;
    }

    Output.info('Synthesizing Docker Compose Payload for Ubuntu VPS...');

    final composeYml = '''
version: '3.8'

services:
  applad_server:
    image: dart:stable
    container_name: applad_server
    working_dir: /app/packages/applad_server
    command: /bin/sh -c "dart pub get && dart pub global activate dart_frog_cli && dart_frog build && dart build/bin/server.dart"
    volumes:
      - ./packages/applad_server:/app/packages/applad_server
      - ./packages/applad_core:/app/packages/applad_core
      - ./config:/app/config
    environment:
      - APPLAD_WORKSPACE_ROOT=/app/config
    ports:
      - "8080:8080"
    restart: unless-stopped
''';

    File(p.join(deployStagingDir.path, 'docker-compose.yml'))
        .writeAsStringSync(composeYml);

    Output.info(
        'Establishing secure SSH pipeline to $user@$host and syncing payloads...');

    final mkdirProc = await Process.start(
        'ssh',
        [
          '-o',
          'StrictHostKeyChecking=no',
          '$user@$host',
          'mkdir -p /opt/applad/packages /opt/applad/config'
        ],
        mode: ProcessStartMode.inheritStdio);
    await mkdirProc.exitCode;

    await _rsync(p.join(appladRepoRoot, 'packages', 'applad_core'),
        '$user@$host:/opt/applad/packages/applad_core');
    await _rsync(p.join(appladRepoRoot, 'packages', 'applad_server'),
        '$user@$host:/opt/applad/packages/applad_server');

    await _rsync(rootPath, '$user@$host:/opt/applad/config', excludes: [
      '--exclude=node_modules',
      '--exclude=.applad',
      '--exclude=.git',
      '--exclude=dist',
      '--exclude=build'
    ]);

    await _rsync(p.join(deployStagingDir.path, 'docker-compose.yml'),
        '$user@$host:/opt/applad/docker-compose.yml',
        isFile: true);

    Output.info('Triggering native Docker orchestration on the remote host...');
    final upProc = await Process.start(
        'ssh',
        [
          '-o',
          'StrictHostKeyChecking=no',
          '$user@$host',
          'sed -i "s/resolution: workspace//g" /opt/applad/packages/applad_server/pubspec.yaml && cd /opt/applad && docker compose up -d applad_server'
        ],
        mode: ProcessStartMode.inheritStdio);

    if (await upProc.exitCode == 0) {
      Output.success('Infrastructure provisioned successfully!');
      Output.info('API Gateway live at: http://$host:8080');
    } else {
      Output.error(
          'Failed to trigger docker compose up on the remote instance. Is docker installed?');
    }
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
          'Rsync failed for $source -> $destination with exit code $exit');
    }
  }
}
