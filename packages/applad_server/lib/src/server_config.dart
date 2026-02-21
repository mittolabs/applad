library;

import 'dart:io';
import 'package:applad_core/applad_core.dart';

/// Loads and caches the Applad config at server startup.
final class ServerConfig {
  ServerConfig._();

  static ApplAdConfig? _instance;

  /// Load config from the given root path (or environment variable).
  static ApplAdConfig load({String? rootPath}) {
    if (_instance != null) return _instance!;

    final path = rootPath ??
        Platform.environment['APPLAD_CONFIG_PATH'] ??
        Directory.current.path;

    final merger = ConfigMerger();
    try {
      _instance = merger.merge(path);
      return _instance!;
    } catch (e) {
      // In Phase 1, we allow the server to start without full config
      // and return a minimal default config.
      _instance = _minimalConfig(path);
      return _instance!;
    }
  }

  static ApplAdConfig _minimalConfig(String rootPath) {
    return ApplAdConfig(
      instance: InstanceConfig.fromMap({'version': '1'}),
      org: OrgConfig.fromMap({'id': 'default', 'name': 'Default Org'}),
      project: ProjectConfig.fromMap({
        'id': 'default',
        'name': 'Default Project',
        'org_id': 'default',
      }),
      rootPath: rootPath,
    );
  }

  /// Returns the loaded config. Throws if not yet loaded.
  static ApplAdConfig get instance {
    if (_instance == null) throw StateError('ServerConfig not initialized. Call load() first.');
    return _instance!;
  }

  static void reset() => _instance = null;
}
