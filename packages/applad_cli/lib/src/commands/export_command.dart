import 'package:args/command_runner.dart';
import '../utils/output.dart';

final class ExportCommand extends Command<void> {
  @override
  String get name => 'export';
  @override
  String get description => 'Export all project data. (Phase 4)';
  @override
  Future<void> run() async => Output.info('applad export — coming in Phase 4');
}
