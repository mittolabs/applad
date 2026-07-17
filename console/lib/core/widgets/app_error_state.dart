import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:lucide_icons_flutter/lucide_icons.dart';
import '../theme/console_colors.dart';

/// Converts any exception into a short, human-readable message.
String friendlyError(dynamic e) {
  if (e is DioException) {
    if (e.type == DioExceptionType.connectionError ||
        e.type == DioExceptionType.unknown) {
      return 'Cannot connect to the server. Make sure Applad is running.';
    }
    if (e.type == DioExceptionType.connectionTimeout ||
        e.type == DioExceptionType.receiveTimeout ||
        e.type == DioExceptionType.sendTimeout) {
      return 'Request timed out. The server may be starting up.';
    }
    final data = e.response?.data;
    if (data is Map) {
      final msg = data['message'] ?? data['error'];
      if (msg is String && msg.isNotEmpty) return msg;
    }
    final code = e.response?.statusCode;
    if (code != null) return 'Server returned an error ($code).';
    return 'Network error. Please try again.';
  }
  return e
      .toString()
      .replaceFirst('Exception: ', '')
      .replaceFirst('Error: ', '');
}

class _ErrorKind {
  final IconData icon;
  final Color iconColor;
  final Color bgColor;
  final String title;
  const _ErrorKind(this.icon, this.iconColor, this.bgColor, this.title);
}

_ErrorKind _classify(dynamic e) {
  if (e is DioException) {
    if (e.type == DioExceptionType.connectionError ||
        e.type == DioExceptionType.unknown) {
      return const _ErrorKind(
        LucideIcons.wifiOff,
        Color(0xFFEF4444),
        Color(0x1FEF4444),
        'Server unreachable',
      );
    }
    if (e.type == DioExceptionType.connectionTimeout ||
        e.type == DioExceptionType.receiveTimeout ||
        e.type == DioExceptionType.sendTimeout) {
      return const _ErrorKind(
        LucideIcons.clock,
        Color(0xFFF59E0B),
        Color(0x1FF59E0B),
        'Request timed out',
      );
    }
    final code = e.response?.statusCode ?? 0;
    if (code == 401 || code == 403) {
      return const _ErrorKind(
        LucideIcons.lock,
        Color(0xFFF59E0B),
        Color(0x1FF59E0B),
        'Access denied',
      );
    }
    if (code >= 500) {
      return const _ErrorKind(
        LucideIcons.serverCrash,
        Color(0xFFEF4444),
        Color(0x1FEF4444),
        'Server error',
      );
    }
  }
  return const _ErrorKind(
    LucideIcons.alertTriangle,
    Color(0xFFEF4444),
    Color(0x1FEF4444),
    'Something went wrong',
  );
}

class AppErrorState extends StatelessWidget {
  final dynamic error;
  final VoidCallback? onRetry;

  const AppErrorState({super.key, required this.error, this.onRetry});

  @override
  Widget build(BuildContext context) {
    final cs = consoleColors(context);
    final kind = _classify(error);
    final message = friendlyError(error);

    return Center(
      child: Padding(
        padding: const EdgeInsets.all(48),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Container(
              width: 48,
              height: 48,
              decoration: BoxDecoration(
                color: kind.bgColor,
                borderRadius: BorderRadius.circular(12),
              ),
              child: Icon(kind.icon, size: 22, color: kind.iconColor),
            ),
            const SizedBox(height: 16),
            Text(
              kind.title,
              style: TextStyle(
                color: cs.textPrimary,
                fontSize: 15,
                fontWeight: FontWeight.w500,
              ),
              textAlign: TextAlign.center,
            ),
            const SizedBox(height: 6),
            Text(
              message,
              style: TextStyle(
                color: cs.textMuted,
                fontSize: 13,
              ),
              textAlign: TextAlign.center,
            ),
            const SizedBox(height: 16),
            FilledButton.icon(
              onPressed: onRetry ?? () {
                // No explicit retry: re-navigate to the same route,
                // which rebuilds the widget tree and re-triggers providers.
                final location = GoRouterState.of(context).uri.toString();
                context.go(location);
              },
              icon: const Icon(LucideIcons.refreshCw, size: 13),
              label: const Text('Try again'),
              style: FilledButton.styleFrom(
                backgroundColor: const Color(0xFF3472A4),
                padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 10),
                shape: RoundedRectangleBorder(
                    borderRadius: BorderRadius.circular(8)),
                textStyle: const TextStyle(
                    fontSize: 13, fontWeight: FontWeight.w500),
              ),
            ),
          ],
        ),
      ),
    );
  }
}
