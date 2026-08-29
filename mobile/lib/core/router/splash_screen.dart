import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:hamsa/core/l10n/app_localizations.dart';
import 'package:hamsa/core/theme/app_theme.dart';

/// Shown while the persisted session is being restored from secure storage.
class SplashScreen extends ConsumerWidget {
  const SplashScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context);
    return Scaffold(
      body: Center(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            // Brand mark: same treatment as the auth screens.
            Container(
              width: 72,
              height: 72,
              decoration: BoxDecoration(
                color: theme.colorScheme.primary,
                borderRadius: BorderRadius.circular(AppTheme.radiusLarge),
              ),
              child: const Icon(
                Icons.apartment,
                size: 36,
                color: Colors.white,
              ),
            ),
            const SizedBox(height: AppTheme.spaceXl),
            Text(l10n.appTitle, style: theme.textTheme.displaySmall),
            const SizedBox(height: AppTheme.spaceS),
            Text(
              l10n.splashMessage,
              style: theme.textTheme.bodyMedium?.copyWith(
                color: theme.colorScheme.onSurfaceVariant,
              ),
            ),
            const SizedBox(height: AppTheme.spaceXxxl),
            const CircularProgressIndicator(),
          ],
        ),
      ),
    );
  }
}
