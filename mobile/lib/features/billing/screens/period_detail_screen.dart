import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../core/datetime/jalali.dart';
import '../../../core/l10n/app_localizations.dart';
import '../../../core/network/api_exception.dart';
import '../../../shared/formatters/money_text.dart';
import '../../../shared/widgets/calc_method.dart';
import '../../../shared/widgets/status_chip.dart';
import '../billing_controller.dart';
import '../models/billing.dart';

/// Period detail: cost items, run-calculate action, and the reviewable
/// per-unit breakdown (BR-08) with Σ reconciliation indicators (BR-09),
/// issue confirmation flow, reopen and close (US4/T052).
class PeriodDetailScreen extends ConsumerWidget {
  const PeriodDetailScreen({super.key, required this.buildingId, required this.periodId});

  final String buildingId;
  final String periodId;

  Future<void> _confirm(
    BuildContext context,
    WidgetRef ref, {
    required String title,
    required String message,
    required Future<void> Function() action,
  }) async {
    final l10n = AppLocalizations.of(context);
    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(title),
        content: Text(message),
        actions: [
          TextButton(
              onPressed: () => Navigator.pop(ctx, false),
              child: Text(l10n.cancel)),
          FilledButton(
              onPressed: () => Navigator.pop(ctx, true),
              child: Text(l10n.confirm)),
        ],
      ),
    );
    if (ok != true) return;
    try {
      await action();
    } on ApiException catch (e) {
      if (context.mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(e.serverMessage ?? l10n.errorServer)),
        );
      }
    }
  }

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final previewAsync = ref.watch(previewControllerProvider(periodId));

    return Scaffold(
      appBar: AppBar(title: Text(l10n.previewTitle)),
      body: previewAsync.when(
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (e, _) => Center(
          child: Text(e is ApiException
              ? (e.serverMessage ?? l10n.errorServer)
              : l10n.errorUnknown),
        ),
        data: (ps) {
          final period = ps.preview.period;
          final canEdit = period.status == 'draft';
          final canCalculate = period.status == 'draft' || period.status == 'calculated';
          final canIssue = period.status == 'calculated';
          final canReopen = period.status == 'calculated';
          final canClose = period.status == 'issued';
          return ListView(
            padding: const EdgeInsets.all(16),
            children: [
              // --- period header -------------------------------------------
              Card(
                child: ListTile(
                  title: Text(period.title),
                  subtitle: Text(
                    '${formatJalaliDate(DateTime.tryParse(period.startDate) ?? DateTime(2000))}'
                    ' — ${formatJalaliDate(DateTime.tryParse(period.endDate) ?? DateTime(2000))}'
                    ' • ${l10n.periodDue}: ${formatJalaliDate(DateTime.tryParse(period.dueDate) ?? DateTime(2000))}',
                  ),
                  trailing: StatusChip(kind: StatusKind.period, value: period.status),
                ),
              ),
              const SizedBox(height: 12),
              Wrap(
                spacing: 8,
                runSpacing: 8,
                children: [
                  if (canCalculate)
                    FilledButton.icon(
                      onPressed: ps.busy
                          ? null
                          : () async {
                              try {
                                await ref
                                    .read(previewControllerProvider(periodId)
                                        .notifier)
                                    .calculate();
                              } on ApiException catch (e) {
                                if (context.mounted) {
                                  ScaffoldMessenger.of(context).showSnackBar(
                                    SnackBar(content: Text(
                                        e.serverMessage ?? l10n.errorServer)),
                                  );
                                }
                              }
                            },
                      icon: const Icon(Icons.calculate_outlined),
                      label: Text(period.status == 'calculated'
                          ? l10n.recalculate
                          : l10n.calculate),
                    ),
                  if (canIssue)
                    FilledButton.icon(
                      onPressed: ps.busy
                          ? null
                          : () => _confirm(context, ref,
                              title: l10n.issue,
                              message: l10n.issueConfirm,
                              action: () async {
                              await ref
                                  .read(previewControllerProvider(periodId).notifier)
                                  .issue();
                              if (context.mounted) {
                                ScaffoldMessenger.of(context).showSnackBar(
                                    SnackBar(content: Text(l10n.issuedOk)));
                              }
                            }),
                      icon: const Icon(Icons.publish_outlined),
                      label: Text(l10n.issue),
                    ),
                  if (canReopen)
                    OutlinedButton.icon(
                      onPressed: ps.busy
                          ? null
                          : () => _confirm(context, ref,
                              title: l10n.reopen,
                              message: l10n.reopenConfirm,
                              action: () => ref
                                  .read(previewControllerProvider(periodId).notifier)
                                  .reopen()),
                      icon: const Icon(Icons.undo),
                      label: Text(l10n.reopen),
                    ),
                  if (canClose)
                    OutlinedButton.icon(
                      onPressed: ps.busy
                          ? null
                          : () => _confirm(context, ref,
                              title: l10n.closePeriod,
                              message: l10n.closeConfirm,
                              action: () => ref
                                  .read(previewControllerProvider(periodId).notifier)
                                  .close()),
                      icon: const Icon(Icons.lock_outline),
                      label: Text(l10n.closePeriod),
                    ),
                ],
              ),
              const SizedBox(height: 16),

              // --- cost items (editable only in draft) ----------------------
              Row(
                mainAxisAlignment: MainAxisAlignment.spaceBetween,
                children: [
                  Text(l10n.costItemsTitle,
                      style: Theme.of(context).textTheme.titleMedium),
                  if (canEdit)
                    TextButton.icon(
                      onPressed: () => context.push(
                        '/manager/periods/$buildingId/$periodId/cost-items/new',
                      ),
                      icon: const Icon(Icons.add),
                      label: Text(l10n.addCostItem),
                    ),
                ],
              ),
              ...ps.preview.items.map((item) => ListTile(
                    contentPadding: EdgeInsets.zero,
                    title: Text(item.title),
                    subtitle: Text(calcMethodLabel(l10n, item.method)),
                    trailing: MoneyText(amount: item.totalUsed),
                    onTap: canEdit
                        ? () => context.push(
                            '/manager/periods/$buildingId/$periodId/cost-items/${item.id}',
                            extra: item.costItem)
                        : null,
                  )),
              const Divider(height: 32),

              // --- reviewable breakdown (BR-08) ------------------------------
              if (!ps.preview.reconciled)
                Card(
                  color: Theme.of(context).colorScheme.errorContainer,
                  child: Padding(
                    padding: const EdgeInsets.all(12),
                    child: Text(l10n.reconciliationBad),
                  ),
                ),
              ...ps.preview.items.map(
                (item) => _PreviewItemCard(item: item),
              ),
              const SizedBox(height: 16),

              // --- resulting invoices ----------------------------------------
              Text(l10n.invoicesTitle,
                  style: Theme.of(context).textTheme.titleMedium),
              ...ps.preview.invoices.map(
                (inv) => ListTile(
                  contentPadding: EdgeInsets.zero,
                  title: Text(inv.unitNumber.isEmpty
                      ? inv.unitId
                      : '${l10n.unitNumber} ${inv.unitNumber}'),
                  subtitle: inv.status == 'unpaid'
                      ? null
                      : StatusChip(kind: StatusKind.invoice, value: inv.status),
                  trailing: MoneyText(
                    amount: inv.finalAmount,
                    style: Theme.of(context).textTheme.titleMedium,
                  ),
                  onTap: () => context.push('/invoice/${inv.id}'),
                ),
              ),
            ],
          );
        },
      ),
    );
  }
}

/// One cost item's per-unit breakdown with its reconciliation indicator.
class _PreviewItemCard extends StatelessWidget {
  const _PreviewItemCard({required this.item});

  final PreviewItem item;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Expanded(child: Text(item.title)),
                Icon(
                  item.reconciled ? Icons.check_circle : Icons.error,
                  size: 18,
                  color: item.reconciled ? Colors.green : Colors.red,
                ),
                const SizedBox(width: 4),
                Text(
                  item.reconciled
                      ? l10n.reconciliationOk
                      : l10n.reconciliationBad,
                  style: Theme.of(context).textTheme.labelSmall,
                ),
              ],
            ),
            const Divider(),
            ...item.shares.map(
              (s) => Padding(
                padding: const EdgeInsets.symmetric(vertical: 2),
                child: Row(
                  children: [
                    Expanded(
                      child: Text(s.unitNumber.isEmpty
                          ? s.unitId
                          : '${l10n.unitNumber} ${s.unitNumber}'),
                    ),
                    Text(
                      '${l10n.exactShare}: ${toPersianDigits(s.exactShare)}',
                      style: Theme.of(context).textTheme.bodySmall,
                    ),
                    const SizedBox(width: 12),
                    SizedBox(
                      width: 130,
                      child: Align(
                        alignment: AlignmentDirectional.centerEnd,
                        child: MoneyText(
                          amount: s.roundedShare,
                          showCurrencySuffix: false,
                        ),
                      ),
                    ),
                  ],
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }
}
