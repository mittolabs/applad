import 'dart:convert';
import 'dart:io';
import 'package:args/command_runner.dart';
import 'package:applad_core/applad_core.dart';
import 'package:http/http.dart' as http;
import '../utils/output.dart';
import '../utils/config_finder.dart';

final class StatusCommand extends Command<void> {
  StatusCommand() {
    argParser.addOption(
      'env',
      abbr: 'e',
      help: 'Environment to check (e.g., production)',
      defaultsTo: 'local',
    );
    argParser.addOption(
      'path',
      abbr: 'p',
      help: 'Path to the root config directory (default: auto-discover).',
    );
  }

  @override
  String get name => 'status';

  @override
  String get description =>
      'Shows the health and connectivity status of your environment.';

  @override
  Future<void> run() async {
    final envName = argResults!['env'] as String;
    final configPath = argResults!['path'] as String?;

    Output.header('Checking environment status: $envName');

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
      await _runLocal(rootPath);
    } else if (envConfig.infraTarget == 'vps') {
      await _runVps(envConfig, config);
    } else {
      Output.error('Unsupported infraTarget: ${envConfig.infraTarget}');
    }
  }

  Future<void> _runLocal(String workspaceRoot) async {
    // 1. Check Docker container
    final containerName = 'applad_server_local';
    Output.info('Infrastructure: Docker Compose (Local)');

    try {
      final dockerResult = await Process.run('docker',
          ['ps', '--filter', 'name=$containerName', '--format', '{{.Status}}']);

      final status = dockerResult.stdout.toString().trim();
      if (dockerResult.exitCode != 0 || status.isEmpty) {
        Output.error('Status: OFFLINE');
        if (dockerResult.exitCode != 0) {
          Output.info('Failed to query Docker daemon. Is it running?');
        } else {
          Output.info(
              'Container "$containerName" is not running. Start it with `applad up`.');
        }
        return;
      } else {
        Output.success('Status: RUNNING ($status)');
      }
    } catch (e) {
      Output.error('Failed to query Docker. Is it installed and running?');
      return;
    }

    // 2. Check API Health
    Output.info('API Gateway: http://localhost:8080');
    try {
      final response = await http
          .get(Uri.parse('http://localhost:8080/v1/health'))
          .timeout(const Duration(seconds: 3));

      if (response.statusCode == 200) {
        final data = jsonDecode(response.body) as Map<String, dynamic>;
        Output.success('Health Check: OK (v${data['version']})');

        final configStatus = data['config'] as String?;
        if (configStatus == 'loaded') {
          Output.success('Configuration: LOADED');
        } else {
          Output.error('Configuration: NOT LOADED');
        }
      } else {
        Output.error('Health Check: FAILED (HTTP ${response.statusCode})');
      }
    } catch (e) {
      Output.error('Health Check: UNREACHABLE (Timeout or Connection Refused)');
    }
  }

  Future<void> _runVps(
      ProjectEnvironmentConfig envConfig, ApplAdConfig config) async {
    final host = envConfig.host;
    final user = envConfig.user ?? 'root';
    final port = 8080;

    Output.info('Infrastructure: VPS ($user@$host)');

    if (host == null) {
      Output.error('No "host" specified in the infrastructure configuration.');
      return;
    }

    // 1. Check Docker container on VPS via SSH
    try {
      final sshResult = await Process.run('ssh', [
        '-o',
        'BatchMode=yes',
        '-o',
        'ConnectTimeout=5',
        '-o',
        'StrictHostKeyChecking=no',
        '$user@$host',
        'docker ps --filter name=applad_server --format "{{.Status}}"'
      ]);

      if (sshResult.exitCode != 0) {
        Output.error('SSH Connection: FAILED');
        Output.error(sshResult.stderr.toString().trim());
        return;
      }

      final status = sshResult.stdout.toString().trim();
      if (status.isEmpty) {
        Output.error('Status: OFFLINE (Container not found)');
      } else {
        Output.success('Status: RUNNING ($status)');
      }
    } catch (e) {
      Output.error('Failed to establish SSH connection to $host');
      return;
    }

    // 2. Ping API Gateway remotely
    final remoteUrl = 'http://$host:$port/v1/health';
    Output.info('API Gateway: $remoteUrl');
    try {
      final response = await http
          .get(Uri.parse(remoteUrl))
          .timeout(const Duration(seconds: 5));

      if (response.statusCode == 200) {
        final data = jsonDecode(response.body) as Map<String, dynamic>;
        Output.success('Health Check: OK (v${data['version']})');
      } else {
        Output.error('Health Check: FAILED (HTTP ${response.statusCode})');
      }
    } catch (e) {
      Output.error('Health Check: UNREACHABLE (Access blocked or server down)');
    }
  }
}
