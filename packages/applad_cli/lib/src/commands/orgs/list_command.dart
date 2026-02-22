import 'dart:io';
import 'package:args/command_runner.dart';
import 'package:path/path.dart' as p;
import '../../utils/output.dart';
import '../../utils/config_finder.dart';

/// `applad orgs list` — List all organizations in the current workspace.
final class OrgsListCommand extends Command<void> {
  @override
  String get name => 'list';

  @override
  String get description => 'List all organizations in the current workspace.';

  @override
  Future<void> run() async {
    final finder = const ConfigFinder();
    final String rootPath;
    try {
      rootPath = finder.requireRoot();
    } catch (e) {
      Output.error(e.toString());
      return;
    }

    final orgsDir = Directory(p.join(rootPath, 'orgs'));
    if (!orgsDir.existsSync()) {
      Output.info('No organizations found (orgs/ directory missing).');
      return;
    }

    final orgs = orgsDir
        .listSync()
        .whereType<Directory>()
        .where((d) => File(p.join(d.path, 'org.yaml')).existsSync())
        .toList();

    if (orgs.isEmpty) {
      Output.info('No organizations found.');
      return;
    }

    Output.header('Organizations');
    for (final org in orgs) {
      final name = p.basename(org.path);
      // We could parse org.yaml here for more info if needed
      Output.info('• $name');
    }
    Output.info('');
    Output.info('Total: ${orgs.length} organizations');
  }
}
