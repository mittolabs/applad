library;

/// Represents a source location for functions or deployments.
final class SourceBlock {
  const SourceBlock({
    required this.type,
    this.path,
    this.repo,
    this.branch,
    this.sshKey,
    this.image,
    this.credentials,
  });

  factory SourceBlock.fromMap(Map<String, dynamic> map) {
    return SourceBlock(
      type: map['type'] as String,
      path: map['path'] as String?,
      repo: map['repo'] as String?,
      branch: map['branch'] as String?,
      sshKey: map['ssh_key'] as String?,
      image: map['image'] as String?,
      credentials: map['credentials'] as String?,
    );
  }

  final String type; // local, github, registry
  final String? path;
  final String? repo;
  final String? branch;
  final String? sshKey;
  final String? image;
  final String? credentials;

  Map<String, dynamic> toJson() => {
        'type': type,
        'path': path,
        'repo': repo,
        'branch': branch,
        'ssh_key': sshKey,
        'image': image,
        'credentials': credentials,
      };

  @override
  String toString() => 'SourceBlock(type: $type, path: $path, repo: $repo)';
}
