library;

/// Authentication configuration (`auth/auth.yaml`).
final class AuthConfig {
  const AuthConfig({
    this.providers = const [],
    this.mfa,
    this.sso,
    this.sessionDurationSeconds = 86400,
    this.rbac,
  });

  factory AuthConfig.fromMap(Map<String, dynamic> map) {
    return AuthConfig(
      providers: (map['providers'] as List?)
              ?.map((p) => AuthProvider.fromMap(p as Map<String, dynamic>))
              .toList() ??
          [],
      mfa: map['mfa'] != null
          ? MfaConfig.fromMap(map['mfa'] as Map<String, dynamic>)
          : null,
      sso: map['sso'] != null
          ? SsoConfig.fromMap(map['sso'] as Map<String, dynamic>)
          : null,
      sessionDurationSeconds: map['session_duration_seconds'] as int? ?? 86400,
      rbac: map['rbac'] != null
          ? RbacConfig.fromMap(map['rbac'] as Map<String, dynamic>)
          : null,
    );
  }

  final List<AuthProvider> providers;
  final MfaConfig? mfa;
  final SsoConfig? sso;
  final int sessionDurationSeconds;
  final RbacConfig? rbac;
}

final class AuthProvider {
  const AuthProvider({required this.type, this.clientId, this.enabled = true});

  factory AuthProvider.fromMap(Map<String, dynamic> map) {
    return AuthProvider(
      type: map['type'] as String,
      clientId: map['client_id'] as String?,
      enabled: map['enabled'] as bool? ?? true,
    );
  }

  final String type; // email, google, github, apple, etc.
  final String? clientId;
  final bool enabled;
}

final class MfaConfig {
  const MfaConfig({this.required = false, this.methods = const ['totp']});

  factory MfaConfig.fromMap(Map<String, dynamic> map) {
    return MfaConfig(
      required: map['required'] as bool? ?? false,
      methods: (map['methods'] as List?)?.cast<String>() ?? ['totp'],
    );
  }

  final bool required;
  final List<String> methods;
}

final class SsoConfig {
  const SsoConfig({required this.provider, this.domain, this.required = false});

  factory SsoConfig.fromMap(Map<String, dynamic> map) {
    return SsoConfig(
      provider: map['provider'] as String,
      domain: map['domain'] as String?,
      required: map['required'] as bool? ?? false,
    );
  }

  final String provider;
  final String? domain;
  final bool required;
}

final class RbacConfig {
  const RbacConfig({this.rolesFile, this.defaultRole = 'user'});

  factory RbacConfig.fromMap(Map<String, dynamic> map) {
    return RbacConfig(
      rolesFile: map['roles_file'] as String?,
      defaultRole: map['default_role'] as String? ?? 'user',
    );
  }

  final String? rolesFile;
  final String defaultRole;
}
