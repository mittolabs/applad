import 'package:args/command_runner.dart';
import '../utils/output.dart';

final class StorageCommand extends Command<void> {
  @override
  String get name => 'storage';
  @override
  String get description => 'Manage storage buckets and files. (Phase 2)';
  @override
  Future<void> run() async => Output.info('applad storage — coming in Phase 2');
}
