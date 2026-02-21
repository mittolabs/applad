library;

/// Organization-level configuration (`org.yaml`).
final class OrgConfig {
  const OrgConfig({
    required this.id,
    required this.name,
    this.members = const [],
    this.infrastructureTargets = const [],
  });

  factory OrgConfig.fromMap(Map<String, dynamic> map) {
    return OrgConfig(
      id: map['id'] as String,
      name: map['name'] as String,
      members: (map['members'] as List?)
              ?.map((m) => OrgMember.fromMap(m as Map<String, dynamic>))
              .toList() ??
          [],
      infrastructureTargets:
          (map['infrastructure_targets'] as List?)?.cast<String>() ?? [],
    );
  }

  final String id;
  final String name;
  final List<OrgMember> members;
  final List<String> infrastructureTargets;

  Map<String, dynamic> toJson() => {
        'id': id,
        'name': name,
        'members': members.map((m) => m.toJson()).toList(),
        'infrastructure_targets': infrastructureTargets,
      };
}

final class OrgMember {
  const OrgMember({required this.email, required this.role});

  factory OrgMember.fromMap(Map<String, dynamic> map) {
    return OrgMember(
      email: map['email'] as String,
      role: map['role'] as String? ?? 'member',
    );
  }

  final String email;
  final String role;

  Map<String, dynamic> toJson() => {
        'email': email,
        'role': role,
      };
}
