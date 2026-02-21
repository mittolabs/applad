import 'package:applad_core/applad_core.dart';
import 'package:applad_server/src/server_config.dart';
import 'package:dart_frog/dart_frog.dart';

Handler middleware(Handler handler) {
  // Inject the fully merged config into the request context.
  final config = ServerConfig.load();
  return handler.use(
    provider<ApplAdConfig>((_) => config),
  );
}
