import 'dart:io';

/// Terminal output helpers with ANSI colors.
final class Output {
  Output._();

  static bool _supportsColor = stdout.supportsAnsiEscapes;

  static void setColorSupport(bool value) => _supportsColor = value;

  // ANSI color codes
  static String _red(String s) => _supportsColor ? '\x1B[31m$s\x1B[0m' : s;
  static String _green(String s) => _supportsColor ? '\x1B[32m$s\x1B[0m' : s;
  static String _yellow(String s) => _supportsColor ? '\x1B[33m$s\x1B[0m' : s;
  static String _cyan(String s) => _supportsColor ? '\x1B[36m$s\x1B[0m' : s;
  static String _bold(String s) => _supportsColor ? '\x1B[1m$s\x1B[0m' : s;
  static String _dim(String s) => _supportsColor ? '\x1B[2m$s\x1B[0m' : s;

  /// Print a success message with a green checkmark.
  static void success(String message) {
    stdout.writeln('${_green('✓')} $message');
  }

  /// Print an error message with a red X.
  static void error(String message) {
    stderr.writeln('${_red('✗')} $message');
  }

  /// Print a warning message with a yellow warning symbol.
  static void warning(String message) {
    stdout.writeln('${_yellow('⚠')} $message');
  }

  /// Print an info message with a cyan bullet.
  static void info(String message) {
    stdout.writeln('${_cyan('›')} $message');
  }

  /// Print a step with a numbered prefix.
  static void step(int number, String message) {
    stdout.writeln('${_dim('$number.')} $message');
  }

  /// Print a section header.
  static void header(String title) {
    stdout.writeln();
    stdout.writeln(_bold(title));
    stdout.writeln(_dim('─' * title.length));
  }

  /// Print a key-value pair.
  static void kv(String key, String value) {
    stdout.writeln('  ${_dim('$key:')} $value');
  }

  /// Print a blank line.
  static void blank() => stdout.writeln();

  /// Print a table with header row and data rows.
  static void table(List<String> headers, List<List<String>> rows) {
    if (rows.isEmpty) {
      stdout.writeln(_dim('(no results)'));
      return;
    }

    // Calculate column widths
    final widths = List.generate(headers.length, (i) {
      final maxRow = rows
          .map((r) => i < r.length ? r[i].length : 0)
          .fold(0, (a, b) => a > b ? a : b);
      return [headers[i].length, maxRow].reduce((a, b) => a > b ? a : b);
    });

    // Print header
    final headerRow = headers.asMap().entries.map((e) {
      return _bold(e.value.padRight(widths[e.key]));
    }).join('  ');
    stdout.writeln(headerRow);
    stdout.writeln(_dim(widths.map((w) => '─' * w).join('  ')));

    // Print rows
    for (final row in rows) {
      final line = row.asMap().entries.map((e) {
        return e.key < widths.length
            ? e.value.padRight(widths[e.key])
            : e.value;
      }).join('  ');
      stdout.writeln(line);
    }
  }

  /// Prompt the user for input.
  static String prompt(String question, {String? defaultValue}) {
    final hint = defaultValue != null ? ' [${_dim(defaultValue)}]' : '';
    stdout.write('${_cyan('?')} $question$hint: ');
    final input = stdin.readLineSync() ?? '';
    return input.isEmpty && defaultValue != null ? defaultValue : input;
  }

  /// Prompt the user for sensitive input (echo disabled).
  static String secretPrompt(String question) {
    stdout.write('${_cyan('?')} $question: ');
    final oldEchoMode = stdin.echoMode;
    try {
      stdin.echoMode = false;
      final input = stdin.readLineSync() ?? '';
      stdout.writeln(); // Move to next line after input
      return input;
    } finally {
      stdin.echoMode = oldEchoMode;
    }
  }

  /// Prompt the user for a yes/no answer.
  static bool confirm(String question, {bool defaultValue = false}) {
    final hint = defaultValue ? '[Y/n]' : '[y/N]';
    stdout.write('${_cyan('?')} $question $hint: ');
    final input = (stdin.readLineSync() ?? '').trim().toLowerCase();
    if (input.isEmpty) return defaultValue;
    return input == 'y' || input == 'yes';
  }

  /// Prompt the user for a yes/no answer with a descriptive text.
  static bool confirmWithDescription(String question, String description,
      {bool defaultValue = false}) {
    stdout.writeln('  ${_dim('└─ $description')}');
    final hint = defaultValue ? '[Y/n]' : '[y/N]';
    stdout.write('  ${_cyan('?')} $question $hint: ');
    final input = (stdin.readLineSync() ?? '').trim().toLowerCase();
    if (input.isEmpty) return defaultValue;
    return input == 'y' || input == 'yes';
  }

  /// Print "next steps" section.
  static void nextSteps(List<String> steps) {
    blank();
    stdout.writeln(_bold('Next steps:'));
    for (var i = 0; i < steps.length; i++) {
      stdout.writeln('  ${_dim('${i + 1}.')} ${steps[i]}');
    }
  }
}
