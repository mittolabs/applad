import 'package:args/command_runner.dart';
import '../utils/version.dart';
import 'init_command.dart';
import 'up_command.dart';
import 'config_command.dart';
import 'db_command.dart';
import 'auth_command.dart';
import 'storage_command.dart';
import 'functions_command.dart';
import 'workflows_command.dart';
import 'deploy_command.dart';
import 'hosting_command.dart';
import 'flags_command.dart';
import 'messaging_command.dart';
import 'instruct_command.dart';
import 'keys_command.dart';
import 'audit_command.dart';
import 'secrets_command.dart';
import 'users_command.dart';
import 'export_command.dart';
import 'serve_command.dart';
import 'tables_command.dart';
import 'orgs_command.dart';
import 'projects_command.dart';

/// The root command runner for the `applad` CLI.
final class ApplAdCommandRunner extends CommandRunner<void> {
  ApplAdCommandRunner()
      : super(
          'applad',
          'Open-source BaaS + IaC + AI assistant — manage your backend with config.',
        ) {
    argParser.addFlag(
      'version',
      abbr: 'v',
      negatable: false,
      help: 'Print the current version.',
    );
    argParser.addFlag(
      'verbose',
      negatable: false,
      help: 'Enable verbose output.',
    );
    argParser.addOption(
      'config',
      help: 'Path to the root applad.yaml (default: auto-discover from CWD).',
    );
    argParser.addOption(
      'project',
      help:
          'Active project context (can also be set via APPLAD_PROJECT env var).',
    );
    argParser.addOption(
      'org',
      help: 'Active organization context.',
    );
    argParser.addOption(
      'env',
      help: 'Active environment context (default: development).',
    );

    addCommand(InitCommand());
    addCommand(UpCommand());
    addCommand(ConfigCommand());
    addCommand(DbCommand());
    addCommand(AuthCommand());
    addCommand(StorageCommand());
    addCommand(FunctionsCommand());
    addCommand(WorkflowsCommand());
    addCommand(DeployCommand());
    addCommand(HostingCommand());
    addCommand(FlagsCommand());
    addCommand(MessagingCommand());
    addCommand(InstructCommand());
    addCommand(KeysCommand());
    addCommand(AuditCommand());
    addCommand(SecretsCommand());
    addCommand(UsersCommand());
    addCommand(ExportCommand());
    addCommand(ServeCommand());
    addCommand(TablesCommand());
    addCommand(OrgsCommand());
    addCommand(ProjectsCommand());
  }
  String _getAppladLogo() {
    const logo = '''
      █████╗ ██████╗ ██████╗ ██╗      █████╗ ██████╗ 
     ██╔══██╗██╔══██╗██╔══██╗██║     ██╔══██╗██╔══██╗
     ███████║██████╔╝██████╔╝██║     ███████║██║  ██║
     ██╔══██║██╔═══╝ ██╔═══╝ ██║     ██╔══██║██║  ██║
     ██║  ██║██║     ██║     ███████╗██║  ██║██████╔╝
     ╚═╝  ╚═╝╚═╝     ╚═╝     ╚══════╝╚═╝  ╚═╝╚═════╝ ''';

    final StringBuffer buffer = StringBuffer();
    final lines = logo.split('\n');

    for (int i = 0; i < lines.length; i++) {
      final line = lines[i];
      final runes = line.runes.toList();
      for (int j = 0; j < runes.length; j++) {
        final double t = j / (runes.isEmpty ? 1 : runes.length);

        // Gradient from Cyan (0,255,255) to Purple/Pink (255,0,255)
        final int r = (0 + t * 255).round().clamp(0, 255);
        final int g = (255 - t * 255).round().clamp(0, 255);
        final int b = 255;

        buffer.write('\x1b[38;2;$r;$g;${b}m${String.fromCharCode(runes[j])}');
      }
      buffer.write('\x1b[0m\n');
    }

    return buffer.toString();
  }

  @override
  void printUsage() {
    print(_getAppladLogo());
    print('\x1b[1m* Welcome to Applad AI-Native CLI!\x1b[0m\n');
    super.printUsage();
  }

  @override
  Future<void> run(Iterable<String> args) async {
    final argResults = parse(args);

    if (argResults['version'] == true) {
      print('applad v$kApplAdVersion');
      return;
    }

    await super.run(args);
  }

  @override
  String get usageFooter => '\nDocumentation: https://applad.dev\n'
      'Report issues: https://github.com/mittolabs/applad/issues';
}
