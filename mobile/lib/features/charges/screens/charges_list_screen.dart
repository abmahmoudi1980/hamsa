import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../core/l10n/app_localizations.dart';
import '../../../shared/formatters/money_text.dart';
import '../../../shared/widgets/empty_state.dart';
import '../../../shared/widgets/status_chip.dart';
import '../../billing/billing_controller.dart';

/// Resident charges list (US4/T053, part of the US9 menu): the resident's
/// own invoices, newest first; tap → shared invoice detail.
class ChargesListScreen extends ConsumerWidget {
  const ChargesListScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final async = ref.watch(myInvoicesControllerProvider);

    return Scaffold(
      appBar: AppBar(title: Text(l10n.myCharges)),
      body: async.when(
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (e, _) => Center(child: Text(l10n.errorServer)),
        data: (invoices) => invoices.isEmpty
            ? EmptyState(title: l10n.myCharges, subtitle: l10n.emptyStateSubtitle)
            : RefreshIndicator(
                onRefresh: () async =>
                    ref.invalidate(myInvoicesControllerProvider),
                child: ListView.separated(
                  itemCount: invoices.length,
                  separatorBuilder: (_, _) => const Divider(height: 1),
                  itemBuilder: (context, i) {
                    final inv = invoices[i];
                    return ListTile(
                      title: Text(inv.periodTitle.isEmpty
                          ? inv.invoiceNumber
                          : inv.periodTitle),
                      subtitle: inv.unitNumber.isEmpty
                          ? null
                          : Text('${l10n.unitNumber} ${inv.unitNumber}'),
                      trailing: Row(
                        mainAxisSize: MainAxisSize.min,
                        children: [
                          StatusChip(
                              kind: StatusKind.invoice, value: inv.status),
                          const SizedBox(width: 8),
                          MoneyText(amount: inv.finalAmount, showCurrencySuffix: false),
                        ],
                      ),
                      onTap: () => context.push('/invoice/${inv.id}'),
                    );
                  },
                ),
              ),
      ),
    );
  }
}
