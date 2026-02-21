import 'package:args/command_runner.dart';
import 'package:applad_core/applad_core.dart';
import '../utils/output.dart';
import '../utils/config_finder.dart';

/// `applad config` — config management subcommands.
final class ConfigCommand extends Command<void> {
  ConfigCommand() {
    addSubcommand(_ValidateSubcommand());
    addSubcommand(_PushSubcommand());
    addSubcommand(_PullSubcommand());
    addSubcommand(_DiffSubcommand());
  }

  @override
  String get name => 'config';

  @override
  String get description => 'Manage and validate your Applad configuration.';
}

/// `applad config validate` — validates the config tree.
final class _ValidateSubcommand extends Command<void> {
  _ValidateSubcommand() {
    argParser.addOption(
      'path',
      abbr: 'p',
      help: 'Path to the root config directory (default: auto-discover).',
    );
  }

  @override
  String get name => 'validate';

  @override
  String get description => 'Validate the config tree for errors and warnings.';

  @override
  Future<void> run() async {
    Output.header('Config Validation');

    final configPath = argResults!['path'] as String?;
    final finder = const ConfigFinder();

    final String rootPath;
    try {
      rootPath = configPath ?? finder.requireRoot();
    } catch (e) {
      Output.error(e.toString());
      return;
    }

    Output.info('Loading config from: $rootPath');

    try {
      final merger = ConfigMerger();
      final config = merger.merge(rootPath);

      Output.blank();
      Output.info('Config loaded:');
      Output.kv('Instance version', config.instance.version);
      Output.kv('Org', '${config.org.name} (${config.org.id})');
      Output.kv('Project', '${config.project.name} (${config.project.id})');
      Output.kv('Tables', '${config.tables.length}');
      Output.kv('Functions', '${config.functions.length}');
      Output.kv('Flags', '${config.flags.length}');
      Output.blank();

      final validator = const ConfigValidator();
      final violations = validator.validate(config);

      final warnings = violations
          .where((v) => v.severity == ViolationSeverity.warning)
          .toList();
      final infos = violations
          .where((v) => v.severity == ViolationSeverity.info)
          .toList();

      for (final warning in warnings) {
        Output.warning('${warning.path}: ${warning.message}');
      }
      for (final info in infos) {
        Output.info('${info.path}: ${info.message}');
      }

      if (warnings.isEmpty && infos.isEmpty) {
        Output.success('Config is valid! No issues found.');
      } else {
        Output.success('Config is valid with ${warnings.length} warning(s).');
      }
    } on ValidationError catch (e) {
      Output.error('Config validation failed:');
      for (final v in e.violations) {
        if (v.severity == ViolationSeverity.error) {
          Output.error('  ${v.path}: ${v.message}');
        } else if (v.severity == ViolationSeverity.warning) {
          Output.warning('  ${v.path}: ${v.message}');
        }
      }
    } on ConfigError catch (e) {
      Output.error('Failed to load config: ${e.message}');
    } catch (e, st) {
      Output.error('Unexpected error: $e');
      Output.error('Stacktrace: $st');
    }
  }
}

final class _PushSubcommand extends Command<void> {
  @override
  String get name => 'push';
  @override
  String get description =>
      'Push local config changes to the server. (Phase 2)';

  @override
  Future<void> run() async {
    Output.info('applad config push — coming in Phase 2');
  }
}

final class _PullSubcommand extends Command<void> {
  @override
  String get name => 'pull';
  @override
  String get description =>
      'Pull config from the server to local files. (Phase 2)';

  @override
  Future<void> run() async {
    Output.info('applad config pull — coming in Phase 2');
  }
}

final class _DiffSubcommand extends Command<void> {
  @override
  String get name => 'diff';
  @override
  String get description =>
      'Show diff between local config and server state. (Phase 2)';

  @override
  Future<void> run() async {
    Output.info('applad config diff — coming in Phase 2');
  }
}
