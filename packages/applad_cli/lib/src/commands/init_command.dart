import 'dart:io';
import 'package:args/command_runner.dart';
import 'package:path/path.dart' as p;
import '../utils/output.dart';

/// `applad init` — scaffolds a new Applad project config tree.
final class InitCommand extends Command<void> {
  InitCommand() {
    argParser.addFlag(
      'yes',
      abbr: 'y',
      help: 'Accept all defaults without prompting.',
      negatable: false,
    );
    argParser.addOption(
      'org',
      help: 'Organization name.',
    );
    argParser.addOption(
      'project',
      help: 'Project name.',
    );
  }

  @override
  String get name => 'init';

  @override
  String get description =>
      'Scaffold a new Applad project config tree in the current directory.';

  @override
  Future<void> run() async {
    Output.header('Applad Init');
    Output.info('Setting up a new Applad project...');
    Output.blank();

    final useDefaults = argResults!['yes'] as bool;

    // Gather project details
    final orgName = (argResults!['org'] as String?) ??
        (useDefaults
            ? 'my-org'
            : Output.prompt('Organization name', defaultValue: 'my-org'));

    final projectName = (argResults!['project'] as String?) ??
        (useDefaults
            ? 'my-project'
            : Output.prompt('Project name', defaultValue: 'my-project'));

    final dbAdapter = useDefaults
        ? 'sqlite'
        : Output.prompt('Database adapter (sqlite/postgres/mysql)',
            defaultValue: 'sqlite');

    final orgId = _toId(orgName);
    final projectId = _toId(projectName);
    final rootDir = Directory.current.path;

    Output.blank();
    Output.info('Creating project structure...');

    // Create directory structure
    _createDir(p.join(rootDir, 'orgs', orgId, 'projects', projectId, 'auth'));
    _createDir(
        p.join(rootDir, 'orgs', orgId, 'projects', projectId, 'database'));
    _createDir(p.join(rootDir, 'orgs', orgId, 'projects', projectId, 'tables'));
    _createDir(
        p.join(rootDir, 'orgs', orgId, 'projects', projectId, 'storage'));
    _createDir(
        p.join(rootDir, 'orgs', orgId, 'projects', projectId, 'functions'));
    _createDir(
        p.join(rootDir, 'orgs', orgId, 'projects', projectId, 'workflows'));
    _createDir(p.join(rootDir, 'orgs', orgId, 'projects', projectId, 'flags'));
    _createDir(
        p.join(rootDir, 'orgs', orgId, 'projects', projectId, 'hosting'));
    _createDir(
        p.join(rootDir, 'orgs', orgId, 'projects', projectId, 'messaging'));
    _createDir(p.join(rootDir, 'orgs', orgId, 'projects', projectId,
        'messaging', 'templates'));
    _createDir(p.join(rootDir, 'shared', 'roles'));

    // Write config files
    _writeFile(
      p.join(rootDir, 'applad.yaml'),
      _instanceConfig(),
    );
    _writeFile(
      p.join(rootDir, 'orgs', orgId, 'org.yaml'),
      _orgConfig(orgId, orgName),
    );
    _writeFile(
      p.join(rootDir, 'orgs', orgId, 'projects', projectId, 'project.yaml'),
      _projectConfig(projectId, projectName, orgId),
    );
    _writeFile(
      p.join(
          rootDir, 'orgs', orgId, 'projects', projectId, 'auth', 'auth.yaml'),
      _authConfig(),
    );
    _writeFile(
      p.join(rootDir, 'orgs', orgId, 'projects', projectId, 'database',
          'database.yaml'),
      _databaseConfig(dbAdapter),
    );
    _writeFile(
      p.join(rootDir, 'orgs', orgId, 'projects', projectId, 'messaging',
          'messaging.yaml'),
      _messagingConfig(),
    );

    Output.blank();
    Output.success('Project initialized successfully!');
    Output.blank();
    Output.kv('Org', '$orgName ($orgId)');
    Output.kv('Project', '$projectName ($projectId)');
    Output.kv('Database', dbAdapter);

    Output.nextSteps([
      'Review your config in orgs/$orgId/projects/$projectId/',
      'Run `applad config validate` to check your config',
      'Run `applad up` to start the server',
    ]);
  }

  void _createDir(String path) {
    Directory(path).createSync(recursive: true);
    Output.step(0, 'Created $path');
  }

  void _writeFile(String path, String content) {
    File(path).writeAsStringSync(content);
    Output.step(0, 'Wrote $path');
  }

  String _toId(String name) {
    return name
        .toLowerCase()
        .replaceAll(RegExp(r'[^a-z0-9]+'), '-')
        .replaceAll(RegExp(r'^-+|-+$'), '');
  }

  String _instanceConfig() => '''
# Applad Instance Configuration
# This file defines the root instance settings.
version: "1"

# AI assistant configuration (optional)
# ai:
#   provider: anthropic
#   model: claude-opus-4-6
#   api_key: \${ANTHROPIC_API_KEY}

# Features enabled at the instance level
enabled_features:
  - auth
  - database
  - storage
''';

  String _orgConfig(String id, String name) => '''
# Organization Configuration
id: $id
name: $name

# Team members
members:
  - email: admin@example.com
    role: admin

# Infrastructure targets (where to deploy)
# infrastructure_targets:
#   - fly
#   - aws
''';

  String _projectConfig(String id, String name, String orgId) => '''
# Project Configuration
id: $id
name: $name
org_id: $orgId

# Default environment to use when not specified
default_environment: development

# Per-environment configuration
environments:
  development:
    variables:
      APP_ENV: development
      LOG_LEVEL: debug
  staging:
    variables:
      APP_ENV: staging
      LOG_LEVEL: info
  production:
    variables:
      APP_ENV: production
      LOG_LEVEL: warning
''';

  String _authConfig() => '''
# Authentication Configuration
# Configure who can sign up and how they authenticate.

# Auth providers
providers:
  - type: email
  # - type: google
  #   client_id: \${GOOGLE_CLIENT_ID}
  # - type: github
  #   client_id: \${GITHUB_CLIENT_ID}

# Session duration (in seconds). Default: 24 hours.
session_duration_seconds: 86400

# Multi-factor authentication
mfa:
  required: false
  methods:
    - totp

# Role-based access control
rbac:
  default_role: user
''';

  String _databaseConfig(String adapter) => '''
# Database Configuration
adapter: $adapter

${adapter == 'sqlite' ? '# SQLite database file location (relative to project root)\n# database: .applad/data.db' : '''
# Connection settings
# host: localhost
# port: 5432
# database: \${DB_NAME}
# username: \${DB_USER}
# password: \${DB_PASSWORD}
'''}

# Connection pool settings
pool:
  min: 1
  max: 10

# Migration settings
migrations:
  directory: migrations
  auto_run: false
''';

  String _messagingConfig() => '''
# Messaging Configuration
# Provider config per channel.
email:
  enabled: true
  provider: resend
  config:
    api_key: \${RESEND_API_KEY}
    from: "hello@example.com"
''';
}
