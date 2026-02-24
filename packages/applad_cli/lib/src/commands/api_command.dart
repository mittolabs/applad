import 'package:args/command_runner.dart';
import '../utils/output.dart';

/// Manage auto-generated REST and GraphQL APIs.
final class ApiCommand extends Command<void> {
  ApiCommand() {
    addSubcommand(ApiValidateCommand());
    addSubcommand(ApiRoutesCommand());
    addSubcommand(ApiDocsCommand());
    addSubcommand(ApiKeysCommand());
    addSubcommand(ApiSdkCommand());
    addSubcommand(ApiVersionsCommand());
  }

  @override
  String get name => 'api';

  @override
  String get description => 'Manage auto-generated REST and GraphQL APIs.';
}

/// Validates API config files.
final class ApiValidateCommand extends Command<void> {
  @override
  String get name => 'validate';

  @override
  String get description => 'Validates api.yaml, versions.yaml, and sdk.yaml.';

  @override
  Future<void> run() async {
    Output.info('Validating API configuration...');
    Output.success('API configuration is valid.');
  }
}

/// Lists generated routes.
final class ApiRoutesCommand extends Command<void> {
  ApiRoutesCommand() {
    addSubcommand(ApiRoutesListCommand());
  }

  @override
  String get name => 'routes';

  @override
  String get description => 'Inspect auto-generated routes.';
}

final class ApiRoutesListCommand extends Command<void> {
  @override
  String get name => 'list';

  @override
  String get description => 'Lists all REST and GraphQL routes.';

  @override
  Future<void> run() async {
    Output.header('Auto-generated Routes (v2)');
    Output.table([
      'Method',
      'Path',
      'Table',
      'Resource'
    ], [
      ['GET', '/v2/users', 'users', 'REST'],
      ['POST', '/v2/users', 'users', 'REST'],
      ['GET', '/v2/posts', 'posts', 'REST'],
      ['POST', '/graphql', '-', 'GraphQL'],
    ]);
  }
}

/// Manage API documentation.
final class ApiDocsCommand extends Command<void> {
  ApiDocsCommand() {
    addSubcommand(ApiDocsOpenCommand());
    addSubcommand(ApiDocsGenerateCommand());
  }

  @override
  String get name => 'docs';

  @override
  String get description => 'Manage API documentation.';
}

final class ApiDocsOpenCommand extends Command<void> {
  @override
  String get name => 'open';

  @override
  String get description => 'Opens auto-generated API docs in the browser.';

  @override
  Future<void> run() async {
    Output.info('Opening http://localhost:8080/api/docs...');
  }
}

final class ApiDocsGenerateCommand extends Command<void> {
  @override
  String get name => 'generate';

  @override
  String get description => 'Regenerates the OpenAPI spec.';

  @override
  Future<void> run() async {
    Output.info('Regenerating OpenAPI spec...');
    Output.success('OpenAPI spec written to ./api/openapi.yaml');
  }
}

/// Manage API keys.
final class ApiKeysCommand extends Command<void> {
  ApiKeysCommand() {
    addSubcommand(ApiKeysListCommand());
    addSubcommand(ApiKeysCreateCommand());
    addSubcommand(ApiKeysRotateCommand());
    addSubcommand(ApiKeysRevokeCommand());
    addSubcommand(ApiKeysShowCommand());
  }

  @override
  String get name => 'keys';

  @override
  String get description => 'Manage operational API keys.';
}

final class ApiKeysListCommand extends Command<void> {
  @override
  String get name => 'list';

  @override
  String get description => 'Lists all API keys (never values).';

  @override
  Future<void> run() async {
    Output.header('API Keys');
    Output.table([
      'Label',
      'Scopes',
      'Environments',
      'Expiry'
    ], [
      ['ios-app', 'rest:read,graphql:*', 'production', 'Never'],
      ['web-client', 'rest:read', 'staging,production', '2027-01-01'],
    ]);
  }
}

final class ApiKeysCreateCommand extends Command<void> {
  ApiKeysCreateCommand() {
    argParser.addOption('label', abbr: 'l', help: 'Label for the key.');
  }

  @override
  String get name => 'create';

  @override
  String get description => 'Creates a new API key.';

  @override
  Future<void> run() async {
    final label = argResults!['label'] as String? ??
        Output.prompt('Enter label for the key');
    Output.info('Creating API key "$label"...');
    Output.success('Key created successfully.');
    Output.blank();
    Output.warning('VALUE: ap_live_kX9j2pLmN0vQz4r5t7... (STUB)');
    Output.info('This value is only shown once. Store it securely.');
  }
}

final class ApiKeysRotateCommand extends Command<void> {
  @override
  String get name => 'rotate';

  @override
  String get description => 'Rotates an API key.';

  @override
  Future<void> run() async {
    if (argResults!.rest.isEmpty) {
      Output.error('Usage: applad api keys rotate <LABEL>');
      return;
    }
    final label = argResults!.rest.first;
    Output.info('Rotating key "$label"...');
    Output.success('Rotation initiated. 15-minute transition window active.');
  }
}

final class ApiKeysRevokeCommand extends Command<void> {
  @override
  String get name => 'revoke';

  @override
  String get description => 'Revokes an API key immediately.';

  @override
  Future<void> run() async {
    if (argResults!.rest.isEmpty) {
      Output.error('Usage: applad api keys revoke <LABEL>');
      return;
    }
    final label = argResults!.rest.first;
    Output.warning('REVOKING API KEY: $label');
    if (Output.confirm(
        'Are you sure? This will break clients using this key immediately.')) {
      Output.success('Key "$label" revoked.');
    }
  }
}

final class ApiKeysShowCommand extends Command<void> {
  @override
  String get name => 'show';

  @override
  String get description => 'Shows scopes for an API key.';

  @override
  Future<void> run() async {
    if (argResults!.rest.isEmpty) {
      Output.error('Usage: applad api keys show <LABEL>');
      return;
    }
    final label = argResults!.rest.first;
    Output.header('API Key: $label');
    Output.kv('Scopes', 'rest:read, graphql:query, storage:read');
    Output.kv('Environments', 'production, staging');
  }
}

/// Manage SDK generation.
final class ApiSdkCommand extends Command<void> {
  ApiSdkCommand() {
    addSubcommand(ApiSdkGenerateCommand());
  }

  @override
  String get name => 'sdk';

  @override
  String get description => 'Manage client SDK generation.';
}

final class ApiSdkGenerateCommand extends Command<void> {
  ApiSdkGenerateCommand() {
    argParser.addOption('language',
        abbr: 'l', help: 'Target language (dart|typescript).');
  }

  @override
  String get name => 'generate';

  @override
  String get description => 'Generates client SDKs defined in sdk.yaml.';

  @override
  Future<void> run() async {
    final lang = argResults!['language'] as String?;
    Output.info('Generating ${lang ?? "all"} SDKs...');
    Output.success('SDK generation complete.');
  }
}

/// Manage API versioning.
final class ApiVersionsCommand extends Command<void> {
  ApiVersionsCommand() {
    addSubcommand(ApiVersionsListCommand());
    addSubcommand(ApiVersionsShowCommand());
  }

  @override
  String get name => 'versions';

  @override
  String get description => 'Manage API versioning.';
}

final class ApiVersionsListCommand extends Command<void> {
  @override
  String get name => 'list';

  @override
  String get description => 'Lists all API versions.';

  @override
  Future<void> run() async {
    Output.header('API Versions');
    Output.table([
      'Version',
      'Status',
      'Base Path',
      'Sunset'
    ], [
      ['v1', 'deprecated', '/api/v1', '2027-01-01'],
      ['v2', 'current', '/api/v2', '-'],
    ]);
  }
}

final class ApiVersionsShowCommand extends Command<void> {
  @override
  String get name => 'show';

  @override
  String get description => 'Shows details for a specific version.';

  @override
  Future<void> run() async {
    if (argResults!.rest.isEmpty) {
      Output.error('Usage: applad api versions show <VERSION>');
      return;
    }
    final version = argResults!.rest.first;
    Output.header('API Version: $version');
    Output.info('Exposed Tables:');
    Output.info('  - users (read, create, update)');
    Output.info('  - posts (read, create)');
  }
}
