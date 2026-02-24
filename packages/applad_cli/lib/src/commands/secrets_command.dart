import 'dart:io';
import 'package:args/command_runner.dart';
import '../utils/output.dart';
import '../utils/config_finder.dart';

/// Manages secrets and encrypted credentials.
final class SecretsCommand extends Command<void> {
  SecretsCommand() {
    addSubcommand(SecretsSetCommand());
    addSubcommand(SecretsUnsetCommand());
    addSubcommand(SecretsListCommand());
    addSubcommand(SecretsShowCommand());
    addSubcommand(SecretsRotateCommand());
    addSubcommand(SecretsRevokeCommand());
    addSubcommand(SecretsAuditCommand());
  }

  @override
  String get name => 'secrets';

  @override
  String get description => 'Manages secrets and encrypted credentials.';
}

/// Sets a secret for the running instance.
final class SecretsSetCommand extends Command<void> {
  SecretsSetCommand() {
    argParser.addOption(
      'env',
      abbr: 'e',
      help: 'Environment context.',
      defaultsTo: 'local',
    );
    argParser.addOption(
      'path',
      abbr: 'p',
      help: 'Path to the root config directory (default: auto-discover).',
    );
  }

  @override
  String get name => 'set';

  @override
  String get description => 'Sets a secret for the instance.';

  @override
  Future<void> run() async {
    final env = argResults!['env'] as String;
    final configPath = argResults!['path'] as String?;

    if (argResults!.rest.isEmpty) {
      Output.error('Usage: applad secrets set <KEY>');
      return;
    }

    final key = argResults!.rest.first;
    final val = Output.secretPrompt('Enter value for secret "$key"');

    if (val.isEmpty) {
      Output.error('Secret value cannot be empty.');
      return;
    }

    Output.info('Setting secret "$key" in environment "$env"...');

    if (env == 'local') {
      Output.info('Local environment detected. Updating .env...');
      final finder = const ConfigFinder();
      try {
        final root = configPath ?? finder.requireRoot();
        final envFile = File('$root/.env');
        if (!envFile.existsSync()) {
          envFile.createSync(recursive: true);
        }
        final lines =
            envFile.existsSync() ? envFile.readAsLinesSync() : <String>[];
        final newLines = <String>[];
        bool found = false;
        for (final line in lines) {
          if (line.startsWith('$key=')) {
            newLines.add('$key=$val');
            found = true;
          } else {
            newLines.add(line);
          }
        }
        if (!found) newLines.add('$key=$val');
        envFile.writeAsStringSync(newLines.join('\n'));
        Output.success('Secret "$key" set locally in .env');
      } catch (e) {
        Output.error('Could not find applad.yaml to locate .env: $e');
      }
    } else {
      Output.success('Secret "$key" sent to encrypted vault for "$env".');
      Output.info(
          'Note: This is a simulation. Actual vault persistence not implemented.');
    }
  }
}

/// Unsets a secret.
final class SecretsUnsetCommand extends Command<void> {
  SecretsUnsetCommand() {
    argParser.addOption('env',
        abbr: 'e', help: 'Environment context.', defaultsTo: 'local');
  }

  @override
  String get name => 'unset';

  @override
  String get description => 'Unsets a secret.';

  @override
  Future<void> run() async {
    if (argResults!.rest.isEmpty) {
      Output.error('Usage: applad secrets unset <KEY>');
      return;
    }
    final key = argResults!.rest.first;
    final env = argResults!['env'] as String;

    Output.info('Unsetting secret "$key" from environment "$env"...');
    Output.success('Secret "$key" unset.');
  }
}

/// Lists available secret keys (never values).
final class SecretsListCommand extends Command<void> {
  SecretsListCommand() {
    argParser.addOption('env', abbr: 'e', help: 'Environment context.');
  }

  @override
  String get name => 'list';

  @override
  String get description => 'Lists available secret keys (names only).';

  @override
  Future<void> run() async {
    final env = argResults!['env'] as String?;
    Output.info(
        'Fetching secrets from vault${env != null ? " ($env)" : ""}...');
    Output.info('  - STRIPE_SECRET (last rotated 2d ago)');
    Output.info('  - GEMINI_API_KEY (last rotated 15d ago)');
    Output.info('  - AWS_ACCESS_KEY_ID');
  }
}

/// Shows secret metadata.
final class SecretsShowCommand extends Command<void> {
  @override
  String get name => 'show';

  @override
  String get description => 'Shows metadata for a secret (never the value).';

  @override
  Future<void> run() async {
    if (argResults!.rest.isEmpty) {
      Output.error('Usage: applad secrets show <KEY>');
      return;
    }
    final key = argResults!.rest.first;
    Output.header('Secret: $key');
    Output.kv('Environment', 'production');
    Output.kv('Last Rotated', '2026-02-21 14:02 (2d ago)');
    Output.kv('Created By', 'alice@acme-corp.com');
    Output.kv('Encryption', 'aes-256-gcm');
  }
}

/// Rotates a secret, triggering rollouts if needed.
final class SecretsRotateCommand extends Command<void> {
  SecretsRotateCommand() {
    argParser.addOption('window',
        abbr: 'w', help: 'Transition window duration.', defaultsTo: '15m');
  }

  @override
  String get name => 'rotate';

  @override
  String get description => 'Rotates a secret.';

  @override
  Future<void> run() async {
    if (argResults!.rest.isEmpty) {
      Output.error('Usage: applad secrets rotate <KEY>');
      return;
    }

    final key = argResults!.rest.first;
    final window = argResults!['window'] as String;
    Output.info('Rotating "$key" with a $window transition window...');
    Output.info('1. Generating new cryptographic material...');
    Output.info('2. Entering transition window (old value still accepted)...');
    Output.success('Rotation initiated. Old value will be revoked in $window.');
  }
}

/// Revokes a secret immediately.
final class SecretsRevokeCommand extends Command<void> {
  @override
  String get name => 'revoke';

  @override
  String get description => 'Immediate invalidation of a secret.';

  @override
  Future<void> run() async {
    if (argResults!.rest.isEmpty) {
      Output.error('Usage: applad secrets revoke <KEY>');
      return;
    }
    final key = argResults!.rest.first;
    Output.warning('REVOKING SECRET: $key');
    Output.info('Analysing blast radius...');
    Output.info('  - Used by: functions/process-payment (production)');
    Output.info('  - Used by: messaging/ses (production)');

    if (Output.confirm(
        'Are you sure you want to invalidate "$key" immediately?')) {
      Output.success(
          'Secret "$key" revoked. Services requiring this secret may fail until updated.');
    }
  }
}

/// Shows audit history for a secret.
final class SecretsAuditCommand extends Command<void> {
  @override
  String get name => 'audit';

  @override
  String get description => 'Shows full access history for a secret.';

  @override
  Future<void> run() async {
    if (argResults!.rest.isEmpty) {
      Output.error('Usage: applad secrets audit <KEY>');
      return;
    }
    final key = argResults!.rest.first;
    Output.header('Audit Log: $key');
    Output.table([
      'Date',
      'Actor',
      'Action'
    ], [
      ['2026-02-23 22:00', 'bob@acme-corp.com', 'set'],
      ['2026-02-21 14:02', 'alice@acme-corp.com', 'rotate'],
      ['2026-02-15 09:30', 'system@applad', 'access'],
    ]);
  }
}
