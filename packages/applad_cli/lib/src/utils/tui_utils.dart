import 'dart:io';

/// Terminal UI utilities for coordinate-based drawing and ANSI formatting.
final class TuiUtils {
  TuiUtils._();

  static final bool _supportsAnsi = stdout.supportsAnsiEscapes;

  /// Move cursor to [x], [y] (1-indexed).
  static void moveTo(int x, int y) {
    if (!_supportsAnsi) return;
    stdout.write('\x1b[$y;${x}H');
  }

  /// Clear the entire screen.
  static void clear() {
    if (!_supportsAnsi) return;
    stdout.write('$bgBase\x1b[2J\x1b[3J\x1b[H$reset');

    // Explicitly flood the screen with absolute black empty strings
    // to combat macOS terminal transparent tearing.
    final width = stdout.terminalColumns;
    final height = stdout.terminalLines;
    for (var i = 1; i <= height; i++) {
      moveTo(1, i);
      stdout.write(bgBase + (' ' * width) + reset);
    }
    moveTo(1, 1);
  }

  /// Draw a box with a [title] and [borderColor].
  static void drawBox(int x, int y, int width, int height,
      {String? title, String? borderColor}) {
    if (!_supportsAnsi) return;

    final color = borderColor ?? borderNormal;
    final bottom = '╰${'─' * (width - 2)}╯';
    final side = '│';

    moveTo(x, y);
    if (title != null) {
      final rightDashesCount =
          width - title.length - 4; // 2 for corners, 2 for spaces
      final rightDashes = rightDashesCount > 0 ? '─' * rightDashesCount : '';
      stdout.write('$color╭ $textPrimary$title$color $rightDashes╮$reset');
    } else {
      stdout.write('$color╭${'─' * (width - 2)}╮$reset');
    }

    for (var i = 1; i < height - 1; i++) {
      moveTo(x, y + i);
      stdout.write('$color$side$reset');
      moveTo(x + width - 1, y + i);
      stdout.write('$color$side$reset');
    }

    moveTo(x, y + height - 1);
    stdout.write('$color$bottom$reset');
  }

  /// Draw a horizontal line.
  static void drawLine(int x, int y, int width, {String? color}) {
    if (!_supportsAnsi) return;
    final c = color ?? borderNormal;
    moveTo(x, y);
    stdout.write('$c${'─' * width}$reset');
  }

  /// Print text at a specific coordinate.
  static void printAt(int x, int y, String text,
      {bool bold = false, String? color, int? maxWidth}) {
    moveTo(x, y);
    var result = text;
    if (maxWidth != null && text.length > maxWidth) {
      result = '${text.substring(0, maxWidth - 3)}...';
    }
    if (bold) {
      result = '\x1b[1m$result\x1b[22m';
    }
    if (color != null) {
      result =
          '$color$result\x1b[39m\x1b[40m'; // Reset to default FG, force Black BG
    }
    stdout.write(result);
  }

  static const String reset =
      '\x1b[0m\x1b[40m'; // Reset formatting, force Black BG

  // Posting aesthetic Standard 16-Color approximations
  static const String textPrimary = '\x1b[37m'; // White
  static const String textMuted = '\x1b[90m'; // Slate

  static const String accentCyan = '\x1b[36m'; // Neon Cyan
  static const String accentMagenta = '\x1b[35m'; // Magenta
  static const String accentGreen = '\x1b[32m'; // Emerald

  static const String bgBase = '\x1b[40m'; // Black Background
  static const String bgSurface = '\x1b[40m'; // Black Background
  static const String bgActiveTab = '\x1b[46m'; // Cyan Background
  static const String bgMagenta = '\x1b[45m';

  static const String borderNormal = '\x1b[90m'; // Slate border
  static const String borderActive = '\x1b[35m'; // Active border (Magenta)

  // Legacy mappings for existing code
  static const String cyan = accentCyan;
  static const String magenta = accentMagenta;
  static const String green = accentGreen;
  static const String yellow = '\x1b[33m';
  static const String blue = '\x1b[34m';
  static const String black = '\x1b[30m';
  static const String white = textPrimary;
  static const String bgBlue = bgSurface;
  static const String bgBlack = bgBase;

  static const String enableMouse =
      '\x1b[?1000h\x1b[?1006h'; // Basic buttons + SGR
  static const String disableMouse = '\x1b[?1000l\x1b[?1006l';

  static const String enterAltScreen = '\x1b[?1049h';
  static const String exitAltScreen = '\x1b[?1049l';

  /// Parse an SGR mouse event escape sequence (e.g., \x1b[<0;20;10M).
  /// Returns a [TuiMouseEvent] or null if not a mouse event.
  static TuiMouseEvent? parseSgrMouse(String sequence) {
    if (!sequence.startsWith('\x1b[<')) return null;
    final match =
        RegExp(r'\x1b\[<(\d+);(\d+);(\d+)([Mm])').firstMatch(sequence);
    if (match == null) return null;

    final button = int.parse(match.group(1)!);
    final x = int.parse(match.group(2)!);
    final y = int.parse(match.group(3)!);
    final isDown = match.group(4) == 'M';

    return TuiMouseEvent(x, y, button: button, isDown: isDown);
  }
}

class TuiMouseEvent {
  final int x;
  final int y;
  final int button;
  final bool isDown;

  TuiMouseEvent(this.x, this.y, {required this.button, required this.isDown});

  @override
  String toString() => 'Mouse(x: $x, y: $y, btn: $button, down: $isDown)';
}
