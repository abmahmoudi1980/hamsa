import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/datetime/jalali.dart';
import '../../../core/l10n/app_localizations.dart';
import '../../../core/theme/app_theme.dart';
import '../../../shared/formatters/money_text.dart';
import '../../../shared/widgets/empty_state.dart';
import '../../../shared/widgets/status_chip.dart';
import '../../../shared/widgets/status_labels.dart';
import '../payment_controller.dart';

/// US5/T061 — resident payment history (GET /me/payments): amount, method,
/// Jalali paid_at, status chip, and the gateway tracking number as receipt.
class PaymentHistoryScreen extends ConsumerWidget {
  const PaymentHistoryScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final payments = ref.watch(myPaymentsControllerProvider);
    return Scaffold(
      appBar: AppBar(title: Text(l10n.paymentHistoryTitle)),
      body: payments.when(
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (e, _) =>
            EmptyState(icon: Icons.error_outline, title: l10n.errorServer),
        data: (items) {
          if (items.isEmpty) return EmptyState(title: l10n.noPayments);
          return RefreshIndicator(
            onRefresh: () =>
                ref.read(myPaymentsControllerProvider.notifier).refresh(),
            child: ListView.separated(
              padding: AppTheme.pagePadding,
              itemCount: items.length,
              separatorBuilder: (_, _) => const SizedBox(height: 10),
              itemBuilder: (context, i) {
                final p = items[i];
                return Card(
                  child: ListTile(
                    leading: const Icon(Icons.payments_outlined),
                    title: Row(
                      children: [
                        MoneyText(amount: p.amount),
                        const Spacer(),
                        StatusChip(kind: StatusKind.payment, value: p.status),
                      ],
                    ),
                    subtitle: Text(
                      '${paymentMethodLabels[p.method] ?? p.method}'
                      ' — ${formatJalaliDate(DateTime.parse(p.paidAt))}'
                      '${p.invoiceNumber == null ? '' : ' — ${p.invoiceNumber!}'}'
                      '${p.trackingNumber == null ? '' : ' — ${toPersianDigits(p.trackingNumber!)}'}',
                    ),
                  ),
                );
              },
            ),
          );
        },
      ),
    );
  }
}
