import 'dart:io';
import 'package:args/command_runner.dart';
import 'package:path/path.dart' as p;
import '../utils/output.dart';
import '../utils/config_finder.dart';

/// `applad storage` — group for storage management.
final class StorageCommand extends Command<void> {
  StorageCommand() {
    addSubcommand(StorageBucketsCommand());
  }

  @override
  String get name => 'storage';
  @override
  String get description => 'File storage management.';
}

/// `applad storage buckets` — group for bucket management.
final class StorageBucketsCommand extends Command<void> {
  StorageBucketsCommand() {
    addSubcommand(StorageBucketsCreateCommand());
  }

  @override
  String get name => 'buckets';
  @override
  String get description => 'Bucket management.';
}

/// `applad storage buckets create` — guided bucket creation.
final class StorageBucketsCreateCommand extends Command<void> {
  @override
  String get name => 'create';
  @override
  String get description => 'Guided creation of a new storage bucket.';

  @override
  Future<void> run() async {
    Output.header('Create Bucket');

    final projectDir = ConfigFinder.discoverProjectRoot();
    if (projectDir == null) {
      Output.error('No Applad project found.');
      return;
    }

    final projectName = p.basename(projectDir.path);
    Output.info('Selected project: $projectName');

    final storageDir = Directory('${projectDir.path}/storage/buckets');
    if (!storageDir.existsSync()) {
      Output.info('Storage namespace is not enabled. Creating directory...');
      storageDir.createSync(recursive: true);
    }

    final name = Output.prompt('Bucket name', defaultValue: 'avatars');
    final isPublic = Output.confirm('Public access?', defaultValue: false);

    final yamlFile = File('${storageDir.path}/$name.yaml');
    if (yamlFile.existsSync()) {
      Output.error('Bucket "$name" already exists.');
      return;
    }

    final content = '''
# ============================================================
# BUCKET: $name
# Generated via applad storage buckets create
# ============================================================

name: "$name"
public: $isPublic

# Security policies
policies:
  - role: "authenticated"
    allow: ["read", "write"]
  - role: "public"
    allow: [${isPublic ? '"read"' : ""}]
''';

    yamlFile.writeAsStringSync(content);

    Output.blank();
    Output.success('Created bucket config: ${yamlFile.path}');
    Output.blank();
    Output.nextSteps([
      'Edit ${yamlFile.path} to refine security policies.',
      'Run `applad up` to deploy.'
    ]);
  }
}
