library;

/// Security configuration — rate limits, CORS, CSP, IP allowlists.
final class SecurityConfig {
  const SecurityConfig({
    this.rateLimiting,
    this.cors,
    this.csp,
    this.ipAllowlist = const [],
    this.ipBlocklist = const [],
  });

  factory SecurityConfig.fromMap(Map<String, dynamic> map) {
    return SecurityConfig(
      rateLimiting: map['rate_limiting'] != null
          ? RateLimitConfig.fromMap(
              map['rate_limiting'] as Map<String, dynamic>)
          : null,
      cors: map['cors'] != null
          ? CorsConfig.fromMap(map['cors'] as Map<String, dynamic>)
          : null,
      csp: map['csp'] as String?,
      ipAllowlist: (map['ip_allowlist'] as List?)?.cast<String>() ?? [],
      ipBlocklist: (map['ip_blocklist'] as List?)?.cast<String>() ?? [],
    );
  }

  final RateLimitConfig? rateLimiting;
  final CorsConfig? cors;
  final String? csp;
  final List<String> ipAllowlist;
  final List<String> ipBlocklist;

  Map<String, dynamic> toJson() => {
        'rate_limiting': rateLimiting?.toJson(),
        'cors': cors?.toJson(),
        'csp': csp,
        'ip_allowlist': ipAllowlist,
        'ip_blocklist': ipBlocklist,
      };
}

final class RateLimitConfig {
  const RateLimitConfig({
    this.requestsPerMinute = 60,
    this.burstSize = 20,
    this.byIp = true,
    this.byUser = false,
  });

  factory RateLimitConfig.fromMap(Map<String, dynamic> map) {
    return RateLimitConfig(
      requestsPerMinute: map['requests_per_minute'] as int? ?? 60,
      burstSize: map['burst_size'] as int? ?? 20,
      byIp: map['by_ip'] as bool? ?? true,
      byUser: map['by_user'] as bool? ?? false,
    );
  }

  final int requestsPerMinute;
  final int burstSize;
  final bool byIp;
  final bool byUser;

  Map<String, dynamic> toJson() => {
        'requests_per_minute': requestsPerMinute,
        'burst_size': burstSize,
        'by_ip': byIp,
        'by_user': byUser,
      };
}

final class CorsConfig {
  const CorsConfig({
    this.allowedOrigins = const ['*'],
    this.allowedMethods = const ['GET', 'POST', 'PUT', 'DELETE', 'OPTIONS'],
    this.allowedHeaders = const ['*'],
    this.allowCredentials = false,
    this.maxAgeSecs = 86400,
  });

  factory CorsConfig.fromMap(Map<String, dynamic> map) {
    return CorsConfig(
      allowedOrigins:
          (map['allowed_origins'] as List?)?.cast<String>() ?? ['*'],
      allowedMethods: (map['allowed_methods'] as List?)?.cast<String>() ??
          ['GET', 'POST', 'PUT', 'DELETE', 'OPTIONS'],
      allowedHeaders:
          (map['allowed_headers'] as List?)?.cast<String>() ?? ['*'],
      allowCredentials: map['allow_credentials'] as bool? ?? false,
      maxAgeSecs: map['max_age_secs'] as int? ?? 86400,
    );
  }

  final List<String> allowedOrigins;
  final List<String> allowedMethods;
  final List<String> allowedHeaders;
  final bool allowCredentials;
  final int maxAgeSecs;

  Map<String, dynamic> toJson() => {
        'allowed_origins': allowedOrigins,
        'allowed_methods': allowedMethods,
        'allowed_headers': allowedHeaders,
        'allow_credentials': allowCredentials,
        'max_age_secs': maxAgeSecs,
      };
}
