import 'package:args/command_runner.dart';
import '../utils/output.dart';

/// Authenticates to the Applad instance.
final class LoginCommand extends Command<void> {
  @override
  String get name => 'login';

  @override
  String get description =>
      'Authenticates to the instance described in applad.yaml.';

  @override
  Future<void> run() async {
    Output.info('Looking for instance in applad.yaml...');
    Output.info('Found: https://api.myapp.com (acme-corp)');

    final email = Output.prompt('Your email');
    final keyPath = Output.prompt('SSH public key path',
        defaultValue: '~/.ssh/id_ed25519.pub');

    Output.info('Registering SSH key ($keyPath) for $email with instance...');
    Output.success('Access request sent — SHA256:def456...');
    Output.blank();
    Output.info('Waiting for an administrator to approve your request.');
    Output.info(
        'Once approved, run "applad up" to start your local environment.');
  }
}

/// Revokes the local session.
final class LogoutCommand extends Command<void> {
  @override
  String get name => 'logout';

  @override
  String get description => 'Revokes the local session.';

  @override
  Future<void> run() async {
    Output.info('Logging out...');
    Output.success('Local session revoked.');
  }
}

/// Shows the currently authenticated identity.
final class WhoamiCommand extends Command<void> {
  @override
  String get name => 'whoami';

  @override
  String get description => 'Shows the currently authenticated identity.';

  @override
  Future<void> run() async {
    Output.header('Identity');
    Output.kv('Email', 'bob@acme-corp.com');
    Output.kv('Role', 'developer');
    Output.kv('Org', 'acme-corp');
    Output.kv('Key Fingerprint', 'SHA256:def456...');
    Output.kv('Effective Scopes', 'config:*, functions:*, infra:apply:staging');
  }
}
