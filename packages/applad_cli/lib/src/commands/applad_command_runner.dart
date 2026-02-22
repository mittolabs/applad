import 'package:args/command_runner.dart';
import '../utils/version.dart';
import 'init_command.dart';
import 'up_command.dart';
import 'config_command.dart';
import 'db_command.dart';
import 'messaging_command.dart';
import 'tables_command.dart';
import 'orgs_command.dart';
import 'projects_command.dart';
import 'deploy_command.dart';
import 'down_command.dart';
import 'status_command.dart';
import 'version_command.dart';

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
    addCommand(MessagingCommand());
    addCommand(TablesCommand());
    addCommand(OrgsCommand());
    addCommand(ProjectsCommand());
    addCommand(DeployCommand());
    addCommand(DownCommand());
    addCommand(StatusCommand());
    addCommand(VersionCommand());
  }
  String _getAppladLogo() {
    const logo = '''
      █████╗ ██████╗ ██████╗ ██╗      █████╗ ██████╗ 
     ██╔══██╗██╔══██╗██╔══██╗██║     ██╔══██╗██╔══██╗
     ███████║██████╔╝██████╔╝██║     ███████║██║  ██║
     ██╔══██║██╔═══╝ ██╔═══╝ ██║     ██╔══██║██║  ██║
     ██║  ██║██║     ██║     ███████╗██║  ██║██████╔╝
     ╚═╝  ╚═╝╚═╝     ╚═╝     ╚══════╝╚═╝  ╚═╝╚═════╝ ''';

    // Simple, clean cyan foreground text color
    return '\x1b[36m$logo\x1b[0m\n';
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
