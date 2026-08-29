import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/datetime/jalali.dart';
import '../../../core/l10n/app_localizations.dart';
import '../../../core/network/api_exception.dart';
import '../../../shared/formatters/money_text.dart';
import '../../../shared/widgets/calc_method.dart';
import '../../../shared/widgets/status_chip.dart';
import '../../auth/auth_controller.dart';
import '../../auth/models/user_session.dart';
import '../../billing/billing_controller.dart';
import '../../billing/models/billing.dart';

/// Invoice detail (US4/T053) — shared manager/resident surface. Items are
/// grouped by kind (charge lines, late fee, adjustments); managers get the
/// explicit corrections: cancel and debit/credit adjustments (FR-017).
class InvoiceDetailScreen extends ConsumerStatefulWidget {
  const InvoiceDetailScreen({super.key, required this.invoiceId});

  final String invoiceId;

  @override
  ConsumerState<InvoiceDetailScreen> createState() =>
      _InvoiceDetailScreenState();
}

class _InvoiceDetailScreenState extends ConsumerState<InvoiceDetailScreen> {
  Invoice? _invoice;
  String? _error;
  bool _loading = true;

  bool get _isManager =>
      ref.read(authControllerProvider).user?.role == UserRole.manager;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final inv =
          await ref.read(billingRepositoryProvider).getInvoice(widget.invoiceId);
      setState(() => _invoice = inv);
    } on ApiException catch (e) {
      if (!mounted) return;
      final l10n = AppLocalizations.of(context);
      setState(() => _error = e.serverMessage ?? l10n.errorServer);
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  Future<void> _adjust() async {
    final l10n = AppLocalizations.of(context);
    final amountCtrl = TextEditingController();
    final reasonCtrl = TextEditingController();
    String kind = 'credit';
    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => StatefulBuilder(
        builder: (ctx, setState) => AlertDialog(
          title: Text(l10n.addAdjustment),
          content: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              SegmentedButton<String>(
                segments: [
                  ButtonSegment(value: 'credit', label: Text(l10n.adjustmentCredit)),
                  ButtonSegment(value: 'debit', label: Text(l10n.adjustmentDebit)),
                ],
                selected: {kind},
                onSelectionChanged: (s) => setState(() => kind = s.first),
              ),
              TextField(
                controller: amountCtrl,
                keyboardType: TextInputType.number,
                decoration: InputDecoration(labelText: l10n.costItemAmount),
              ),
              TextField(
                controller: reasonCtrl,
                decoration: InputDecoration(labelText: l10n.adjustmentReason),
              ),
            ],
          ),
          actions: [
            TextButton(
                onPressed: () => Navigator.pop(ctx, false),
                child: Text(l10n.cancel)),
            FilledButton(
                onPressed: () => Navigator.pop(ctx, true),
                child: Text(l10n.confirm)),
          ],
        ),
      ),
    );
    if (ok != true) return;
    final amount = int.tryParse(fromPersianDigits(amountCtrl.text.trim()));
    if (amount == null || amount <= 0 || reasonCtrl.text.trim().isEmpty) {
      return;
    }
    try {
      await ref.read(billingRepositoryProvider).addAdjustment(
            widget.invoiceId,
            kind: kind,
            amount: amount,
            reason: reasonCtrl.text.trim(),
          );
      await _load();
    } on ApiException catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(e.serverMessage ?? l10n.errorServer)),
        );
      }
    }
  }

  Future<void> _cancel() async {
    final l10n = AppLocalizations.of(context);
    final reasonCtrl = TextEditingController();
    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(l10n.cancelInvoice),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Text(l10n.cancelInvoiceConfirm),
            const SizedBox(height: 8),
            TextField(
              controller: reasonCtrl,
              decoration: InputDecoration(labelText: l10n.cancelReason),
            ),
          ],
        ),
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
      await ref
          .read(billingRepositoryProvider)
          .cancelInvoice(widget.invoiceId, reasonCtrl.text.trim());
      await _load();
    } on ApiException catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(e.serverMessage ?? l10n.errorServer)),
        );
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    if (_loading) {
      return Scaffold(
        appBar: AppBar(),
        body: const Center(child: CircularProgressIndicator()),
      );
    }
    if (_error != null || _invoice == null) {
      return Scaffold(
        appBar: AppBar(),
        body: Center(child: Text(_error ?? l10n.errorUnknown)),
      );
    }
    final inv = _invoice!;

    return Scaffold(
      appBar: AppBar(
        title: Text(inv.invoiceNumberPresent
            ? inv.invoiceNumber
            : l10n.invoicesTitle),
        actions: [
          if (_isManager && inv.status != 'cancelled') ...[
            IconButton(
              icon: const Icon(Icons.receipt_long),
              tooltip: l10n.addAdjustment,
              onPressed: _adjust,
            ),
            IconButton(
              icon: const Icon(Icons.cancel_outlined),
              tooltip: l10n.cancelInvoice,
              onPressed: _cancel,
            ),
          ],
        ],
      ),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          Row(
            children: [
              if (inv.unitNumber.isNotEmpty)
                Expanded(
                  child: Text(
                    '${l10n.unitNumber} ${inv.unitNumber}',
                    style: Theme.of(context).textTheme.titleMedium,
                  ),
                ),
              StatusChip(kind: StatusKind.invoice, value: inv.status),
            ],
          ),
          const SizedBox(height: 16),
          _amountRow(context, l10n.baseAmount, inv.baseAmount),
          if (inv.priorDebt > 0)
            _amountRow(context, l10n.priorDebt, inv.priorDebt),
          if (inv.lateFeeAmount > 0)
            _amountRow(context, l10n.lateFeeAmount, inv.lateFeeAmount),
          if (inv.creditAmount > 0)
            _amountRow(context, l10n.creditAmount, inv.creditAmount),
          const Divider(),
          _amountRow(context, l10n.finalAmount, inv.finalAmount,
              emphasized: true),
          if (inv.paidAmount > 0)
            _amountRow(context, l10n.paidAmount, inv.paidAmount),
          if (inv.dueDate != null)
            ListTile(
              contentPadding: EdgeInsets.zero,
              dense: true,
              title: Text(l10n.dueDateLabel),
              trailing: Text(formatJalaliLongDate(
                  DateTime.tryParse(inv.dueDate!) ?? DateTime(2000))),
            ),
          if (inv.issueDate != null)
            ListTile(
              contentPadding: EdgeInsets.zero,
              dense: true,
              title: Text(l10n.issueDateLabel),
              trailing: Text(formatJalaliLongDate(
                  DateTime.tryParse(inv.issueDate!) ?? DateTime(2000))),
            ),
          const Divider(height: 32),
          Text(l10n.costItemsTitle, style: Theme.of(context).textTheme.titleMedium),
          ...inv.items.map((item) => ListTile(
                contentPadding: EdgeInsets.zero,
                leading: Icon(_kindIcon(item.kind)),
                title: Text(item.title),
                subtitle: item.method != null
                    ? Text(calcMethodLabel(l10n, item.method!))
                    : item.kind == 'adjustment'
                        ? Text(l10n.itemKindAdjustment)
                        : null,
                trailing: MoneyText(amount: item.amount),
              )),
          if (inv.adjustments.isNotEmpty) ...[
            const Divider(height: 32),
            Text(l10n.addAdjustment,
                style: Theme.of(context).textTheme.titleMedium),
            ...inv.adjustments.map((a) => ListTile(
                  contentPadding: EdgeInsets.zero,
                  leading: Icon(a.kind == 'credit'
                      ? Icons.arrow_downward
                      : Icons.arrow_upward),
                  title: Text(a.kind == 'credit'
                      ? l10n.adjustmentCredit
                      : l10n.adjustmentDebit),
                  subtitle: Text(a.reason),
                  trailing: MoneyText(amount: a.amount),
                )),
          ],
        ],
      ),
    );
  }

  IconData _kindIcon(String kind) => switch (kind) {
        'late_fee' => Icons.hourglass_bottom,
        'adjustment' => Icons.receipt_long,
        _ => Icons.receipt_outlined,
      };

  Widget _amountRow(BuildContext context, String label, int amount,
      {bool emphasized = false}) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 4),
      child: Row(
        children: [
          Expanded(child: Text(label)),
          MoneyText(
            amount: amount,
            style: emphasized
                ? Theme.of(context)
                    .textTheme
                    .titleLarge
                    ?.copyWith(fontWeight: FontWeight.bold)
                : null,
          ),
        ],
      ),
    );
  }
}
