library;

/// Static hosting configuration (`hosting/*.yaml`).
final class HostingConfig {
  const HostingConfig({
    required this.name,
    this.domain,
    this.source,
    this.buildCommand,
    this.outputDirectory = 'build/web',
    this.headers = const [],
    this.redirects = const [],
  });

  factory HostingConfig.fromMap(Map<String, dynamic> map) {
    return HostingConfig(
      name: map['name'] as String,
      domain: map['domain'] as String?,
      source: map['source'] != null
          ? GitSource.fromMap(map['source'] as Map<String, dynamic>)
          : null,
      buildCommand: map['build_command'] as String?,
      outputDirectory: map['output_directory'] as String? ?? 'build/web',
      headers: (map['headers'] as List?)
              ?.map((h) => HostingHeader.fromMap(h as Map<String, dynamic>))
              .toList() ??
          [],
      redirects: (map['redirects'] as List?)
              ?.map((r) => HostingRedirect.fromMap(r as Map<String, dynamic>))
              .toList() ??
          [],
    );
  }

  final String name;
  final String? domain;
  final GitSource? source;
  final String? buildCommand;
  final String outputDirectory;
  final List<HostingHeader> headers;
  final List<HostingRedirect> redirects;
}

final class GitSource {
  const GitSource({required this.repo, this.branch = 'main'});

  factory GitSource.fromMap(Map<String, dynamic> map) {
    return GitSource(
      repo: map['repo'] as String,
      branch: map['branch'] as String? ?? 'main',
    );
  }

  final String repo;
  final String branch;
}

final class HostingHeader {
  const HostingHeader({required this.path, required this.headers});

  factory HostingHeader.fromMap(Map<String, dynamic> map) {
    return HostingHeader(
      path: map['path'] as String,
      headers: (map['headers'] as Map).cast<String, String>(),
    );
  }

  final String path;
  final Map<String, String> headers;
}

final class HostingRedirect {
  const HostingRedirect(
      {required this.from, required this.to, this.statusCode = 301});

  factory HostingRedirect.fromMap(Map<String, dynamic> map) {
    return HostingRedirect(
      from: map['from'] as String,
      to: map['to'] as String,
      statusCode: map['status'] as int? ?? 301,
    );
  }

  final String from;
  final String to;
  final int statusCode;
}
