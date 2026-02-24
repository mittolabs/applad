import 'package:args/command_runner.dart';
import '../utils/version.dart';
import 'init_command.dart';
import 'up_command.dart';
import 'config_command.dart';
import 'db_command.dart';
import 'messaging_command.dart';
import 'orgs_command.dart';
import 'projects_command.dart';
import 'deploy_command.dart';
import 'down_command.dart';
import 'status_command.dart';
import 'version_command.dart';
import 'instruct_command.dart';
import 'auth_commands.dart';
import 'api_command.dart';
import 'access_command.dart';
import 'functions_command.dart';
import 'storage_command.dart';
import 'flags_command.dart';
import 'workflows_command.dart';
import 'workspace_command.dart';
import 'tables_command.dart';
import 'secrets_command.dart';
import 'env_command.dart';
import 'uninstall_command.dart';
import '../utils/interactive_shell.dart';
import '../utils/output.dart';
import '../security/trust_manager.dart';

/// The root command runner for the `applad` CLI.
final class ApplAdCommandRunner extends CommandRunner<void> {
  ApplAdCommandRunner()
      : super(
          'applad',
          'Open-source BaaS + IaC + AI assistant — manage your backend with config.',
        ) {
    argParser.addFlag(
      'version',
      negatable: false,
      help: 'Print the current version.',
    );
    argParser.addFlag(
      'verbose',
      abbr: 'v',
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
    addCommand(InstructCommand());
    addCommand(ConfigCommand());
    addCommand(DbCommand());
    addCommand(FunctionsCommand());
    addCommand(StorageCommand());
    addCommand(MessagingCommand());
    addCommand(FlagsCommand());
    addCommand(WorkflowsCommand());
    addCommand(OrgsCommand());
    addCommand(ProjectsCommand());
    addCommand(DeployCommand());
    addCommand(DownCommand());
    addCommand(StatusCommand());
    addCommand(VersionCommand());
    addCommand(LoginCommand());
    addCommand(LogoutCommand());
    addCommand(WhoamiCommand());
    addCommand(ApiCommand());
    addCommand(AccessCommand());
    addCommand(UninstallCommand());
    addCommand(EnvCommand());
    addCommand(SecretsCommand());
    addCommand(TablesCommand());
    addCommand(WorkspaceCommand());
  }

  /// Returns the ASCII logo for Applad.
  String getLogo() {
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
    print(getLogo());
    print('\x1b[1m* Welcome to Applad AI-Native CLI!\x1b[0m\n');
    super.printUsage();
  }

  @override
  Future<void> run(Iterable<String> args) async {
    final argResults = parse(args);

    // Only print version if --version is passed AND no command is specified.
    // This allows `applad up --version` to mean "show me the version of the engine" (if it were an option)
    // but more importantly avoids hijacking `applad up -v`.
    if (argResults['version'] == true && argResults.command == null) {
      print('applad v$kApplAdVersion');
      return;
    }

    if (args.isEmpty) {
      await InteractiveShell(this).start();
      return;
    }

    // Security check: Guard sensitive commands with workspace trust
    final isSensitive = ['up', 'init', 'deploy', 'down', 'access']
        .contains(argResults.command?.name);
    if (isSensitive && !TrustManager.isTrusted()) {
      if (!TrustManager.ensureTrusted()) {
        Output.error(
            'Aborted: This command requires workspace trust to proceed.');
        return;
      }
    }

    try {
      await super.run(args);
    } catch (e) {
      if (e is UsageException) {
        Output.error(e.message);
        printUsage();
      } else {
        rethrow;
      }
    }
  }

  @override
  String get usageFooter => '\nDocumentation: https://applad.dev\n'
      'Report issues: https://github.com/mittolabs/applad/issues';
}
