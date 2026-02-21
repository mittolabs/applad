library;

/// The available deployment environments.
enum Environment {
  development('development', 'dev'),
  staging('staging', 'staging'),
  production('production', 'prod');

  const Environment(this.name, this.shortName);

  final String name;
  final String shortName;

  static Environment fromString(String value) {
    return switch (value.toLowerCase()) {
      'development' || 'dev' || 'local' => Environment.development,
      'staging' => Environment.staging,
      'production' || 'prod' => Environment.production,
      _ => throw ArgumentError('Unknown environment: $value'),
    };
  }

  bool get isProduction => this == Environment.production;
  bool get isDevelopment => this == Environment.development;

  @override
  String toString() => name;
}
