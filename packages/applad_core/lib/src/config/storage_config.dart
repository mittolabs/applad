library;

/// Storage configuration (`storage/*.yaml`).
final class StorageConfig {
  const StorageConfig({
    required this.adapter,
    this.buckets = const [],
    this.maxFileSizeMb = 50,
  });

  factory StorageConfig.fromMap(Map<String, dynamic> map) {
    return StorageConfig(
      adapter: map['adapter'] as String? ?? 'local',
      buckets: (map['buckets'] as List?)
              ?.map((b) => BucketConfig.fromMap(b as Map<String, dynamic>))
              .toList() ??
          [],
      maxFileSizeMb: map['max_file_size_mb'] as int? ?? 50,
    );
  }

  final String adapter; // local, s3, r2, gcs, azure
  final List<BucketConfig> buckets;
  final int maxFileSizeMb;

  Map<String, dynamic> toJson() => {
        'adapter': adapter,
        'buckets': buckets.map((b) => b.toJson()).toList(),
        'max_file_size_mb': maxFileSizeMb,
      };
}

final class BucketConfig {
  const BucketConfig({
    required this.name,
    this.public = false,
    this.allowedMimeTypes = const [],
    this.maxFileSizeMb,
  });

  factory BucketConfig.fromMap(Map<String, dynamic> map) {
    return BucketConfig(
      name: map['name'] as String,
      public: map['public'] as bool? ?? false,
      allowedMimeTypes:
          (map['allowed_mime_types'] as List?)?.cast<String>() ?? [],
      maxFileSizeMb: map['max_file_size_mb'] as int?,
    );
  }

  final String name;
  final bool public;
  final List<String> allowedMimeTypes;
  final int? maxFileSizeMb;

  Map<String, dynamic> toJson() => {
        'name': name,
        'public': public,
        'allowed_mime_types': allowedMimeTypes,
        'max_file_size_mb': maxFileSizeMb,
      };
}
