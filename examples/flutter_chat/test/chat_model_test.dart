import 'package:applad_chat/chat_model.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('InviteCode', () {
    test('round-trips through encode/parse', () {
      final code = const InviteCode('team1', 'mem1', 'sekret').encode();
      expect(code, 'team1:mem1:sekret');
      final parsed = InviteCode.tryParse(code)!;
      expect(parsed.teamId, 'team1');
      expect(parsed.membershipId, 'mem1');
      expect(parsed.secret, 'sekret');
    });

    test('tolerates surrounding whitespace', () {
      final parsed = InviteCode.tryParse('  a:b:c \n')!;
      expect(parsed.teamId, 'a');
      expect(parsed.secret, 'c');
    });

    test('keeps colons that belong to the secret', () {
      final parsed = InviteCode.tryParse('team:mem:aa:bb:cc')!;
      expect(parsed.teamId, 'team');
      expect(parsed.membershipId, 'mem');
      expect(parsed.secret, 'aa:bb:cc');
    });

    test('rejects malformed codes', () {
      expect(InviteCode.tryParse('nope'), isNull);
      expect(InviteCode.tryParse('a:b'), isNull);
      expect(InviteCode.tryParse('a::c'), isNull);
      expect(InviteCode.tryParse(':b:c'), isNull);
      expect(InviteCode.tryParse(''), isNull);
    });
  });

  group('messagePermissions', () {
    test('members read; only the author writes', () {
      final perms = messagePermissions('chan1', 'userA');
      expect(perms, contains('read("team:chan1")'));
      expect(perms, contains('update("user:userA")'));
      expect(perms, contains('delete("user:userA")'));
      // No blanket team write: a member cannot edit someone else's message.
      expect(perms.any((p) => p.startsWith('update("team:')), isFalse);
    });
  });
}
