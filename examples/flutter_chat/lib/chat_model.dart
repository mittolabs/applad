/// Pure, testable helpers for the chat example — the pieces worth verifying
/// on their own, independent of any widget. Keeping them here (rather than
/// inline in a State class) is what lets `test/` exercise them.
library;

/// What a joiner needs to accept an invite: which team, which membership, and
/// the secret that proves the invite belongs to them. They travel as one
/// colon-joined code because a self-hosted instance often has no email to send
/// a link, so the inviter copies the code and hands it over directly.
class InviteCode {
  final String teamId;
  final String membershipId;
  final String secret;

  const InviteCode(this.teamId, this.membershipId, this.secret);

  String encode() => '$teamId:$membershipId:$secret';

  /// Parse a pasted code, or null if it is not well formed. A secret could in
  /// principle contain a colon, so everything after the second colon is the
  /// secret rather than splitting into exactly three.
  static InviteCode? tryParse(String raw) {
    final parts = raw.trim().split(':');
    if (parts.length < 3) return null;
    final teamId = parts[0];
    final membershipId = parts[1];
    final secret = parts.sublist(2).join(':');
    if (teamId.isEmpty || membershipId.isEmpty || secret.isEmpty) return null;
    return InviteCode(teamId, membershipId, secret);
  }
}

/// The permissions a new message carries: every member of the channel's team
/// may read it; only its author may edit or delete it. This is the shape the
/// platform's document-level security enforces, so it lives in one place both
/// the app and its tests agree on.
List<String> messagePermissions(String channelId, String authorUserId) => [
      'read("team:$channelId")',
      'update("user:$authorUserId")',
      'delete("user:$authorUserId")',
    ];
