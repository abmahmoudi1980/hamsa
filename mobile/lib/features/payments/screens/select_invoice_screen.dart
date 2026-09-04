import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../core/datetime/jalali.dart';
import '../../../core/l10n/app_localizations.dart';
import '../../../core/theme/app_theme.dart';
import '../../../shared/formatters/money_text.dart';
import '../../../shared/widgets/empty_state.dart';
import '../../../shared/widgets/status_chip.dart';
import '../../billing/billing_controller.dart';
import '../../billing/models/billing.dart';

/// Manager picks which outstanding invoice a manual payment targets.
///
/// Entry point for [recordPayment] from the payment ledger: lists the
/// building's issued invoices that still carry a balance (unpaid/partial —
/// the same `_payable` rule the invoice detail enforces, since the server
/// rejects payments against anything else with a Persian 409). Tapping an
/// invoice opens the record-payment form; returning refreshes the list so a
/// fully-paid invoice drops out.
class SelectInvoiceScreen extends ConsumerStatefulWidget {
  const SelectInvoiceScreen({super.key, required this.buildingId});

  final String buildingId;

  @override
  ConsumerState<SelectInvoiceScreen> createState() =>
      _SelectInvoiceScreenState();
}

class _SelectInvoiceScreenState extends ConsumerState<SelectInvoiceScreen> {
  List<Invoice>? _invoices;
  bool _loading = false;
  String? _error;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) => _load());
  }

  Future<void> _load() async {
    final l10n = AppLocalizations.of(context);
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final (items, _) = await ref
          .read(billingRepositoryProvider)
          .listBuildingInvoices(widget.buildingId, pageSize: 100);
      if (!mounted) return;
      setState(() => _invoices = items.where(_payable).toList(growable: false));
    } catch (_) {
      if (!mounted) return;
      setState(() => _error = l10n.errorServer);
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  /// Issued and still carrying an outstanding amount — mirrors the invoice
  /// detail's `_payable` so the picker never offers a 409-bound target.
  bool _payable(Invoice inv) =>
      inv.invoiceNumberPresent &&
      (inv.status == 'unpaid' || inv.status == 'partial');

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return Scaffold(
      appBar: AppBar(title: Text(l10n.recordPayment)),
      body: _body(l10n),
    );
  }

  Widget _body(AppLocalizations l10n) {
    if (_loading) {
      return const Center(child: CircularProgressIndicator());
    }
    if (_error != null) {
      return EmptyState(
        icon: Icons.error_outline,
        title: _error!,
        actionLabel: l10n.retry,
        onAction: _load,
      );
    }
    final invoices = _invoices ?? const <Invoice>[];
    if (invoices.isEmpty) {
      return EmptyState(
        title: l10n.invoicesTitle,
        subtitle: l10n.emptyStateSubtitle,
      );
    }
    return RefreshIndicator(
      onRefresh: _load,
      child: ListView.separated(
        padding: AppTheme.pagePadding,
        itemCount: invoices.length,
        separatorBuilder: (_, _) => const SizedBox(height: 10),
        itemBuilder: (context, i) {
          final inv = invoices[i];
          final outstanding = inv.finalAmount - inv.paidAmount;
          return Card(
            child: ListTile(
              title: Text(inv.unitNumber.isEmpty
                  ? inv.invoiceNumber
                  : '${l10n.unitNumber} ${toPersianDigits(inv.unitNumber)}'),
              subtitle: Text(
                '${inv.periodTitle.isEmpty ? inv.invoiceNumber : inv.periodTitle}'
                ' — ${l10n.outstandingLabel}:',
              ),
              trailing: Row(
                mainAxisSize: MainAxisSize.min,
                children: [
                  StatusChip(kind: StatusKind.invoice, value: inv.status),
                  const SizedBox(width: AppTheme.spaceS),
                  MoneyText(amount: outstanding),
                ],
              ),
              onTap: () async {
                await context.push('/manager/records-payment/${inv.id}');
                await _load();
              },
            ),
          );
        },
      ),
    );
  }
}
