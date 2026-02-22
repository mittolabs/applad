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

    if (argResults!['watch'] as bool) {
      await _watch(rootPath, envName, dryRun);
    } else {
      await _executeUp(rootPath, envName, dryRun);
    }
  }

  Future<void> _executeUp(String rootPath, String envName, bool dryRun) async {
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

  Future<void> _watch(String rootPath, String envName, bool dryRun) async {
    Output.info('Watching for changes in $rootPath...');

    // Initial run
    await _executeUp(rootPath, envName, dryRun);

    final watcher = Directory(rootPath).watch(recursive: true);
    await for (final event in watcher) {
      if (p.basename(event.path).startsWith('.') ||
          p.split(event.path).contains('.applad') ||
          p.split(event.path).contains('node_modules') ||
          p.split(event.path).contains('build') ||
          p.split(event.path).contains('dist')) {
        continue;
      }

      Output.info('Change detected: ${event.path}. Reconciling...');
      // Small debounce delay could be nice but let's keep it simple for now
      await _executeUp(rootPath, envName, dryRun);
    }
  }

  Future<void> _runLocal(String workspaceRoot, bool dryRun) async {
    Output.info('\x1b[32mBooting Applad Core Server Locally...\x1b[0m');

    if (dryRun) {
      Output.info(
          'DRY-RUN: Would boot Applad Core Server via Docker Compose in workspace $workspaceRoot');
      return;
    }

    var appladRepoRoot = Platform.environment['APPLAD_REPO_ROOT'];
    if (appladRepoRoot == null ||
        !Directory(p.join(appladRepoRoot, 'packages', 'applad_core'))
            .existsSync()) {
      appladRepoRoot = Directory.current.path;
      if (!Directory(p.join(appladRepoRoot, 'packages', 'applad_core'))
          .existsSync()) {
        // 1. Try parent
        final parent = Directory.current.parent.path;
        if (Directory(p.join(parent, 'packages', 'applad_core')).existsSync()) {
          appladRepoRoot = parent;
        } else {
          // 2. Try discovery relative to the CLI script
          try {
            final scriptPath = Platform.script.toFilePath();
            var dir = Directory(p.dirname(scriptPath));
            for (int i = 0; i < 6; i++) {
              if (Directory(p.join(dir.path, 'packages', 'applad_core'))
                  .existsSync()) {
                appladRepoRoot = dir.path;
                break;
              }
              if (dir.path == dir.parent.path) break;
              dir = dir.parent;
            }
          } catch (_) {}
        }
      }
    }

    // Verify we actually found it
    if (!Directory(p.join(appladRepoRoot!, 'packages', 'applad_core'))
        .existsSync()) {
      Output.error('Could not find Applad repository root.');
      Output.info(
          'Please set the APPLAD_REPO_ROOT environment variable to the path of your Applad clone.');
      return;
    }

    final localStagingDir =
        Directory(p.join(workspaceRoot, '.applad', 'local'));
    if (!localStagingDir.existsSync()) {
      localStagingDir.createSync(recursive: true);
    }

    final filteredPubspecYml = '''
name: applad_workspace
environment:
  sdk: ">=3.5.0 <4.0.0"

workspace:
  - packages/applad_core
  - packages/applad_cli
  - packages/applad_server
''';
    File(p.join(localStagingDir.path, 'root_pubspec.yaml'))
        .writeAsStringSync(filteredPubspecYml);

    final composeYml = '''
version: '3.8'

services:
  applad_server:
    image: dart:stable
    container_name: applad_server_local
    tty: true
    working_dir: /app/packages/applad_server
    command: /bin/sh -c "dart pub get && dart pub global activate dart_frog_cli && dart pub global run dart_frog_cli:dart_frog dev --port 8080 --hostname 0.0.0.0"
    volumes:
      - $appladRepoRoot:/app
      - $workspaceRoot:/app/config
      - ${p.join(localStagingDir.path, 'root_pubspec.yaml')}:/app/pubspec.yaml
    environment:
      - APPLAD_WORKSPACE_ROOT=/app/config
    ports:
      - "8080:8080"
''';

    final composeFile =
        File(p.join(localStagingDir.path, 'docker-compose.yml'));
    composeFile.writeAsStringSync(composeYml);

    Output.info('Booting Applad Core Server via Docker Compose...');

    try {
      final process = await Process.start(
        'docker',
        ['compose', '-f', composeFile.path, 'up', '-d', '--remove-orphans'],
      );
      final exitCode = await process.exitCode;
      if (exitCode != 0) {
        Output.error('Docker Compose failed with exit code $exitCode');
      } else {
        Output.success('Applad Core Server started via Docker Compose.');
        Output.info('API Gateway live at: http://localhost:8080');
      }
    } catch (e) {
      Output.error('Failed to launch Docker Compose.');
      Output.info('Ensure Docker is installed and running.');
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
