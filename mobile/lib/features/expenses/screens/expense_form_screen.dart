import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:image_picker/image_picker.dart';

import '../../../core/datetime/jalali.dart';
import '../../../core/l10n/app_localizations.dart';
import '../../../core/theme/app_theme.dart';
import '../../../core/network/api_exception.dart';
import '../../../shared/widgets/status_labels.dart';
import '../../../shared/validation/validators.dart';
import '../../../shared/widgets/confirm_dialog.dart';
import '../../../shared/widgets/attachment_picker.dart';
import '../expense_controller.dart';

/// US6/T066 — create/edit expense (FR-026): title, Persian-labeled category,
/// amount, Jalali date via the T018 picker, optional description and receipt
/// attachment. Editing also drives the approval workflow (pending →
/// approved/rejected).
class ExpenseFormScreen extends ConsumerStatefulWidget {
  const ExpenseFormScreen({
    super.key,
    required this.buildingId,
    this.expenseId,
  });

  final String buildingId;

  /// Present → edit mode; null → create mode.
  final String? expenseId;

  @override
  ConsumerState<ExpenseFormScreen> createState() => _ExpenseFormScreenState();
}

class _ExpenseFormScreenState extends ConsumerState<ExpenseFormScreen> {
  final _formKey = GlobalKey<FormState>();
  final _titleCtrl = TextEditingController();
  final _amountCtrl = TextEditingController();
  final _descriptionCtrl = TextEditingController();
  String _category = 'other';
  String _approval = 'pending';
  DateTime? _expenseDate;
  XFile? _receipt;
  bool _loading = false;
  bool _saving = false;

  @override
  void initState() {
    super.initState();
    if (widget.expenseId != null) _loadExisting();
  }

  @override
  void dispose() {
    _titleCtrl.dispose();
    _amountCtrl.dispose();
    _descriptionCtrl.dispose();
    super.dispose();
  }

  Future<void> _loadExisting() async {
    final l10n = AppLocalizations.of(context);
    setState(() => _loading = true);
    try {
      final e =
          await ref.read(expenseRepositoryProvider).getExpense(widget.expenseId!);
      if (!mounted) return;
      setState(() {
        _titleCtrl.text = e.title;
        _amountCtrl.text = '${e.amount}';
        _category = e.category;
        _approval = e.approvalStatus;
        _expenseDate = DateTime.parse(e.expenseDate);
        _descriptionCtrl.text = e.description ?? '';
      });
    } catch (_) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(l10n.errorServer)),
      );
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  Future<void> _submit() async {
    final l10n = AppLocalizations.of(context);
    if (!_formKey.currentState!.validate() || _expenseDate == null) return;
    final amount = int.tryParse(fromPersianDigits(_amountCtrl.text.trim()));
    if (amount == null || amount <= 0) return;
    setState(() => _saving = true);
    try {
      final repo = ref.read(expenseRepositoryProvider);
      if (widget.expenseId == null) {
        final receiptId = _receipt == null
            ? null
            : await repo.uploadReceipt(
                path: _receipt!.path,
                name: _receipt!.name,
              );
        await repo.createExpense(
          widget.buildingId,
          title: _titleCtrl.text.trim(),
          category: _category,
          amount: amount,
          expenseDate: isoDate(_expenseDate!),
          description: _descriptionCtrl.text.trim(),
          receiptFileId: receiptId,
          approvalStatus: _approval,
        );
      } else {
        await repo.updateExpense(
          widget.expenseId!,
          title: _titleCtrl.text.trim(),
          category: _category,
          amount: amount,
          expenseDate: isoDate(_expenseDate!),
          description: _descriptionCtrl.text.trim(),
          approvalStatus: _approval,
        );
      }
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(l10n.expenseSavedOk)),
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

  Future<void> _delete() async {
    final l10n = AppLocalizations.of(context);
    final confirmed = await ConfirmDialog.show(
      context,
      title: l10n.deleteExpense,
      message: l10n.deleteExpenseConfirm,
      confirmLabel: l10n.delete,
      destructive: true,
    );
    if (confirmed != true) return;
    setState(() => _saving = true);
    try {
      await ref.read(expenseRepositoryProvider).deleteExpense(widget.expenseId!);
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(l10n.expenseDeletedOk)),
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
    if (_loading) {
      return Scaffold(
        appBar: AppBar(title: Text(l10n.editExpense)),
        body: const Center(child: CircularProgressIndicator()),
      );
    }
    return Scaffold(
      appBar: AppBar(
        title: Text(
            widget.expenseId == null ? l10n.addExpense : l10n.editExpense),
        actions: [
          if (widget.expenseId != null)
            IconButton(
              icon: const Icon(Icons.delete_outline),
              tooltip: l10n.deleteExpense,
              onPressed: _saving ? null : _delete,
            ),
        ],
      ),
      body: Form(
        key: _formKey,
        child: ListView(
          padding: AppTheme.pagePadding,
          children: [
            TextFormField(
              controller: _titleCtrl,
              decoration: InputDecoration(labelText: l10n.expenseTitleLabel),
              validator: (v) =>
                  (v == null || v.trim().isEmpty) ? l10n.requiredField : null,
            ),
            const SizedBox(height: AppTheme.spaceM),
            DropdownButtonFormField<String>(
              initialValue: _category,
              decoration:
                  InputDecoration(labelText: l10n.expenseCategoryLabel),
              items: expenseCategoryLabels.entries
                  .map((e) =>
                      DropdownMenuItem(value: e.key, child: Text(e.value)))
                  .toList(),
              onChanged: (v) => setState(() => _category = v ?? 'other'),
            ),
            const SizedBox(height: AppTheme.spaceM),
            TextFormField(
              controller: _amountCtrl,
              keyboardType: TextInputType.number,
              decoration: InputDecoration(labelText: l10n.expenseAmountLabel),
              validator: (v) => Validators.positiveAmount(l10n, v),
            ),
            const SizedBox(height: AppTheme.spaceM),
            JalaliDatePickerField(
              initialValue: _expenseDate,
              label: l10n.expenseDateLabel,
              validator: (_) =>
                  _expenseDate == null ? l10n.requiredField : null,
              onChanged: (d) => setState(() => _expenseDate = d),
            ),
            const SizedBox(height: AppTheme.spaceM),
            DropdownButtonFormField<String>(
              initialValue: _approval,
              decoration:
                  InputDecoration(labelText: l10n.expenseApprovalLabel),
              items: expenseApprovalLabels.entries
                  .map((e) =>
                      DropdownMenuItem(value: e.key, child: Text(e.value)))
                  .toList(),
              onChanged: (v) => setState(() => _approval = v ?? 'pending'),
            ),
            const SizedBox(height: AppTheme.spaceM),
            TextFormField(
              controller: _descriptionCtrl,
              maxLines: 3,
              decoration:
                  InputDecoration(labelText: l10n.expenseDescriptionLabel),
            ),
            const SizedBox(height: AppTheme.spaceM),
            // Receipt attach (create mode): uploads through POST /files on save.
            if (widget.expenseId == null)
              AttachmentPicker(
                onChanged: (f) => setState(() => _receipt = f),
              ),
          ],
        ),
      ),
      bottomNavigationBar: SafeArea(
        child: Padding(
          padding: const EdgeInsets.symmetric(
            horizontal: AppTheme.spaceXl,
            vertical: AppTheme.spaceM,
          ),
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
      ),
    );
  }
}
