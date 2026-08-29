import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:hamsa/core/l10n/app_localizations.dart';

import '../../auth/auth_controller.dart';

/// Resident navigation shell — full panel arrives in US9 (T083/T084).
/// Logout lives here from day one so a resident is never stuck signed in.
class ResidentShellScreen extends ConsumerWidget {
  const ResidentShellScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
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
        padding: const EdgeInsets.all(16),
        children: [
          // US4 (T053) + US5 (T061): charges and payment history entries;
          // the remaining US9 menu items arrive with T084.
          Card(
            child: ListTile(
              leading: const Icon(Icons.receipt_long),
              title: Text(l10n.myCharges),
              onTap: () => context.push('/home/charges'),
            ),
          ),
          Card(
            child: ListTile(
              leading: const Icon(Icons.payments_outlined),
              title: Text(l10n.paymentHistoryTitle),
              onTap: () => context.push('/home/payments'),
            ),
          ),
          Center(child: Text(l10n.shellUnderConstruction)),
        ],
      ),
    );
  }
}
