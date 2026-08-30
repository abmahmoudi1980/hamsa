import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../core/datetime/jalali.dart';
import '../../../core/l10n/app_localizations.dart';
import '../../../core/theme/app_theme.dart';
import '../../../core/network/api_exception.dart';
import '../../../shared/validation/validators.dart';
import '../../../shared/widgets/save_bar.dart';
import '../payment_controller.dart';

/// US5/T060 — manager records a manual payment against an issued invoice
/// (FR-022): amount, Jalali paid_at via the T018 picker, optional tracking
/// number. Surplus beyond the outstanding becomes unit credit server-side.
class RecordPaymentScreen extends ConsumerStatefulWidget {
  const RecordPaymentScreen({super.key, required this.invoiceId});

  final String invoiceId;

  @override
  ConsumerState<RecordPaymentScreen> createState() =>
      _RecordPaymentScreenState();
}

class _RecordPaymentScreenState extends ConsumerState<RecordPaymentScreen> {
  final _formKey = GlobalKey<FormState>();
  final _amountCtrl = TextEditingController();
  final _trackingCtrl = TextEditingController();
  DateTime? _paidAt;
  bool _saving = false;

  @override
  void dispose() {
    _amountCtrl.dispose();
    _trackingCtrl.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    final l10n = AppLocalizations.of(context);
    if (!_formKey.currentState!.validate() || _paidAt == null) return;
    final amount = int.tryParse(fromPersianDigits(_amountCtrl.text.trim()));
    if (amount == null || amount <= 0) return;
    setState(() => _saving = true);
    try {
      await ref.read(paymentRepositoryProvider).recordManualPayment(
            widget.invoiceId,
            amount: amount,
            paidAt: isoDate(_paidAt!),
            trackingNumber: _trackingCtrl.text.trim(),
          );
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(l10n.paymentRecordedOk)),
      );
      context.pop();
    } on ApiException catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(e.serverMessage ?? l10n.errorServer)),
      );
    } finally {
      if (mounted) setState(() => _saving = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return Scaffold(
      appBar: AppBar(title: Text(l10n.recordPayment)),
      body: Form(
        key: _formKey,
        child: ListView(
          padding: AppTheme.pagePadding,
          children: [
            TextFormField(
              controller: _amountCtrl,
              keyboardType: TextInputType.number,
              decoration: InputDecoration(labelText: l10n.paymentAmountLabel),
              validator: (v) => Validators.positiveAmount(l10n, v),
            ),
            const SizedBox(height: AppTheme.spaceM),
            JalaliDatePickerField(
              initialValue: _paidAt,
              label: l10n.paymentDateLabel,
              validator: (_) => _paidAt == null ? l10n.requiredField : null,
              onChanged: (d) => setState(() => _paidAt = d),
            ),
            const SizedBox(height: AppTheme.spaceM),
            TextFormField(
              controller: _trackingCtrl,
              decoration:
                  InputDecoration(labelText: l10n.trackingNumberLabel),
            ),
          ],
        ),
      ),
      bottomNavigationBar: SaveBar(
        child: FilledButton(
          onPressed: _saving ? null : _submit,
          child: _saving
              ? const SizedBox(
                  width: 18,
                  height: 18,
                  child: CircularProgressIndicator(strokeWidth: 2),
                )
              : Text(l10n.confirm),
        ),
      ),
    );
  }
}
