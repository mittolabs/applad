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
              ?.map((p) =>
                  AuthProvider.fromMap(Map<String, dynamic>.from(p as Map)))
              .toList() ??
          [],
      mfa: map['mfa'] != null
          ? MfaConfig.fromMap(Map<String, dynamic>.from(map['mfa'] as Map))
          : null,
      sso: map['sso'] != null
          ? SsoConfig.fromMap(Map<String, dynamic>.from(map['sso'] as Map))
          : null,
      sessionDurationSeconds: (map['session_duration_seconds'] ??
              (map['session'] is Map
                  ? (map['session'] as Map)['duration']
                  : null)) as int? ??
          86400,
      rbac: map['rbac'] != null
          ? RbacConfig.fromMap(Map<String, dynamic>.from(map['rbac'] as Map))
          : null,
    );
  }

  final List<AuthProvider> providers;
  final MfaConfig? mfa;
  final SsoConfig? sso;
  final int sessionDurationSeconds;
  final RbacConfig? rbac;

  Map<String, dynamic> toJson() => {
        'providers': providers.map((p) => p.toJson()).toList(),
        'mfa': mfa?.toJson(),
        'sso': sso?.toJson(),
        'session_duration_seconds': sessionDurationSeconds,
        'rbac': rbac?.toJson(),
      };
}

final class AuthProvider {
  const AuthProvider({required this.type, this.clientId, this.enabled = true});

  factory AuthProvider.fromMap(Map<String, dynamic> map) {
    return AuthProvider(
      type: (map['type'] ?? map['name'])?.toString() ?? 'email',
      clientId: map['client_id']?.toString(),
      enabled: map['enabled'] as bool? ?? true,
    );
  }

  final String type; // email, google, github, apple, etc.
  final String? clientId;
  final bool enabled;

  Map<String, dynamic> toJson() => {
        'type': type,
        'client_id': clientId,
        'enabled': enabled,
      };
}

final class MfaConfig {
  const MfaConfig({this.required = false, this.methods = const ['totp']});

  factory MfaConfig.fromMap(Map<String, dynamic> map) {
    return MfaConfig(
      required: map['required'] as bool? ?? false,
      methods: (map['methods'] as List?)?.map((m) {
            if (m is Map) return m['type']?.toString() ?? 'totp';
            return m.toString();
          }).toList() ??
          ['totp'],
    );
  }

  final bool required;
  final List<String> methods;

  Map<String, dynamic> toJson() => {
        'required': required,
        'methods': methods,
      };
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

  Map<String, dynamic> toJson() => {
        'provider': provider,
        'domain': domain,
        'required': required,
      };
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

  Map<String, dynamic> toJson() => {
        'roles_file': rolesFile,
        'default_role': defaultRole,
      };
}
