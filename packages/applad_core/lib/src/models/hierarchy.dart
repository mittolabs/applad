library;

/// Represents the three-level Applad hierarchy:
/// Instance → Org → Project
final class ApplAdInstance {
  const ApplAdInstance({
    required this.version,
    required this.orgs,
  });

  final String version;
  final List<ApplAdOrg> orgs;
}

final class ApplAdOrg {
  const ApplAdOrg({
    required this.id,
    required this.name,
    required this.projects,
  });

  final String id;
  final String name;
  final List<ApplAdProject> projects;
}

final class ApplAdProject {
  const ApplAdProject({
    required this.id,
    required this.name,
    required this.orgId,
  });

  final String id;
  final String name;
  final String orgId;
}
