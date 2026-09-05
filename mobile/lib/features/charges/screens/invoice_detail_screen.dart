import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import '../../../core/datetime/jalali.dart';
import '../../../core/l10n/app_localizations.dart';
import '../../../core/network/api_exception.dart';
import '../../../core/theme/app_theme.dart';
import '../../../shared/formatters/money_text.dart';
import '../../../shared/formatters/toman_input.dart';
import '../../../shared/widgets/calc_method.dart';
import '../../../shared/widgets/confirm_dialog.dart';
import '../../../shared/widgets/empty_state.dart';
import '../../../shared/widgets/menu_card.dart';
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
      final inv = await ref
          .read(billingRepositoryProvider)
          .getInvoice(widget.invoiceId);
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
        builder: (ctx, setState) => ConfirmDialog(
          title: l10n.addAdjustment,
          content: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              SegmentedButton<String>(
                segments: [
                  ButtonSegment(
                    value: 'credit',
                    label: Text(l10n.adjustmentCredit),
                  ),
                  ButtonSegment(
                    value: 'debit',
                    label: Text(l10n.adjustmentDebit),
                  ),
                ],
                selected: {kind},
                onSelectionChanged: (s) => setState(() => kind = s.first),
              ),
              TextField(
                controller: amountCtrl,
                keyboardType: TextInputType.number,
                textDirection: TextDirection.ltr,
                inputFormatters: const [TomanInputFormatter()],
              ),
              TextField(
                controller: reasonCtrl,
                decoration: InputDecoration(labelText: l10n.adjustmentReason),
              ),
            ],
          ),
        ),
      ),
    );
    if (ok != true) return;
    final amount = parseToman(amountCtrl.text);
    if (amount == null || amount <= 0 || reasonCtrl.text.trim().isEmpty) {
      return;
    }
    try {
      await ref
          .read(billingRepositoryProvider)
          .addAdjustment(
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
    final ok = await ConfirmDialog.show(
      context,
      title: l10n.cancelInvoice,
      message: l10n.cancelInvoiceConfirm,
      content: TextField(
        controller: reasonCtrl,
        decoration: InputDecoration(labelText: l10n.cancelReason),
      ),
      destructive: true,
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
        body: const SafeArea(
          child: Center(child: CircularProgressIndicator())
        ),
      );
    }
    if (_error != null || _invoice == null) {
      return Scaffold(
        appBar: AppBar(),
        body: SafeArea(
          child: EmptyState(
            icon: Icons.error_outline,
            title: _error ?? l10n.errorUnknown,
          )
        ),
      );
    }
    final inv = _invoice!;

    return Scaffold(
      appBar: AppBar(
        title: Text(
          inv.invoiceNumberPresent ? inv.invoiceNumber : l10n.invoicesTitle,
        ),
        actions: [
          // US5 (T060): manager records manual payments here.
          if (_isManager && _payable(inv))
            IconButton(
              icon: const Icon(Icons.payments_outlined),
              tooltip: l10n.recordPayment,
              onPressed: () async {
                await context.push('/manager/records-payment/${inv.id}');
                await _load();
              },
            ),
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
      body: SafeArea(
        child: ListView(
          padding: AppTheme.pagePadding,
          children: [
            Card(
              child: Padding(
                padding: const EdgeInsets.all(AppTheme.spaceL),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.stretch,
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
                    const SizedBox(height: AppTheme.spaceM),
                    _amountRow(context, l10n.baseAmount, inv.baseAmount),
                    if (inv.priorDebt > 0)
                      _amountRow(context, l10n.priorDebt, inv.priorDebt),
                    if (inv.lateFeeAmount > 0)
                      _amountRow(context, l10n.lateFeeAmount, inv.lateFeeAmount),
                    if (inv.creditAmount > 0)
                      _amountRow(context, l10n.creditAmount, inv.creditAmount),
                    const Divider(),
                    _amountRow(
                      context,
                      l10n.finalAmount,
                      inv.finalAmount,
                      emphasized: true,
                    ),
                    if (inv.paidAmount > 0)
                      _amountRow(context, l10n.paidAmount, inv.paidAmount),
                    if (inv.dueDate != null)
                      _metaRow(
                        context,
                        l10n.dueDateLabel,
                        formatJalaliLongDate(
                          DateTime.tryParse(inv.dueDate!) ?? DateTime(2000),
                        ),
                      ),
                    if (inv.issueDate != null)
                      _metaRow(
                        context,
                        l10n.issueDateLabel,
                        formatJalaliLongDate(
                          DateTime.tryParse(inv.issueDate!) ?? DateTime(2000),
                        ),
                      ),
                    // US5 (T061): resident online payment entry.
                    if (!_isManager && _payable(inv)) ...[
                      const SizedBox(height: AppTheme.spaceL),
                      SizedBox(
                        width: double.infinity,
                        child: FilledButton.icon(
                          icon: const Icon(Icons.credit_card),
                          label: Text(l10n.payNow),
                          onPressed: () async {
                            await context.push('/invoice/${inv.id}/pay');
                            await _load();
                          },
                        ),
                      ),
                    ],
                  ],
                ),
              ),
            ),
            if (inv.items.isNotEmpty) ...[
              const SizedBox(height: AppTheme.spaceXl),
              SectionHeader(title: l10n.costItemsTitle),
              Card(
                child: Column(
                  children: inv.items
                      .map(
                        (item) => ListTile(
                          leading: Icon(_kindIcon(item.kind)),
                          title: Text(item.title),
                          subtitle: item.method != null
                              ? Text(calcMethodLabel(l10n, item.method!))
                              : item.kind == 'adjustment'
                              ? Text(l10n.itemKindAdjustment)
                              : null,
                          trailing: MoneyText(amount: item.amount),
                        ),
                      )
                      .toList(),
                ),
              ),
            ],
            if (inv.adjustments.isNotEmpty) ...[
              const SizedBox(height: AppTheme.spaceXl),
              SectionHeader(title: l10n.addAdjustment),
              Card(
                child: Column(
                  children: inv.adjustments
                      .map(
                        (a) => ListTile(
                          leading: Icon(
                            a.kind == 'credit'
                                ? Icons.arrow_downward
                                : Icons.arrow_upward,
                          ),
                          title: Text(
                            a.kind == 'credit'
                                ? l10n.adjustmentCredit
                                : l10n.adjustmentDebit,
                          ),
                          subtitle: Text(a.reason),
                          trailing: MoneyText(amount: a.amount),
                        ),
                      )
                      .toList(),
                ),
              ),
            ],
          ],
        )
      ),
    );
  }

  /// Issued and still carrying an outstanding amount — the only invoices a
  /// payment can target (server rejects the rest with a Persian 409).
  bool _payable(Invoice inv) =>
      inv.invoiceNumberPresent &&
      (inv.status == 'unpaid' || inv.status == 'partial');

  IconData _kindIcon(String kind) => switch (kind) {
    'late_fee' => Icons.hourglass_bottom,
    'adjustment' => Icons.receipt_long,
    _ => Icons.receipt_outlined,
  };

  Widget _amountRow(
    BuildContext context,
    String label,
    int amount, {
    bool emphasized = false,
  }) {
    final theme = Theme.of(context);
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: AppTheme.spaceXs),
      child: Row(
        children: [
          Expanded(child: Text(label, style: theme.textTheme.bodyMedium)),
          MoneyText(
            amount: amount,
            style: emphasized
                ? theme.textTheme.titleLarge?.copyWith(
                    color: theme.colorScheme.primary,
                  )
                : null,
          ),
        ],
      ),
    );
  }

  /// Label/value caption row for invoice metadata (issue/due dates).
  Widget _metaRow(BuildContext context, String label, String value) {
    final theme = Theme.of(context);
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: AppTheme.spaceXs),
      child: Row(
        children: [
          Expanded(child: Text(label, style: theme.textTheme.bodyMedium)),
          Text(value, style: theme.textTheme.bodyMedium),
        ],
      ),
    );
  }
}
