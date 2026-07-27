// A smoke test that the app boots. With no APPLAD_PROJECT defined (the default
// in a plain `flutter test` run), the app shows its configuration notice rather
// than touching the network, which is exactly what we assert.
import 'package:flutter_test/flutter_test.dart';

import 'package:applad_chat/main.dart';

void main() {
  testWidgets('shows the configuration notice when no project is set',
      (WidgetTester tester) async {
    await tester.pumpWidget(const ChatApp());
    await tester.pump();

    expect(find.text('No project configured'), findsOneWidget);
  });
}
