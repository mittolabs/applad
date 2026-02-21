library;

/// Table schema configuration (`tables/*.yaml`).
final class TableConfig {
  const TableConfig({
    required this.name,
    required this.fields,
    this.indexes = const [],
    this.permissions,
    this.timestamps = true,
    this.softDelete = false,
  });

  factory TableConfig.fromMap(Map<String, dynamic> map) {
    return TableConfig(
      name: map['name'] as String,
      fields: (map['fields'] as List)
          .map((f) => FieldConfig.fromMap(f as Map<String, dynamic>))
          .toList(),
      indexes: (map['indexes'] as List?)
              ?.map((i) => IndexConfig.fromMap(i as Map<String, dynamic>))
              .toList() ??
          [],
      permissions: map['permissions'] != null
          ? TablePermissions.fromMap(map['permissions'] as Map<String, dynamic>)
          : null,
      timestamps: map['timestamps'] as bool? ?? true,
      softDelete: map['soft_delete'] as bool? ?? false,
    );
  }

  final String name;
  final List<FieldConfig> fields;
  final List<IndexConfig> indexes;
  final TablePermissions? permissions;
  final bool timestamps;
  final bool softDelete;
}

final class FieldConfig {
  const FieldConfig({
    required this.name,
    required this.type,
    this.nullable = false,
    this.unique = false,
    this.defaultValue,
    this.references,
  });

  factory FieldConfig.fromMap(Map<String, dynamic> map) {
    return FieldConfig(
      name: map['name'] as String,
      type: map['type'] as String,
      nullable: map['nullable'] as bool? ?? false,
      unique: map['unique'] as bool? ?? false,
      defaultValue: map['default'],
      references: map['references'] as String?,
    );
  }

  final String name;
  final String type; // text, integer, boolean, uuid, timestamp, jsonb, etc.
  final bool nullable;
  final bool unique;
  final dynamic defaultValue;
  final String? references; // e.g. "users.id"
}

final class IndexConfig {
  const IndexConfig({required this.fields, this.unique = false, this.name});

  factory IndexConfig.fromMap(Map<String, dynamic> map) {
    return IndexConfig(
      fields: (map['fields'] as List).cast<String>(),
      unique: map['unique'] as bool? ?? false,
      name: map['name'] as String?,
    );
  }

  final List<String> fields;
  final bool unique;
  final String? name;
}

final class TablePermissions {
  const TablePermissions({
    this.select,
    this.insert,
    this.update,
    this.delete,
  });

  factory TablePermissions.fromMap(Map<String, dynamic> map) {
    return TablePermissions(
      select: map['select'] as String?,
      insert: map['insert'] as String?,
      update: map['update'] as String?,
      delete: map['delete'] as String?,
    );
  }

  /// Permission rule expression (e.g. "auth.uid = user_id", "role = admin").
  final String? select;
  final String? insert;
  final String? update;
  final String? delete;
}
