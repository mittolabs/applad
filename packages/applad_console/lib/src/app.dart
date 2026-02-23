import 'package:flutter/material.dart';
import 'navigation/app_router.dart';
import 'theme/theme.dart';

/// Root application widget for the Applad admin dashboard.
class ApplAdAdminApp extends StatelessWidget {
  const ApplAdAdminApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp.router(
      title: 'Applad Admin',
      theme: ApplAdTheme.light(),
      darkTheme: ApplAdTheme.dark(),
      routerConfig: AppRouter.router,
      debugShowCheckedModeBanner: false,
    );
  }
}
