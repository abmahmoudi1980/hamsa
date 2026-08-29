import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import '../../../core/l10n/app_localizations.dart';
import '../../../core/theme/app_theme.dart';
import '../../../shared/widgets/menu_card.dart';

import '../../auth/auth_controller.dart';

/// Resident navigation shell — full panel arrives in US9 (T083/T084).
/// Logout lives here from day one so a resident is never stuck signed in.
class ResidentShellScreen extends ConsumerWidget {
  const ResidentShellScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final theme = Theme.of(context);
    return Scaffold(
      appBar: AppBar(
        title: Text(l10n.residentShellTitle),
        actions: [
          IconButton(
            icon: const Icon(Icons.logout),
            tooltip: l10n.logout,
            onPressed: () => ref.read(authControllerProvider.notifier).logout(),
          ),
        ],
      ),
      body: ListView(
        padding: AppTheme.pagePadding,
        children: [
          Text(l10n.residentShellTitle, style: theme.textTheme.titleLarge),
          const SizedBox(height: AppTheme.spaceXs),
          Text(l10n.shellUnderConstruction, style: theme.textTheme.bodySmall),
          const SizedBox(height: AppTheme.spaceL),
          // US4 (T053) + US5 (T061): charges and payment history entries;
          // the remaining US9 menu items arrive with T084.
          MenuCard(
            icon: Icons.receipt_long,
            title: l10n.myCharges,
            onTap: () => context.push('/home/charges'),
          ),
          const SizedBox(height: AppTheme.spaceM),
          MenuCard(
            icon: Icons.payments_outlined,
            title: l10n.paymentHistoryTitle,
            onTap: () => context.push('/home/payments'),
          ),
        ],
      ),
    );
  }
}
