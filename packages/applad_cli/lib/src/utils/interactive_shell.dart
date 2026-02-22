import 'dart:io';
import 'package:applad_core/applad_core.dart';
import 'output.dart';
import '../commands/applad_command_runner.dart';
import '../security/trust_manager.dart';
import '../utils/config_finder.dart';
import '../utils/tui_utils.dart';

enum ShellMode { repl, tui }

/// An interactive REPL and TUI Dashboard for the Applad CLI.
final class InteractiveShell {
  final ApplAdCommandRunner runner;
  ShellMode _mode = ShellMode.repl;
  int _activeTab = 0; // 0: Dashboard, 1: Projects, 2: Resources
  List<String> _currentTabs = [' DASHBOARD ', ' PROJECTS '];

  InteractiveShell(this.runner);

  /// Starts the interactive shell.
  Future<void> start() async {
    // Ensure mouse tracking is disabled on startup
    stdout.write(TuiUtils.disableMouse);
    _clearScreen();
    _printWelcome();

    // Security layer: Ensure workspace is trusted
    TrustManager.ensureTrusted();
    Output.blank();

    try {
      while (true) {
        final context = _resolveContext();

        if (_mode == ShellMode.tui) {
          stdout.write(TuiUtils.enableMouse);
          await _renderTui(context);

          stdin.lineMode = false;
          stdin.echoMode = false;

          try {
            final bytes = <int>[];
            while (true) {
              final byte = stdin.readByteSync();
              if (byte == -1) break;
              bytes.add(byte);

              // If it's a typical escape sequence, keep reading until complete
              if (bytes.length == 1 && bytes[0] == 27) continue;
              if (bytes.length > 1 && bytes[0] == 27) {
                // Simple check for end of escape sequence (M or m for mouse, or just a character)
                final str = String.fromCharCodes(bytes);
                if (str.endsWith('M') ||
                    str.endsWith('m') ||
                    bytes.length > 10) {
                  break;
                }
                continue;
              }
              break;
            }

            final input = String.fromCharCodes(bytes);
            if (input == 'q' || input == 'exit' || input == 'quit') break;
            if (input == 'r' || input == 'repl' || input == '\x1b') {
              _mode = ShellMode.repl;
              stdout.write(TuiUtils.disableMouse);
              _clearScreen();
              _printWelcome();
              continue;
            }

            final mouseEvent = TuiUtils.parseSgrMouse(input);
            if (mouseEvent != null) {
              _handleTuiMouse(mouseEvent);
            } else {
              await _handleTuiInput(input);
            }
          } finally {
            stdin.lineMode = true;
            stdin.echoMode = true;
          }
          continue;
        }

        _printContextBar(context);

        final promptPrefix = _getPromptPrefix();
        stdout.write('$promptPrefix ');
        final input = stdin.readLineSync()?.trim() ?? '';

        if (input.isEmpty) continue;

        if (input == 'exit' || input == 'quit') {
          Output.info('Goodbye!');
          break;
        }

        if (input == 'clear' || input == 'cls') {
          _clearScreen();
          _printWelcome();
          continue;
        }

        if (input == 'help' || input == '?') {
          runner.printUsage();
          continue;
        }

        if (input == 'dashboard' || input == 'ui') {
          final hasOrg = context['org'] != '-' && context['org'] != 'error';
          if (!hasOrg) {
            Output.blank();
            Output.info('Dashboard requires an active project.');
            final choice = Output.prompt(
                'Would you like to (1) Select a project or (2) Create a new one?',
                defaultValue: '1');
            if (choice == '1') {
              final root = ConfigFinder.discoverProjectRoot();
              if (root != null) {
                Directory.current = root.path;
                _mode = ShellMode.tui;
              }
            } else if (choice == '2') {
              try {
                await runner.run(['init']);
                final root = ConfigFinder.discoverProjectRoot();
                if (root != null) {
                  Directory.current = root.path;
                  _mode = ShellMode.tui;
                }
              } catch (e) {
                Output.error('Failed to init project: $e');
              }
            }
          } else {
            _mode = ShellMode.tui;
          }
          continue;
        }

        try {
          final args = _parseInput(input);
          if (args.isEmpty) continue;

          final commandName = args.first;

          // Handle 'cd' internally to update the shell's process environment
          if (commandName == 'cd') {
            _handleCd(args);
            Output.blank();
            continue;
          }

          // Check if it's an applad command
          final isAppladCommand = runner.commands.containsKey(commandName);

          if (isAppladCommand) {
            await runner.run(args);
          } else {
            // Pass through to system shell
            await _runSystemCommand(args);
          }
        } catch (e) {
          Output.error('Error executing command: $e');
        }

        Output.blank();
      }
    } finally {
      // Always disable mouse on exit
      stdout.write(TuiUtils.disableMouse);
    }
  }

  void _handleCd(List<String> args) {
    if (args.length < 2) {
      final home = Platform.environment['HOME'] ?? '';
      if (home.isNotEmpty) Directory.current = home;
      return;
    }

    final target = args[1];
    final dir = Directory(target);
    if (dir.existsSync()) {
      Directory.current = dir;
    } else {
      Output.error('cd: no such file or directory: $target');
    }
  }

  Future<void> _runSystemCommand(List<String> args) async {
    try {
      final process = await Process.start(
        args.first,
        args.sublist(1),
        mode: ProcessStartMode.inheritStdio,
      );
      final exitCode = await process.exitCode;
      if (exitCode != 0 && args.first == 'ls' && Platform.isMacOS) {
        // Subtle hint for common mistakes
      }
    } catch (e) {
      stdout.writeln(
          '\x1b[31m✗\x1b[0m Could not find a command named "${args.first}".');
      stdout.writeln('\x1b[2mTry "help" for a list of Applad commands.\x1b[0m');
    }
  }

  void _clearScreen() {
    if (Platform.isWindows) {
      stdout.write('\x1B[2J\x1B[0;0H');
    } else {
      stdout.write('\x1B[2J\x1B[3J\x1B[H');
    }
  }

  void _printWelcome() {
    stdout.writeln(runner.getLogo());
    stdout.writeln('\x1b[1m* Welcome back to your Applad Environment!\x1b[0m');
    stdout.writeln(
        '\x1b[2mType a command below to manage your backend. Use "help" for more info.\x1b[0m');
    stdout.writeln('\x1b[2m  1. Run "init" to scaffold a project.\x1b[0m');
    stdout.writeln('\x1b[2m  2. Use "up" to apply changes.\x1b[0m');
    stdout
        .writeln('\x1b[2m  3. Type "dashboard" to enter the Admin TUI.\x1b[0m');
    stdout.writeln(
        '\x1b[2m  4. Type "exit" to leave the interactive shell.\x1b[0m');
    stdout.writeln();
  }

  Map<String, dynamic> _resolveContext() {
    final finder = const ConfigFinder();
    final root = finder.findRoot();
    if (root == null) {
      return {
        'org': '-',
        'project': '-',
        'env': 'local',
        'features': <String>[]
      };
    }

    try {
      final merger = ConfigMerger();
      final config = merger.merge(root);
      return {
        'org': config.project.orgId,
        'project': config.project.id,
        'env': 'local', // Default in shell for now
        'features': config.project.enabledFeatures,
      };
    } catch (_) {
      return {
        'org': 'error',
        'project': 'error',
        'env': 'local',
        'features': <String>[]
      };
    }
  }

  void _printContextBar(Map<String, dynamic> context) {
    final org = context['org']!;
    final project = context['project']!;
    final env = context['env']!;

    final bar = '\x1b[46m\x1b[30m ORG: $org \x1b[0m '
        '\x1b[45m\x1b[30m PROJECT: $project \x1b[0m '
        '\x1b[42m\x1b[30m ENV: $env \x1b[0m';

    stdout.writeln(bar);
    stdout.writeln();
  }

  String _getPromptPrefix() {
    final cwd = Directory.current.path;
    final home = Platform.environment['HOME'] ?? '';
    final displayCwd =
        cwd.startsWith(home) ? '~${cwd.substring(home.length)}' : cwd;

    return '\x1b[35m$displayCwd\x1b[0m \x1b[36m❯\x1b[0m';
  }

  /// Simple command line parser that handles quotes.
  List<String> _parseInput(String input) {
    final args = <String>[];
    final regex = RegExp(r'[^\s"]+|"([^"]*)"');
    final matches = regex.allMatches(input);

    for (final match in matches) {
      if (match.group(1) != null) {
        args.add(match.group(1)!);
      } else {
        args.add(match.group(0)!);
      }
    }
    return args;
  }

  Future<void> _renderTui(Map<String, dynamic> context) async {
    TuiUtils.clear();
    final width = stdout.terminalColumns;
    final height = stdout.terminalLines;

    // Header bar (Full width surface background)
    stdout.write(TuiUtils.bgSurface);
    stdout.write(' ' * width);
    TuiUtils.printAt(2, 1, 'APPLAD ADMIN',
        bold: true, color: '${TuiUtils.bgSurface}${TuiUtils.accentCyan}');
    TuiUtils.printAt(width - 30, 1, 'ctx: ${context['org']} @local',
        color: '${TuiUtils.bgSurface}${TuiUtils.textPrimary}');
    stdout.write(TuiUtils.reset);

    // Calculate Dynamic Tabs
    final features =
        (context['features'] as List<dynamic>?)?.cast<String>() ?? [];
    _currentTabs = [' DASHBOARD ', ' PROJECTS '];
    if (features.contains('functions')) {
      _currentTabs.add(' FUNCTIONS ');
    }
    if (features.contains('storage')) {
      _currentTabs.add(' STORAGE ');
    }
    if (features.contains('messaging')) {
      _currentTabs.add(' MESSAGING ');
    }
    if (features.contains('realtime')) {
      _currentTabs.add(' REALTIME ');
    }
    if (features.contains('database') || features.contains('graphql')) {
      _currentTabs.add(' DATABASE ');
    }

    // Tabs - Compact Posting Style
    var currentX = 2;
    for (var i = 0; i < _currentTabs.length; i++) {
      final isActive = i == _activeTab;
      if (isActive) {
        TuiUtils.printAt(currentX, 3, _currentTabs[i],
            bold: true, color: '${TuiUtils.bgSurface}${TuiUtils.accentCyan}');
      } else {
        TuiUtils.printAt(currentX, 3, _currentTabs[i],
            color: TuiUtils.textMuted);
      }
      currentX += _currentTabs[i].length + 2;
    }
    TuiUtils.drawLine(1, 4, width, color: TuiUtils.borderNormal);

    // Content area
    if (_activeTab == 0) {
      _renderDashboardTab(context, width, height);
    } else if (_activeTab == 1) {
      _renderProjectsTab(context, width, height);
    } else if (_activeTab < _currentTabs.length) {
      _renderFeatureTab(
          context, width, height, _currentTabs[_activeTab].trim());
    }

    // Lower Shortcut Bar (Posting style)
    final shortcuts = [
      '^q Quit',
      '^r REPL',
    ];
    for (var i = 0; i < _currentTabs.length; i++) {
      shortcuts.add('${i + 1} ${_currentTabs[i].trim().substring(0, 4)}');
    }

    var sx = 2;

    // Fill the footer background
    TuiUtils.moveTo(1, height);
    stdout.write(TuiUtils.bgSurface + (' ' * width) + TuiUtils.reset);

    for (final s in shortcuts) {
      final key = s.split(' ')[0];
      final label = s.split(' ')[1];
      TuiUtils.printAt(sx, height, key,
          color: '${TuiUtils.bgSurface}${TuiUtils.accentMagenta}');
      TuiUtils.printAt(sx + key.length + 1, height, label,
          color: '${TuiUtils.bgSurface}${TuiUtils.textMuted}');
      sx += s.length + 3;
    }

    TuiUtils.moveTo(width - 5, height);
    stdout.write(
        '${TuiUtils.bgSurface}${TuiUtils.accentCyan}❯${TuiUtils.reset} ');
  }

  void _renderDashboardTab(
      Map<String, dynamic> context, int width, int height) {
    final colWidth = (width ~/ 2) - 3;
    TuiUtils.drawBox(2, 6, colWidth, 8,
        title: 'Environment Status', borderColor: TuiUtils.borderActive);
    TuiUtils.printAt(4, 7, '• Infrastructure: Docker',
        color: TuiUtils.accentCyan, maxWidth: colWidth - 4);
    TuiUtils.printAt(4, 8, '• API Gateway: http://localhost:8080',
        color: TuiUtils.textPrimary, maxWidth: colWidth - 4);
    TuiUtils.printAt(4, 10, 'Health: ', color: TuiUtils.textMuted);
    TuiUtils.printAt(12, 10, 'ONLINE', color: TuiUtils.accentGreen, bold: true);

    TuiUtils.drawBox(colWidth + 4, 6, colWidth, 8, title: 'Active Context');
    TuiUtils.printAt(colWidth + 6, 7, '• Active Org: ${context['org']}',
        color: TuiUtils.accentCyan, maxWidth: colWidth - 4);
    TuiUtils.printAt(colWidth + 6, 8, '• Project: ${context['project']}',
        color: TuiUtils.textPrimary, maxWidth: colWidth - 4);
    TuiUtils.printAt(colWidth + 6, 9, '• Env: @local',
        color: TuiUtils.textPrimary, maxWidth: colWidth - 4);

    TuiUtils.drawBox(2, 15, width - 4, height - 20, title: 'System Logs');
    TuiUtils.printAt(
        4, 16, 'Welcome to Applad Dashboard. Select a tab to explore.',
        color: TuiUtils.textMuted, maxWidth: width - 8);
    TuiUtils.printAt(4, 17, 'Run "up" to apply configuration changes.',
        color: TuiUtils.textMuted, maxWidth: width - 8);
  }

  void _renderProjectsTab(Map<String, dynamic> context, int width, int height) {
    TuiUtils.printAt(4, 6, 'Project List for ${context['org']}:',
        bold: true, color: TuiUtils.accentCyan);
    TuiUtils.printAt(6, 8, '• ${context['project']} (Current)',
        color: TuiUtils.accentGreen, bold: true);
    TuiUtils.printAt(6, 9, '• marketing-site', color: TuiUtils.textMuted);
    TuiUtils.printAt(6, 10, '• analytics-api', color: TuiUtils.textMuted);

    TuiUtils.printAt(4, 13, 'Quick Actions:',
        bold: true, color: TuiUtils.accentCyan);
    TuiUtils.printAt(6, 15, 'Run "init" to scaffold a new project.',
        color: TuiUtils.textMuted);
  }

  void _renderFeatureTab(
      Map<String, dynamic> context, int width, int height, String featureName) {
    final colWidth = width - 4;
    TuiUtils.drawBox(2, 6, colWidth, height - 10,
        title: featureName, borderColor: TuiUtils.borderActive);
    TuiUtils.printAt(4, 7, '• Managing $featureName resources',
        color: TuiUtils.accentCyan, maxWidth: colWidth - 4);
    TuiUtils.printAt(4, 8, 'Details pending implementation...',
        color: TuiUtils.textMuted, maxWidth: colWidth - 4);
  }

  void _handleTuiMouse(TuiMouseEvent event) {
    if (!event.isDown) return;

    // Detect click on tabs (Row 3, coordinated X positions based on active dynamic tabs)
    if (event.y == 3) {
      var currentX = 2;
      for (var i = 0; i < _currentTabs.length; i++) {
        final tabWidth = _currentTabs[i].length;
        if (event.x >= currentX && event.x <= currentX + tabWidth) {
          _activeTab = i;
          return;
        }
        currentX += tabWidth + 2;
      }
    }
  }

  Future<void> _handleTuiInput(String input) async {
    final parsed = int.tryParse(input);
    if (parsed != null && parsed >= 1 && parsed <= _currentTabs.length) {
      _activeTab = parsed - 1;
    } else if (input == '\n' || input == '\r') {
      // Enter key could trigger something eventually
    }
  }
}
