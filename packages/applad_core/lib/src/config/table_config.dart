library;

/// Table schema configuration (`tables/*.yaml`).
final class TableConfig {
  const TableConfig({
    required this.name,
    required this.fields,
    this.database,
    this.indexes = const [],
    this.permissions = const [],
    this.timestamps = true,
    this.softDelete = false,
  });

  factory TableConfig.fromMap(Map<String, dynamic> map) {
    return TableConfig(
      name: map['name']?.toString() ?? '',
      fields: (map['fields'] as List?)
              ?.map((f) =>
                  FieldConfig.fromMap(Map<String, dynamic>.from(f as Map)))
              .toList() ??
          [],
      database: map['database']?.toString(),
      indexes: (map['indexes'] as List?)
              ?.map((i) =>
                  IndexConfig.fromMap(Map<String, dynamic>.from(i as Map)))
              .toList() ??
          [],
      permissions: (map['permissions'] as List?)
              ?.map((p) => TablePermissionRule.fromMap(
                  Map<String, dynamic>.from(p as Map)))
              .toList() ??
          [],
      timestamps: map['timestamps'] as bool? ?? true,
      softDelete: map['soft_delete'] as bool? ?? false,
    );
  }

  final String name;
  final List<FieldConfig> fields;
  final String? database;
  final List<IndexConfig> indexes;
  final List<TablePermissionRule> permissions;
  final bool timestamps;
  final bool softDelete;

  Map<String, dynamic> toJson() => {
        'name': name,
        'fields': fields.map((f) => f.toJson()).toList(),
        if (database != null) 'database': database,
        'indexes': indexes.map((i) => i.toJson()).toList(),
        'permissions': permissions.map((p) => p.toJson()).toList(),
        'timestamps': timestamps,
        'soft_delete': softDelete,
      };
}

final class FieldConfig {
  const FieldConfig({
    required this.name,
    required this.type,
    this.requiredLevel = false,
    this.unique = false,
    this.indexed = false,
    this.defaultValue,
    this.references,
  });

  factory FieldConfig.fromMap(Map<String, dynamic> map) {
    return FieldConfig(
      name: map['name']?.toString() ?? '',
      type: map['type']?.toString() ?? '',
      requiredLevel: map['required'] as bool? ?? false,
      unique: map['unique'] as bool? ?? false,
      indexed: map['indexed'] as bool? ?? false,
      defaultValue: map['default'],
      references: map['table']?.toString() ??
          map['references']?.toString(), // Handle relation targets
    );
  }

  final String name;
  final String type; // string, integer, boolean, relation, enum, etc.
  final bool requiredLevel;
  final bool unique;
  final bool indexed;
  final dynamic defaultValue;
  final String? references; // e.g. "organizations"

  Map<String, dynamic> toJson() => {
        'name': name,
        'type': type,
        'required': requiredLevel,
        'unique': unique,
        'indexed': indexed,
        'default': defaultValue,
        'references': references,
      };
}

final class IndexConfig {
  const IndexConfig({required this.fields, this.unique = false, this.name});

  factory IndexConfig.fromMap(Map<String, dynamic> map) {
    return IndexConfig(
      fields: (map['fields'] as List?)?.map((e) => e.toString()).toList() ?? [],
      unique: map['unique'] as bool? ?? false,
      name: map['name']?.toString(),
    );
  }

  final List<String> fields;
  final bool unique;
  final String? name;

  Map<String, dynamic> toJson() => {
        'fields': fields,
        'unique': unique,
        'name': name,
      };
}

final class TablePermissionRule {
  const TablePermissionRule({
    required this.role,
    required this.actions,
    this.filter,
  });

  factory TablePermissionRule.fromMap(Map<String, dynamic> map) {
    return TablePermissionRule(
      role: map['role']?.toString() ?? '',
      actions:
          (map['actions'] as List?)?.map((e) => e.toString()).toList() ?? [],
      filter: map['filter']?.toString(),
    );
  }

  final String role; // e.g. "owner", "admin", "user", "*"
  final List<String> actions; // e.g. ["read", "write"] or ["*"]
  final String? filter; // e.g. "id == $user.id"

  Map<String, dynamic> toJson() => {
        'role': role,
        'actions': actions,
        'filter': filter,
      };
}
