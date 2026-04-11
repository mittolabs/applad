import 'package:flutter/material.dart';
import '../theme/console_colors.dart';

// ── Token types ───────────────────────────────────────────────────────────────

enum _TT { keyword, string, comment, typeName, method, number, punctuation, plain }

class _Token {
  final String text;
  final _TT type;
  const _Token(this.text, this.type);
}

// ── Per-language token rules (priority order) ─────────────────────────────────

typedef _Rule = (RegExp, _TT);

List<_Rule> _rulesFor(String language) {
  final lang = language.toLowerCase();

  // Single-line comment prefix
  final commentPat = lang == 'python'
      ? RegExp(r'#[^\n]*')
      : RegExp(r'//[^\n]*');

  // String literals — single-quoted raw strings concatenated to avoid delimiter issues
  const sq = r"'(?:[^'\\]|\\.)*'";
  const dq = r'"(?:[^"\\]|\\.)*"';
  const bt = r'`(?:[^`\\]|\\.)*`';
  final stringPat = lang == 'dart'
      ? RegExp('$sq|$dq')
      : RegExp('$sq|$dq|$bt');

  // Keywords per language
  final RegExp kwPat;
  if (lang == 'python') {
    kwPat = RegExp(r'\b(import|from|class|def|return|await|async|if|elif|else|for|while|in|not|and|or|True|False|None|with|as|try|except|raise|pass|break|continue)\b');
  } else if (lang == 'javascript' || lang == 'node.js' || lang == 'typescript') {
    kwPat = RegExp(r'\b(import|export|from|const|let|var|function|class|return|await|async|if|else|for|while|new|this|typeof|instanceof|throw|try|catch|finally|of|in|true|false|null|undefined|void)\b');
  } else {
    // Dart
    kwPat = RegExp(r'\b(import|export|final|const|var|class|abstract|extends|implements|mixin|return|await|async|if|else|for|while|new|this|super|void|late|required|static|get|set|factory|enum|typedef|try|catch|finally|throw|null|true|false)\b');
  }

  // Type/class names: PascalCase identifiers
  final typePat = RegExp(r'\b[A-Z][A-Za-z0-9_]*\b');

  // Method/property calls: .identifier
  final methodPat = RegExp(r'(?<=\.)[a-z_][A-Za-z0-9_]*(?=\()');

  // Numbers
  final numPat = RegExp(r'\b\d+(\.\d+)?\b');

  // Punctuation
  final punctPat = RegExp(r'[{}()\[\];,.<>]');

  return [
    (commentPat, _TT.comment),
    (stringPat,  _TT.string),
    (kwPat,      _TT.keyword),
    (typePat,    _TT.typeName),
    (methodPat,  _TT.method),
    (numPat,     _TT.number),
    (punctPat,   _TT.punctuation),
  ];
}

// ── Tokenizer ─────────────────────────────────────────────────────────────────

List<_Token> _tokenize(String code, String language) {
  final rules = _rulesFor(language);
  final tokens = <_Token>[];
  var pos = 0;

  while (pos < code.length) {
    bool matched = false;
    for (final (pattern, type) in rules) {
      final m = pattern.matchAsPrefix(code, pos);
      if (m != null && m.group(0)!.isNotEmpty) {
        tokens.add(_Token(m.group(0)!, type));
        pos += m.group(0)!.length;
        matched = true;
        break;
      }
    }
    if (!matched) {
      // Accumulate plain chars
      final start = pos;
      while (pos < code.length) {
        bool anyMatch = false;
        for (final (p, _) in rules) {
          if (p.matchAsPrefix(code, pos) != null) {
            anyMatch = true;
            break;
          }
        }
        if (anyMatch) break;
        pos++;
      }
      tokens.add(_Token(code.substring(start, pos), _TT.plain));
    }
  }
  return tokens;
}

// ── Color palette (dark / light) ─────────────────────────────────────────────

Color _tokenColor(_TT type, bool isLight) {
  if (isLight) {
    return switch (type) {
      _TT.keyword     => const Color(0xFF7C3AED), // violet-600
      _TT.string      => const Color(0xFF15803D), // green-700
      _TT.comment     => const Color(0xFF64748B), // slate-500
      _TT.typeName    => const Color(0xFFB45309), // amber-700
      _TT.method      => const Color(0xFF1D4ED8), // blue-700
      _TT.number      => const Color(0xFFD97706), // amber-600
      _TT.punctuation => const Color(0xFF0369A1), // sky-700
      _TT.plain       => const Color(0xFF1E293B), // slate-800
    };
  }
  return switch (type) {
    _TT.keyword     => const Color(0xFFC792EA), // violet
    _TT.string      => const Color(0xFFA5D6A7), // soft green
    _TT.comment     => const Color(0xFF546E7A), // blue-gray
    _TT.typeName    => const Color(0xFFFFCB6B), // amber
    _TT.method      => const Color(0xFF82AAFF), // sky blue
    _TT.number      => const Color(0xFFF78C6C), // orange
    _TT.punctuation => const Color(0xFF89DDFF), // cyan
    _TT.plain       => const Color(0xFFE0E0E0), // near-white
  };
}

// ── Public widget ─────────────────────────────────────────────────────────────

/// Syntax-highlighted read-only code block.
///
/// ```dart
/// CodeBlock(code: mySnippet, language: 'dart')
/// ```
///
/// Supported languages: `dart`, `javascript`, `node.js`, `typescript`, `python`.
class CodeBlock extends StatelessWidget {
  final String code;

  /// Language hint for tokenization. Case-insensitive.
  final String language;

  /// Font size (default 11.5).
  final double fontSize;

  const CodeBlock({
    super.key,
    required this.code,
    required this.language,
    this.fontSize = 11.5,
  });

  @override
  Widget build(BuildContext context) {
    final isLight = consoleIsLight(context);
    final tokens = _tokenize(code, language);

    final spans = tokens.map((t) => TextSpan(
          text: t.text,
          style: TextStyle(
            color: _tokenColor(t.type, isLight),
            fontSize: fontSize,
            fontFamily: 'monospace',
            height: 1.7,
          ),
        )).toList();

    return SelectableText.rich(TextSpan(children: spans));
  }
}
