import 'package:args/command_runner.dart';
import '../utils/output.dart';

/// `applad access` — Manage access grants, scopes, and requests in the admin database.
/// These commands act entirely on the admin database, completely disconnected from
/// the application runtime project setup.
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
  String get description =>
      'Manage fine-grained access levels, scopes, and pending requests for Applad instances.';
}

final class AccessListCommand extends Command<void> {
  AccessListCommand() {
    argParser.addOption('org', help: 'Filter by organization context.');
    argParser.addOption('project', help: 'Filter by project context.');
  }
  @override
  String get name => 'list';
  @override
  String get description =>
      'List all access grants across the instance, optionally filtered by org or project.';
  @override
  Future<void> run() async {
    Output.info('Listing access grants (mock)...');
    Output.success('No explicit grants defined currently.');
  }
}

final class AccessGrantCommand extends Command<void> {
  AccessGrantCommand() {
    argParser.addOption('org', help: 'Target org.');
    argParser.addOption('project', help: 'Target project.');
    argParser.addOption('env', help: 'Target environment.');
    argParser.addMultiOption('scope',
        abbr: 's', help: 'Scope to grant (e.g. access:manage).');
  }
  @override
  String get name => 'grant';
  @override
  String get description =>
      'Grant specific scopes to an identity or role on an environment, project, or instance level.';
  @override
  Future<void> run() async {
    final identity =
        argResults!.rest.isNotEmpty ? argResults!.rest.first : 'User';
    final scopes = argResults?['scope'] as List<String>? ?? [];
    Output.success('Granted ${scopes.join(', ')} to $identity (mock).');
  }
}

final class AccessRevokeCommand extends Command<void> {
  AccessRevokeCommand() {
    argParser.addOption('org', help: 'Target org.');
    argParser.addOption('project', help: 'Target project.');
    argParser.addOption('env', help: 'Target environment.');
    argParser.addFlag('all',
        help: 'Revoke all access from the specific namespace.');
  }
  @override
  String get name => 'revoke';
  @override
  String get description =>
      'Revoke specific scopes or all access from an identity.';
  @override
  Future<void> run() async {
    final identity =
        argResults!.rest.isNotEmpty ? argResults!.rest.first : 'User';
    Output.success('Revoked access for $identity (mock).');
  }
}

final class AccessApproveCommand extends Command<void> {
  @override
  String get name => 'approve';
  @override
  String get description =>
      'Approves a pending access request from applad login.';
  @override
  Future<void> run() async {
    final identity =
        argResults!.rest.isNotEmpty ? argResults!.rest.first : 'User';
    Output.success('Approved access request for $identity (mock).');
  }
}

final class AccessRejectCommand extends Command<void> {
  @override
  String get name => 'reject';
  @override
  String get description => 'Rejects a pending access request.';
  @override
  Future<void> run() async {
    final identity =
        argResults!.rest.isNotEmpty ? argResults!.rest.first : 'User';
    Output.success('Rejected access request for $identity (mock).');
  }
}

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
  String get description =>
      'Lists all pending access requests across the instance.';
  @override
  Future<void> run() async {
    Output.info('Listing pending access requests (mock)...');
    Output.success('No pending requests found.');
  }
}

final class AccessShowCommand extends Command<void> {
  @override
  String get name => 'show';
  @override
  String get description =>
      'Shows detailed active access grants and attributes bound to a specific identity.';
  @override
  Future<void> run() async {
    final identity =
        argResults!.rest.isNotEmpty ? argResults!.rest.first : 'User';
    Output.info('Showing grants for $identity (mock)...');
  }
}
