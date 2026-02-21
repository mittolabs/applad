import 'package:applad_core/applad_core.dart';
import 'package:dart_frog/dart_frog.dart';

Handler middleware(Handler handler) {
  // Inject the ConfigLoader into the request context.
  return handler.use(
    provider<ConfigLoader>((_) => const ConfigLoader()),
  );
}
