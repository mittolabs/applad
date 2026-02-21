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
    final Map<String, dynamic>? sourceMap =
        map['source'] is Map ? Map<String, dynamic>.from(map['source']) : null;

    return HostingConfig(
      name: map['name']?.toString() ?? '',
      domain: map['domain']?.toString(),
      source: sourceMap != null ? GitSource.fromMap(sourceMap) : null,
      buildCommand: map['build_command']?.toString() ??
          sourceMap?['build_command']?.toString(),
      outputDirectory: map['output_directory']?.toString() ??
          map['output_dir']?.toString() ??
          sourceMap?['output_dir']?.toString() ??
          'build/web',
      headers: (map['headers'] as List?)
              ?.map((h) =>
                  HostingHeader.fromMap(Map<String, dynamic>.from(h as Map)))
              .toList() ??
          [],
      redirects: (map['redirects'] as List?)
              ?.map((r) =>
                  HostingRedirect.fromMap(Map<String, dynamic>.from(r as Map)))
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
      repo: map['repo']?.toString() ?? '',
      branch: map['branch']?.toString() ?? 'main',
    );
  }

  final String repo;
  final String branch;
}

final class HostingHeader {
  const HostingHeader({required this.path, required this.headers});

  factory HostingHeader.fromMap(Map<String, dynamic> map) {
    final rawHeaders = map['headers'] ?? map['values'];
    final headersMap = <String, String>{};
    if (rawHeaders is Map) {
      for (final entry in rawHeaders.entries) {
        headersMap[entry.key.toString()] = entry.value?.toString() ?? '';
      }
    }

    return HostingHeader(
      path: map['path']?.toString() ?? '',
      headers: headersMap,
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
      from: map['from']?.toString() ?? '',
      to: map['to']?.toString() ?? '',
      statusCode: (map['status'] ?? map['status_code']) as int? ?? 301,
    );
  }

  final String from;
  final String to;
  final int statusCode;
}
