import 'package:args/command_runner.dart';
import '../utils/output.dart';

final class AuditCommand extends Command<void> {
  @override
  String get name => 'audit';
  @override
  String get description => 'View the audit log. (Phase 2)';
  @override
  Future<void> run() async => Output.info('applad audit — coming in Phase 2');
}
