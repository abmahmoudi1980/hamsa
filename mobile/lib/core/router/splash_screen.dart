import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:hamsa/core/l10n/app_localizations.dart';

/// Shown while the persisted session is being restored from secure storage.
class SplashScreen extends ConsumerWidget {
  const SplashScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return Scaffold(
      body: Center(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            const FlutterLogo(size: 72),
            const SizedBox(height: 16),
            Text(AppLocalizations.of(context).splashMessage),
          ],
        ),
      ),
    );
  }
}
