/// Applad Core — shared config models, merge engine, and domain types.
library;

// Config models
export 'src/config/instance_config.dart';
export 'src/config/org_config.dart';
export 'src/config/project_config.dart';
export 'src/config/auth_config.dart';
export 'src/config/database_config.dart';
export 'src/config/table_config.dart';
export 'src/config/storage_config.dart';
export 'src/config/function_config.dart';
export 'src/config/workflow_config.dart';
export 'src/config/messaging_config.dart';
export 'src/config/flag_config.dart';

export 'src/config/deployment_config.dart';
export 'src/config/realtime_config.dart';
export 'src/config/analytics_config.dart';
export 'src/config/observability_config.dart';
export 'src/config/security_config.dart';

// Merge engine
export 'src/merge/config_loader.dart';
export 'src/merge/config_merger.dart';
export 'src/merge/config_validator.dart';

// Domain models
export 'src/models/audit_entry.dart';
export 'src/models/ssh_identity.dart';
export 'src/models/hierarchy.dart';
export 'src/models/environment.dart';
export 'src/models/secret_ref.dart';

// Utils
export 'src/utils/env_parser.dart';
export 'src/utils/vvar_extractor.dart';

// Errors
export 'src/errors/applad_error.dart';
