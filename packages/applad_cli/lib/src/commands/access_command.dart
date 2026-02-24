import 'package:args/command_runner.dart';
import '../utils/output.dart';

/// Manage access grants, roles, and permissions.
final class AccessCommand extends Command<void> {
  AccessCommand() {
    addSubcommand(AccessListCommand());
    addSubcommand(AccessGrantCommand());
    addSubcommand(AccessRevokeCommand());
    addSubcommand(AccessApproveCommand());
    addSubcommand(AccessRejectCommand());
    addSubcommand(AccessRequestsCommand());
    addSubcommand(AccessShowCommand());
  }

  @override
  String get name => 'access';

  @override
  String get description => 'Manage access grants, roles, and permissions.';
}

/// Lists all access grants.
final class AccessListCommand extends Command<void> {
  AccessListCommand() {
    argParser.addOption('org', help: 'Scope to a specific organization.');
    argParser.addOption('project', help: 'Scope to a specific project.');
  }

  @override
  String get name => 'list';

  @override
  String get description => 'Lists all access grants for an org or project.';

  @override
  Future<void> run() async {
    final org = argResults!['org'] as String?;
    final project = argResults!['project'] as String?;

    Output.header(
        'Access Grants${org != null ? " ($org)" : ""}${project != null ? " [$project]" : ""}');
    Output.table([
      'Identity',
      'Role',
      'Scopes',
      'Expiry'
    ], [
      ['alice@acme-corp.com', 'owner', '*', 'Never'],
      ['bob@acme-corp.com', 'developer', 'infra:apply:staging', '2026-12-31'],
      ['ci@acme-corp.com', 'ci', 'deploy:*, infra:apply:*', 'Never'],
    ]);
  }
}

/// Grants a scope or role.
final class AccessGrantCommand extends Command<void> {
  AccessGrantCommand() {
    argParser.addOption('scope', help: 'Grant specific scope(s).');
    argParser.addOption('role', help: 'Grant a specific role.');
    argParser.addOption('org', help: 'Organization context.');
    argParser.addOption('project', help: 'Project context.');
    argParser.addOption('expires', help: 'Expiry date (YYYY-MM-DD).');
  }

  @override
  String get name => 'grant';

  @override
  String get description => 'Grants a scope or role to an identity.';

  @override
  Future<void> run() async {
    if (argResults!.rest.isEmpty) {
      Output.error(
          'Usage: applad access grant <IDENTITY> [--scope <S>] [--role <R>]');
      return;
    }
    final identity = argResults!.rest.first;
    final scope = argResults!['scope'] as String?;
    final role = argResults!['role'] as String?;
    final expires = argResults!['expires'] as String?;

    Output.info('Granting ${role ?? scope} to "$identity"...');
    if (expires != null) Output.info('  - Access expires on $expires');

    Output.success('Grant applied to admin database.');
  }
}

/// Revokes a scope or role.
final class AccessRevokeCommand extends Command<void> {
  AccessRevokeCommand() {
    argParser.addOption('scope', help: 'Revoke specific scope(s).');
    argParser.addOption('org', help: 'Organization context.');
    argParser.addFlag('all',
        help: 'Revoke ALL grants for this identity.', negatable: false);
  }

  @override
  String get name => 'revoke';

  @override
  String get description => 'Revokes a scope or role from an identity.';

  @override
  Future<void> run() async {
    if (argResults!.rest.isEmpty) {
      Output.error(
          'Usage: applad access revoke <IDENTITY> [--scope <S>] [--all]');
      return;
    }
    final identity = argResults!.rest.first;
    final scope = argResults!['scope'] as String?;
    final all = argResults!['all'] as bool;

    if (all) {
      Output.warning('REVOKING ALL ACCESS FOR: $identity');
      if (Output.confirm('Are you sure?')) {
        Output.success('All grants revoked for "$identity".');
      }
    } else {
      Output.info('Revoking $scope from "$identity"...');
      Output.success('Scope revoked.');
    }
  }
}

/// Approves an access request.
final class AccessApproveCommand extends Command<void> {
  AccessApproveCommand() {
    argParser.addOption('role',
        help: 'Role to grant (defaults to developer).',
        defaultsTo: 'developer');
    argParser.addOption('org', help: 'Organization context.');
  }

  @override
  String get name => 'approve';

  @override
  String get description => 'Approves a pending access request.';

  @override
  Future<void> run() async {
    if (argResults!.rest.isEmpty) {
      Output.error('Usage: applad access approve <EMAIL>');
      return;
    }
    final email = argResults!.rest.first;
    final role = argResults!['role'] as String;

    Output.info('Approving request from "$email" with role "$role"...');
    Output.success('Access approved. Developer notified.');
  }
}

/// Rejects an access request.
final class AccessRejectCommand extends Command<void> {
  @override
  String get name => 'reject';

  @override
  String get description => 'Rejects a pending access request.';

  @override
  Future<void> run() async {
    if (argResults!.rest.isEmpty) {
      Output.error('Usage: applad access reject <EMAIL>');
      return;
    }
    final email = argResults!.rest.first;
    Output.info('Rejecting request from "$email"...');
    Output.success('Request rejected.');
  }
}

/// Manage pending requests.
final class AccessRequestsCommand extends Command<void> {
  AccessRequestsCommand() {
    addSubcommand(AccessRequestsListCommand());
  }

  @override
  String get name => 'requests';

  @override
  String get description => 'Manage pending access requests.';
}

final class AccessRequestsListCommand extends Command<void> {
  @override
  String get name => 'list';

  @override
  String get description => 'Lists all pending access requests.';

  @override
  Future<void> run() async {
    Output.header('Pending Access Requests');
    Output.table([
      'Email',
      'Key Fingerprint',
      'Requested At'
    ], [
      ['bob@acme-corp.com', 'SHA256:def456...', '2026-02-23 21:50'],
      ['charlie@acme-corp.com', 'SHA256:ghi789...', '2026-02-23 21:55'],
    ]);
  }
}

/// Shows effective permissions.
final class AccessShowCommand extends Command<void> {
  @override
  String get name => 'show';

  @override
  String get description => 'Shows effective permissions for an identity.';

  @override
  Future<void> run() async {
    if (argResults!.rest.isEmpty) {
      Output.error('Usage: applad access show <IDENTITY>');
      return;
    }
    final identity = argResults!.rest.first;
    Output.header('Effective Permissions: $identity');
    Output.info('Org Role: developer (Acme Corp)');
    Output.info('SSH Key Scope: config:*, functions:*, infra:apply:staging');
    Output.blank();
    Output.kv('Final Decision',
        'Can apply to STAGING, but NOT PRODUCTION (Key scope bottleneck)');
  }
}
