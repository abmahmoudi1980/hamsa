import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/datetime/jalali.dart';
import '../../../core/l10n/app_localizations.dart';
import '../../../core/network/api_exception.dart';
import '../../../core/platform/app_url_launcher.dart';
import '../../../core/theme/app_theme.dart';
import '../../../shared/formatters/money_text.dart';
import '../../../shared/widgets/confirm_dialog.dart';
import '../../../shared/widgets/empty_state.dart';
import '../../../shared/widgets/menu_card.dart';
import '../../../shared/widgets/status_chip.dart';
import '../../billing/billing_controller.dart';
import '../../billing/models/billing.dart';
import '../payment_controller.dart';

/// US5/T061 — resident online payment flow: shows the invoice's outstanding,
/// starts the gateway payment, opens the gateway page in the browser, and on
/// return refreshes the invoice plus the payment history to show the receipt
/// (verified status + tracking number).
class PayInvoiceScreen extends ConsumerStatefulWidget {
  const PayInvoiceScreen({super.key, required this.invoiceId});

  final String invoiceId;

  @override
  ConsumerState<PayInvoiceScreen> createState() => _PayInvoiceScreenState();
}

class _PayInvoiceScreenState extends ConsumerState<PayInvoiceScreen>
    with WidgetsBindingObserver {
  Invoice? _invoice;
  bool _loading = true;
  bool _starting = false;
  String? _error;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);
    _load();
  }

  @override
  void dispose() {
    WidgetsBinding.instance.removeObserver(this);
    super.dispose();
  }

  /// The gateway page runs in the browser; returning to the app is the
  /// signal to re-check the payment status (verified server-side by the
  /// callback against the DB amount).
  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    if (state == AppLifecycleState.resumed) _load();
  }

  Future<void> _load() async {
    final l10n = AppLocalizations.of(context);
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final inv =
          await ref.read(billingRepositoryProvider).getInvoice(widget.invoiceId);
      if (!mounted) return;
      setState(() => _invoice = inv);
    } on ApiException catch (e) {
      if (!mounted) return;
      setState(() => _error = e.serverMessage ?? l10n.errorServer);
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  int get _outstanding {
    final inv = _invoice;
    if (inv == null) return 0;
    final remaining = inv.finalAmount - inv.paidAmount;
    return remaining > 0 ? remaining : 0;
  }

  Future<void> _pay() async {
    final l10n = AppLocalizations.of(context);
    final confirmed = await ConfirmDialog.show(
      context,
      title: l10n.payTitle,
      message: l10n.payLaunchConfirm,
    );
    if (confirmed != true) return;
    setState(() => _starting = true);
    try {
      final start = await ref
          .read(paymentRepositoryProvider)
          .startGatewayPayment(widget.invoiceId);
      await AppUrlLauncher.launch(start.paymentUrl);
    } on ApiException catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(e.serverMessage ?? l10n.errorServer)),
        );
      }
    } on PlatformException {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(l10n.launchFailed)),
        );
      }
    } finally {
      if (mounted) setState(() => _starting = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return Scaffold(
      appBar: AppBar(title: Text(l10n.payTitle)),
      body: _loading && _invoice == null
          ? const Center(child: CircularProgressIndicator())
          : _error != null && _invoice == null
              ? EmptyState(icon: Icons.error_outline, title: _error!)
              : RefreshIndicator(
                  onRefresh: _load,
                  child: ListView(
                    padding: AppTheme.pagePadding,
                    children: [
                      if (_invoice != null) ...[
                        Card(
                          child: Padding(
                            padding: const EdgeInsets.all(AppTheme.spaceL),
                            child: Column(
                              crossAxisAlignment: CrossAxisAlignment.stretch,
                              children: [
                                Row(
                                  children: [
                                    Text(_invoice!.invoiceNumber,
                                        style: Theme.of(context)
                                            .textTheme
                                            .titleMedium),
                                    const Spacer(),
                                    StatusChip(
                                        kind: StatusKind.invoice,
                                        value: _invoice!.status),
                                  ],
                                ),
                                const SizedBox(height: AppTheme.spaceM),
                                Text(l10n.outstandingLabel,
                                    style: Theme.of(context)
                                        .textTheme
                                        .bodySmall),
                                MoneyText(
                                  amount: _outstanding,
                                  style: Theme.of(context)
                                      .textTheme
                                      .titleLarge
                                      ?.copyWith(
                                          color: Theme.of(context)
                                              .colorScheme
                                              .primary),
                                ),
                                const SizedBox(height: AppTheme.spaceM),
                                if (_outstanding > 0 &&
                                    _invoice!.status != 'cancelled')
                                  FilledButton(
                                    onPressed: _starting ? null : _pay,
                                    child: _starting
                                        ? const SizedBox(
                                            width: 18,
                                            height: 18,
                                            child: CircularProgressIndicator(
                                                strokeWidth: 2))
                                        : Text(l10n.payNow),
                                  ),
                              ],
                            ),
                          ),
                        ),
                        const SizedBox(height: AppTheme.spaceL),
                        SectionHeader(title: l10n.paymentHistoryTitle),
                        _InvoicePayments(invoiceId: widget.invoiceId),
                      ],
                    ],
                  ),
                ),
    );
  }
}
/// Receipt rows: this invoice's payments (newest first) from the resident
/// history — the verified row carries the gateway tracking number.
class _InvoicePayments extends ConsumerWidget {
  const _InvoicePayments({required this.invoiceId});

  final String invoiceId;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final payments = ref.watch(myPaymentsControllerProvider);
    return payments.when(
      loading: () => const Center(child: CircularProgressIndicator()),
      error: (e, _) => Center(child: Text(l10n.errorServer)),
      data: (all) {
        final mine = all
            .where((p) => p.invoiceId == invoiceId)
            .toList(growable: false);
        if (mine.isEmpty) return EmptyState(title: l10n.noPayments);
        return Column(
          children: mine.map((p) {
            final verified = p.status == 'verified';
            return Card(
              child: ListTile(
                leading:
                    Icon(verified ? Icons.check_circle : Icons.hourglass_top),
                title: Row(
                  children: [
                    MoneyText(amount: p.amount),
                    const Spacer(),
                    StatusChip(kind: StatusKind.payment, value: p.status),
                  ],
                ),
                subtitle: p.trackingNumber == null
                    ? null
                    : Text(toPersianDigits(p.trackingNumber!)),
              ),
            );
          }).toList(),
        );
      },
    );
  }
}

