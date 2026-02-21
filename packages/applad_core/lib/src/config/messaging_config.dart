library;

/// Messaging configuration (`messaging/messaging.yaml`).
final class MessagingConfig {
  const MessagingConfig({
    this.email,
    this.sms,
    this.push,
    this.inApp,
    this.integrations = const [],
  });

  factory MessagingConfig.fromMap(Map<String, dynamic> map) {
    return MessagingConfig(
      email: map['email'] != null
          ? EmailConfig.fromMap(Map<String, dynamic>.from(map['email'] as Map))
          : null,
      sms: map['sms'] != null
          ? SmsConfig.fromMap(Map<String, dynamic>.from(map['sms'] as Map))
          : null,
      push: map['push'] != null
          ? PushConfig.fromMap(Map<String, dynamic>.from(map['push'] as Map))
          : null,
      inApp: map['in_app'] != null
          ? InAppConfig.fromMap(Map<String, dynamic>.from(map['in_app'] as Map))
          : null,
      integrations: (map['integrations'] as List?)
              ?.map((i) => IntegrationConfig.fromMap(
                  Map<String, dynamic>.from(i as Map)))
              .toList() ??
          [],
    );
  }

  final EmailConfig? email;
  final SmsConfig? sms;
  final PushConfig? push;
  final InAppConfig? inApp;
  final List<IntegrationConfig> integrations;

  Map<String, dynamic> toJson() => {
        'email': email?.toJson(),
        'sms': sms?.toJson(),
        'push': push?.toJson(),
        'in_app': inApp?.toJson(),
        'integrations': integrations.map((i) => i.toJson()).toList(),
      };
}

final class EmailConfig {
  const EmailConfig({
    this.enabled = false,
    this.provider,
    this.config,
    this.environmentOverrides,
  });

  factory EmailConfig.fromMap(Map<String, dynamic> map) {
    return EmailConfig(
      enabled: map['enabled'] as bool? ?? false,
      provider: map['provider']?.toString(),
      config: map['config'] != null
          ? Map<String, dynamic>.from(map['config'] as Map)
          : null,
      environmentOverrides: map['environment_overrides'] != null
          ? Map<String, dynamic>.from(map['environment_overrides'] as Map)
          : null,
    );
  }

  final bool enabled;
  final String? provider;
  final Map<String, dynamic>? config;
  final Map<String, dynamic>? environmentOverrides;

  Map<String, dynamic> toJson() => {
        'enabled': enabled,
        'provider': provider,
        'config': config,
        'environment_overrides': environmentOverrides,
      };
}

final class SmsConfig {
  const SmsConfig({
    this.enabled = false,
    this.provider,
    this.config,
  });

  factory SmsConfig.fromMap(Map<String, dynamic> map) {
    return SmsConfig(
      enabled: map['enabled'] as bool? ?? false,
      provider: map['provider']?.toString(),
      config: map['config'] != null
          ? Map<String, dynamic>.from(map['config'] as Map)
          : null,
    );
  }

  final bool enabled;
  final String? provider;
  final Map<String, dynamic>? config;

  Map<String, dynamic> toJson() => {
        'enabled': enabled,
        'provider': provider,
        'config': config,
      };
}

final class PushConfig {
  const PushConfig({
    this.enabled = false,
    this.providers = const [],
  });

  factory PushConfig.fromMap(Map<String, dynamic> map) {
    return PushConfig(
      enabled: map['enabled'] as bool? ?? false,
      providers: (map['providers'] as List?)
              ?.map((p) => PushProviderConfig.fromMap(
                  Map<String, dynamic>.from(p as Map)))
              .toList() ??
          [],
    );
  }

  final bool enabled;
  final List<PushProviderConfig> providers;

  Map<String, dynamic> toJson() => {
        'enabled': enabled,
        'providers': providers.map((p) => p.toJson()).toList(),
      };
}

final class PushProviderConfig {
  const PushProviderConfig({
    required this.platform,
    required this.provider,
    this.config,
  });

  factory PushProviderConfig.fromMap(Map<String, dynamic> map) {
    return PushProviderConfig(
      platform: map['platform']?.toString() ?? '',
      provider: map['provider']?.toString() ?? '',
      config: map['config'] != null
          ? Map<String, dynamic>.from(map['config'] as Map)
          : null,
    );
  }

  final String platform;
  final String provider;
  final Map<String, dynamic>? config;

  Map<String, dynamic> toJson() => {
        'platform': platform,
        'provider': provider,
        'config': config,
      };
}

final class InAppConfig {
  const InAppConfig({
    this.enabled = false,
    this.persistence = true,
    this.maxAge,
    this.realtimeDelivery = true,
  });

  factory InAppConfig.fromMap(Map<String, dynamic> map) {
    return InAppConfig(
      enabled: map['enabled'] as bool? ?? false,
      persistence: map['persistence'] as bool? ?? true,
      maxAge: map['max_age']?.toString(),
      realtimeDelivery: map['realtime_delivery'] as bool? ?? true,
    );
  }

  final bool enabled;
  final bool persistence;
  final String? maxAge;
  final bool realtimeDelivery;

  Map<String, dynamic> toJson() => {
        'enabled': enabled,
        'persistence': persistence,
        'max_age': maxAge,
        'realtime_delivery': realtimeDelivery,
      };
}

final class IntegrationConfig {
  const IntegrationConfig({
    required this.name,
    this.enabled = false,
    this.webhookUrl,
    this.defaultChannel,
  });

  factory IntegrationConfig.fromMap(Map<String, dynamic> map) {
    return IntegrationConfig(
      name: map['name']?.toString() ?? '',
      enabled: map['enabled'] as bool? ?? false,
      webhookUrl: map['webhook_url']?.toString(),
      defaultChannel: map['default_channel']?.toString(),
    );
  }

  final String name;
  final bool enabled;
  final String? webhookUrl;
  final String? defaultChannel;

  Map<String, dynamic> toJson() => {
        'name': name,
        'enabled': enabled,
        'webhook_url': webhookUrl,
        'default_channel': defaultChannel,
      };
}
