import 'dart:async';
import 'dart:io';
import 'package:applad_core/applad_core.dart';
import 'package:path/path.dart' as p;
import 'output.dart';
import '../commands/applad_command_runner.dart';
import '../security/trust_manager.dart';
import '../security/session_manager.dart';
import '../utils/config_finder.dart';
import '../utils/tui_utils.dart';

enum ShellMode { repl, tui }

enum TuiScope { org, project }

/// An interactive REPL and TUI Dashboard for the Applad CLI.
final class InteractiveShell {
  final ApplAdCommandRunner runner;
  ShellMode _mode = ShellMode.repl;
  TuiScope _tuiScope = TuiScope.org;
  bool _tuiActive = false;

  // Org Scope State
  int _activeOrgTab = 0; // 0: Projects, 1: Domains, etc.
  List<String> _orgTabs = [
    ' Projects ',
    ' Domains ',
    ' Members ',
    ' Usage ',
    ' Billing ',
    ' Settings '
  ];

  // Project Scope State
  String? _selectedProject;
  int _activeTab = 0; // Sidebar selections
  List<String> _currentTabs = [' Overview '];

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

    late final StreamSubscription sigIntSub;
    late final StreamSubscription sigTermSub;

    try {
      if (Platform.isMacOS || Platform.isLinux) {
        sigIntSub = ProcessSignal.sigint.watch().listen((_) => _exitCleanly());
        sigTermSub =
            ProcessSignal.sigterm.watch().listen((_) => _exitCleanly());
      }

      while (true) {
        final context = _resolveContext();

        if (_mode == ShellMode.tui) {
          if (!_tuiActive) {
            stdout.write(TuiUtils.enterAltScreen);
            _tuiActive = true;
          }
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
                // Simple check for end of escape sequence (M or m for mouse, or roughly a max length)
                final str = String.fromCharCodes(bytes);
                if (str.endsWith('M') ||
                    str.endsWith('m') ||
                    bytes.length > 20) {
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
              if (_tuiActive) {
                stdout.write(TuiUtils.exitAltScreen);
                _tuiActive = false;
              }
              _clearScreen();
              _printWelcome();
              continue;
            }

            final mouseEvent = TuiUtils.parseSgrMouse(input);
            if (mouseEvent != null) {
              await _handleTuiMouse(mouseEvent);
            } else {
              await _handleTuiInput(input);
            }
          } finally {
            stdin.lineMode = true;
            stdin.echoMode = true;
          }
          continue;
        }

        final width = stdout.terminalColumns;
        final borderColor = '\x1b[36m'; // Cyan
        final reset = '\x1b[0m';

        // interaction box around the input only
        stdout.write('\n$borderColor┌── Applad Repl ' +
            ('─' * (width - 16)) +
            '┐$reset\n');
        final promptPrefix = _getPromptPrefix();
        stdout.write('$promptPrefix ');
        final input = stdin.readLineSync()?.trim() ?? '';
        stdout.write('\x1b[0m'); // Reset formatting after input

        // Close the box
        stdout.write('$borderColor└' + ('─' * (width - 2)) + '┘$reset\n');

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

        if (input == 'console' || input == 'ui' || input == 'dashboard') {
          final hasOrg = context['org'] != '-' && context['org'] != 'error';
          if (!hasOrg) {
            Output.blank();
            Output.info('Console requires an active project.');
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
      if (Platform.isMacOS || Platform.isLinux) {
        sigIntSub.cancel();
        sigTermSub.cancel();
      }
      _exitCleanly();
    }
  }

  void _exitCleanly() {
    stdout.write(TuiUtils.disableMouse);
    if (_tuiActive) {
      stdout.write(TuiUtils.exitAltScreen);
      _tuiActive = false;
    }
    stdout.write('\x1b[0m'); // Absolute system formatting reset
    stdout.write('\x1b[?25h'); // Ensure cursor is visible
    _clearScreen();
    exit(0);
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

    final context = _resolveContext();
    final hasProject = context['org'] != '-' && context['org'] != 'error';
    final isLoggedIn = SessionManager.isLoggedIn();

    if (hasProject && isLoggedIn) {
      stdout
          .writeln('\x1b[2m  3. Type "console" to enter the Admin TUI.\x1b[0m');
    }

    stdout.writeln(
        '\x1b[2m  4. Type "exit" to leave the interactive shell.\x1b[0m');
  }

  Map<String, dynamic> _resolveContext() {
    final finder = const ConfigFinder();
    final root = finder.findRoot();
    if (root == null) {
      return {
        'org': '-',
        'project': '-',
        'env': 'local',
        'features': <String>[],
        'allProjects': <String>[],
      };
    }

    try {
      final merger = ConfigMerger();
      final config = merger.merge(root);

      // Discover all projects in the org
      final orgsDir = Directory(p.join(root, 'orgs', config.org.id));
      final allProjects = <String>[];
      if (orgsDir.existsSync()) {
        for (final entity in orgsDir.listSync()) {
          if (entity is Directory) {
            final projectName = p.basename(entity.path);
            if (!projectName.startsWith('.') &&
                File(p.join(entity.path, 'project.yaml')).existsSync()) {
              allProjects.add(projectName);
            }
          }
        }
      }

      return {
        'org': config.org.id,
        'project': _selectedProject ?? config.project.id,
        'env': 'local',
        'features': config.project.enabledFeatures,
        'allProjects': allProjects,
      };
    } catch (_) {
      return {
        'org': 'error',
        'project': 'error',
        'env': 'local',
        'features': <String>[],
        'allProjects': <String>[],
      };
    }
  }

  String _getPromptPrefix() {
    final cwd = Directory.current.path;
    final home = Platform.environment['HOME'] ?? '';
    final displayCwd =
        cwd.startsWith(home) ? '~${cwd.substring(home.length)}' : cwd;

    final borderColor = '\x1b[36m'; // Cyan
    final reset = '\x1b[0m';

    // Box side border
    return '$borderColor│$reset \x1b[35m$displayCwd\x1b[0m \x1b[36m❯\x1b[0m';
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

    if (_tuiScope == TuiScope.org) {
      _renderOrgDashboard(context, width, height);
    } else {
      _renderProjectDashboard(context, width, height);
    }
  }

  void _renderOrgDashboard(
      Map<String, dynamic> context, int width, int height) {
    // Org Header
    TuiUtils.moveTo(1, 1);
    stdout.write(TuiUtils.bgSurface + (' ' * width) + TuiUtils.reset);
    TuiUtils.printAt(2, 1, 'APPLAD CONSOLE // ${context['org']}',
        bold: true, color: '${TuiUtils.bgSurface}${TuiUtils.accentCyan}');
    TuiUtils.printAt(width - 15, 1, 'Free Tier',
        color: '${TuiUtils.bgSurface}${TuiUtils.textMuted}');

    // Horizontal Tabs
    var currentX = 2;
    for (var i = 0; i < _orgTabs.length; i++) {
      final isActive = i == _activeOrgTab;
      if (isActive) {
        TuiUtils.printAt(currentX, 3, _orgTabs[i],
            bold: true, color: '${TuiUtils.bgMagenta}${TuiUtils.black}');
      } else {
        TuiUtils.printAt(currentX, 3, _orgTabs[i],
            color: '${TuiUtils.bgBase}${TuiUtils.textMuted}');
      }
      currentX += _orgTabs[i].length + 2;
    }
    TuiUtils.drawLine(1, 4, width, color: TuiUtils.borderNormal);

    // Tab Content
    if (_activeOrgTab == 0) {
      // Projects
      final allProjects =
          (context['allProjects'] as List<dynamic>?)?.cast<String>() ?? [];

      TuiUtils.printAt(4, 6, 'Projects',
          bold: true, color: TuiUtils.textPrimary);

      // Top Right: Create Project Action Button
      final btnLabel = ' + Create project ';
      final btnX = width - btnLabel.length - 4;
      TuiUtils.printAt(btnX, 6, btnLabel,
          color: '${TuiUtils.bgBase}${TuiUtils.accentMagenta}');

      var cardX = 4;
      var cardY = 8;
      final cardWidth = (width ~/ 3) - 4;

      if (allProjects.isEmpty) {
        TuiUtils.drawBox(cardX, cardY, cardWidth, 8,
            title: 'No Projects Found');
        TuiUtils.printAt(cardX + 2, cardY + 2, 'Run `applad init`',
            color: TuiUtils.textMuted);
      } else {
        for (var i = 0; i < allProjects.length; i++) {
          final pName = allProjects[i];
          TuiUtils.drawBox(cardX, cardY, cardWidth, 8,
              title: pName, borderColor: TuiUtils.borderNormal);
          TuiUtils.printAt(cardX + 2, cardY + 2, pName,
              bold: true, color: TuiUtils.textPrimary);
          TuiUtils.printAt(cardX + 2, cardY + 4, '</> Web',
              color: TuiUtils.accentCyan);
          TuiUtils.printAt(cardX + cardWidth - 12, cardY + 6, 'Local',
              color: TuiUtils.textMuted);

          cardX += cardWidth + 2;
          if (cardX + cardWidth > width) {
            // wrap
            cardX = 4;
            cardY += 10;
          }
        }
      }

      // Always draw the + Create project card at the end of the list
      TuiUtils.drawBox(cardX, cardY, cardWidth, 8,
          title: 'New Project', borderColor: TuiUtils.borderNormal);
      TuiUtils.printAt(
          cardX + cardWidth ~/ 2 - 8, cardY + 4, '+ Create project',
          color: TuiUtils.accentMagenta, bold: true);
    } else {
      TuiUtils.printAt(4, 8, 'Feature pending implementation',
          color: TuiUtils.textMuted);
    }

    // Lower Shortcut Bar (Posting style)
    final shortcuts = ['^q Quit', '^r REPL'];
    var sx = 2;
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

  void _renderProjectDashboard(
      Map<String, dynamic> context, int width, int height) {
    TuiUtils.moveTo(1, 1);
    stdout.write(TuiUtils.bgSurface + (' ' * width) + TuiUtils.reset);
    TuiUtils.printAt(2, 1, 'APPLAD CONSOLE',
        bold: true, color: '${TuiUtils.bgSurface}${TuiUtils.accentCyan}');
    TuiUtils.printAt(20, 1, 'ctx: ${context['org']} / ${context['project']}',
        color: '${TuiUtils.bgSurface}${TuiUtils.textPrimary}');
    TuiUtils.printAt(width - 40, 1, 'API Endpoint: http://localhost:8080',
        color: '${TuiUtils.bgSurface}${TuiUtils.textMuted}');

    // Calculate Dynamic Sidebar Items
    final features =
        (context['features'] as List<dynamic>?)?.cast<String>() ?? [];
    _currentTabs = [' Overview '];
    _currentTabs.add(' Auth '); // Standard for most apps

    if (features.contains('database') || features.contains('graphql')) {
      _currentTabs.add(' Databases ');
    }
    if (features.contains('functions')) {
      _currentTabs.add(' Functions ');
    }
    if (features.contains('messaging')) {
      _currentTabs.add(' Messaging ');
    }
    if (features.contains('storage')) {
      _currentTabs.add(' Storage ');
    }
    if (features.contains('realtime')) {
      _currentTabs.add(' Realtime ');
    }
    _currentTabs.add(' Settings ');

    // Vertical Sidebar Render
    final sidebarWidth = 22;
    var currentY = 4;
    for (var i = 0; i < _currentTabs.length; i++) {
      final isActive = i == _activeTab;
      final label = _currentTabs[i];
      final paddedLabel = label.padRight(sidebarWidth - 4);

      if (isActive) {
        TuiUtils.printAt(2, currentY, '  $paddedLabel',
            bold: true, color: '${TuiUtils.bgMagenta}${TuiUtils.black}');
      } else {
        TuiUtils.printAt(2, currentY, '  $paddedLabel',
            color: '${TuiUtils.bgBase}${TuiUtils.textMuted}');
      }
      currentY += 2;
    }

    // Content area
    final contentX = sidebarWidth + 2;
    final contentWidth = width - contentX - 2;

    if (_activeTab == 0) {
      _renderOverviewTab(context, contentX, contentWidth, height);
    } else if (_currentTabs[_activeTab].trim() == 'Settings') {
      _renderSettingsTab(context, contentX, contentWidth, height);
    } else {
      _renderFeatureTab(context, contentX, contentWidth, height,
          _currentTabs[_activeTab].trim());
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

  void _renderOverviewTab(
      Map<String, dynamic> context, int startX, int width, int height) {
    // Top Row Cards (Bandwidth / Requests)
    final topCardWidth = (width ~/ 2) - 2;
    TuiUtils.drawBox(startX, 4, topCardWidth, 8,
        title: 'Bandwidth', borderColor: TuiUtils.borderNormal);
    TuiUtils.printAt(startX + 2, 6, '340 KB',
        color: TuiUtils.textPrimary, bold: true);
    TuiUtils.printAt(startX + 2, 10, 'Last 30 days', color: TuiUtils.textMuted);

    TuiUtils.drawBox(startX + topCardWidth + 2, 4, topCardWidth, 8,
        title: 'Requests', borderColor: TuiUtils.borderNormal);
    TuiUtils.printAt(startX + topCardWidth + 4, 6, '171',
        color: TuiUtils.textPrimary, bold: true);
    TuiUtils.printAt(startX + topCardWidth + 4, 10, 'Last 30 days',
        color: TuiUtils.textMuted);

    // Grid Cards
    var currentX = startX;
    final gridY = 13;
    final colWidth = (width ~/ 4) - 2;

    TuiUtils.drawBox(currentX, gridY, colWidth, 6,
        title: 'Database', borderColor: TuiUtils.borderNormal);
    TuiUtils.printAt(currentX + 2, gridY + 2, '0',
        color: TuiUtils.textPrimary, bold: true);
    TuiUtils.printAt(currentX + 2, gridY + 4, 'Rows',
        color: TuiUtils.textMuted);
    currentX += colWidth + 2;

    TuiUtils.drawBox(currentX, gridY, colWidth, 6,
        title: 'Storage', borderColor: TuiUtils.borderNormal);
    TuiUtils.printAt(currentX + 2, gridY + 2, '0 B',
        color: TuiUtils.textPrimary, bold: true);
    TuiUtils.printAt(currentX + 2, gridY + 4, 'Storage',
        color: TuiUtils.textMuted);
    currentX += colWidth + 2;

    TuiUtils.drawBox(currentX, gridY, colWidth, 6,
        title: 'Auth', borderColor: TuiUtils.borderNormal);
    TuiUtils.printAt(currentX + 2, gridY + 2, '1',
        color: TuiUtils.textPrimary, bold: true);
    TuiUtils.printAt(currentX + 2, gridY + 4, 'Users',
        color: TuiUtils.textMuted);
    currentX += colWidth + 2;

    TuiUtils.drawBox(currentX, gridY, colWidth, 6,
        title: 'Functions', borderColor: TuiUtils.borderNormal);
    TuiUtils.printAt(currentX + 2, gridY + 2, '1.3K',
        color: TuiUtils.textPrimary, bold: true);
    TuiUtils.printAt(currentX + 2, gridY + 4, 'Executions',
        color: TuiUtils.textMuted);
  }

  void _renderSettingsTab(
      Map<String, dynamic> context, int startX, int width, int height) {
    TuiUtils.drawBox(startX, 4, width, height - 10,
        title: 'Settings', borderColor: TuiUtils.borderNormal);
    TuiUtils.printAt(startX + 2, 6, 'Project Settings:',
        bold: true, color: TuiUtils.accentCyan);
    TuiUtils.printAt(startX + 2, 8, '• Project ID: ${context['project']}',
        color: TuiUtils.textPrimary);
    TuiUtils.printAt(startX + 2, 9, '• Organization: ${context['org']}',
        color: TuiUtils.textPrimary);
  }

  void _renderFeatureTab(Map<String, dynamic> context, int startX, int width,
      int height, String featureName) {
    TuiUtils.drawBox(startX, 4, width, height - 10,
        title: featureName, borderColor: TuiUtils.borderActive);
    TuiUtils.printAt(startX + 2, 6, '• Managing $featureName resources',
        color: TuiUtils.accentCyan);
    TuiUtils.printAt(startX + 2, 7, 'Details pending implementation...',
        color: TuiUtils.textMuted);
  }

  Future<void> _startProjectCreationFlow() async {
    // Suspend TUI
    stdout.write(TuiUtils.disableMouse);
    if (_tuiActive) {
      stdout.write(TuiUtils.exitAltScreen);
      _tuiActive = false;
    }
    _clearScreen();

    // Run the interactive CLI creation flow
    stdout.writeln('\x1b[35mStarting Project Creation Flow...\x1b[0m\n');

    try {
      // Drop terminal back to standard line buffering for prompts
      stdin.lineMode = true;
      stdin.echoMode = true;
      await runner.run(['init']);
    } catch (e) {
      Output.error('Failed to init project: $e');
      // Pause so the user can see the error
      await Future.delayed(Duration(seconds: 2));
    } finally {
      // Restore TUI mode settings
      stdin.lineMode = false;
      stdin.echoMode = false;
    }

    // Restart TUI state
    _tuiScope = TuiScope.org;
  }

  Future<void> _handleTuiMouse(TuiMouseEvent event) async {
    if (!event.isDown) return;

    if (_tuiScope == TuiScope.org) {
      // Handle Org horizontal tabs
      if (event.y == 3) {
        var currentX = 2;
        for (var i = 0; i < _orgTabs.length; i++) {
          final tabWidth = _orgTabs[i].length;
          if (event.x >= currentX && event.x <= currentX + tabWidth) {
            _activeOrgTab = i;
            return;
          }
          currentX += tabWidth + 2;
        }
      }

      // Handle Project Card Clicks
      if (_activeOrgTab == 0) {
        final width = stdout.terminalColumns;

        // Detect '+ Create project' button click
        if (event.y == 6 && event.x >= width - 25) {
          await _startProjectCreationFlow();
          return;
        }

        if (event.y >= 8) {
          final context = _resolveContext();
          final allProjects =
              (context['allProjects'] as List<dynamic>?)?.cast<String>() ?? [];
          final cardWidth = (width ~/ 3) - 4;

          var cardX = 4;
          var cardY = 8;

          for (var i = 0; i <= allProjects.length; i++) {
            if (event.x >= cardX &&
                event.x <= cardX + cardWidth &&
                event.y >= cardY &&
                event.y <= cardY + 8) {
              if (i == allProjects.length) {
                // Clicked the [+ Create project] card at the end
                await _startProjectCreationFlow();
                return;
              } else {
                // Clicked an existing project
                _selectedProject = allProjects[i];
                _tuiScope = TuiScope.project;
                _activeTab = 0; // Reset sidebar
                return;
              }
            }
            cardX += cardWidth + 2;
            if (cardX + cardWidth > width) {
              cardX = 4;
              cardY += 10;
            }
          }
        }
      }
    } else {
      // Breadcrumb click to go back
      if (event.y == 1 && event.x < 15) {
        _tuiScope = TuiScope.org;
        return;
      }

      // Detect Project sidebar clicks
      final sidebarWidth = 22;
      if (event.x >= 2 && event.x <= sidebarWidth) {
        if (event.y >= 4 && event.y < 4 + (_currentTabs.length * 2)) {
          final index = (event.y - 4) ~/ 2;
          if (index >= 0 && index < _currentTabs.length) {
            if ((event.y - 4) % 2 == 0) {
              _activeTab = index;
              return;
            }
          }
        }
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
