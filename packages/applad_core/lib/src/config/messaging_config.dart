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
          ? EmailConfig.fromMap(map['email'] as Map<String, dynamic>)
          : null,
      sms: map['sms'] != null
          ? SmsConfig.fromMap(map['sms'] as Map<String, dynamic>)
          : null,
      push: map['push'] != null
          ? PushConfig.fromMap(map['push'] as Map<String, dynamic>)
          : null,
      inApp: map['in_app'] != null
          ? InAppConfig.fromMap(map['in_app'] as Map<String, dynamic>)
          : null,
      integrations: (map['integrations'] as List?)
              ?.map((i) => IntegrationConfig.fromMap(i as Map<String, dynamic>))
              .toList() ??
          [],
    );
  }

  final EmailConfig? email;
  final SmsConfig? sms;
  final PushConfig? push;
  final InAppConfig? inApp;
  final List<IntegrationConfig> integrations;
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
      provider: map['provider'] as String?,
      config: map['config'] as Map<String, dynamic>?,
      environmentOverrides:
          map['environment_overrides'] as Map<String, dynamic>?,
    );
  }

  final bool enabled;
  final String? provider;
  final Map<String, dynamic>? config;
  final Map<String, dynamic>? environmentOverrides;
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
      provider: map['provider'] as String?,
      config: map['config'] as Map<String, dynamic>?,
    );
  }

  final bool enabled;
  final String? provider;
  final Map<String, dynamic>? config;
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
              ?.map(
                  (p) => PushProviderConfig.fromMap(p as Map<String, dynamic>))
              .toList() ??
          [],
    );
  }

  final bool enabled;
  final List<PushProviderConfig> providers;
}

final class PushProviderConfig {
  const PushProviderConfig({
    required this.platform,
    required this.provider,
    this.config,
  });

  factory PushProviderConfig.fromMap(Map<String, dynamic> map) {
    return PushProviderConfig(
      platform: map['platform'] as String,
      provider: map['provider'] as String,
      config: map['config'] as Map<String, dynamic>?,
    );
  }

  final String platform;
  final String provider;
  final Map<String, dynamic>? config;
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
      maxAge: map['max_age'] as String?,
      realtimeDelivery: map['realtime_delivery'] as bool? ?? true,
    );
  }

  final bool enabled;
  final bool persistence;
  final String? maxAge;
  final bool realtimeDelivery;
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
      name: map['name'] as String,
      enabled: map['enabled'] as bool? ?? false,
      webhookUrl: map['webhook_url'] as String?,
      defaultChannel: map['default_channel'] as String?,
    );
  }

  final String name;
  final bool enabled;
  final String? webhookUrl;
  final String? defaultChannel;
}
