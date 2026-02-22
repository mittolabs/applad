import 'package:args/command_runner.dart';
import 'package:applad_core/applad_core.dart';
import '../../utils/output.dart';
import '../../utils/config_finder.dart';

/// `applad tables list` — List all database tables defined in config.
final class TablesListCommand extends Command<void> {
  @override
  String get name => 'list';

  @override
  String get description => 'List all database tables defined in config.';

  TablesListCommand() {
    argParser.addOption(
      'path',
      abbr: 'p',
      help: 'Path to the root config directory (default: auto-discover).',
    );
  }

  @override
  Future<void> run() async {
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

    final tables = config.tables;

    if (tables.isEmpty) {
      Output.info('No database tables defined in configuration.');
      return;
    }

    Output.header('Database Tables');
    for (final table in tables) {
      Output.info('• ${table.name}');
    }
    Output.info('');
    Output.info('Total: ${tables.length} tables');
  }
}
