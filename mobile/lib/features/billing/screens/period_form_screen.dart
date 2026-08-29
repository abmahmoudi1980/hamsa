import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../core/datetime/jalali.dart';
import '../../../core/l10n/app_localizations.dart';
import '../../../core/network/api_exception.dart';
import '../../../core/theme/app_theme.dart';
import '../billing_controller.dart';
import '../models/billing.dart';

/// Create/edit billing period (US4/T050). Dates go through the Jalali picker
/// (T018 — the only date input in the app) and travel ISO-8601 on the wire.
class PeriodFormScreen extends ConsumerStatefulWidget {
  const PeriodFormScreen({super.key, required this.buildingId, this.existing});

  final String buildingId;
  final BillingPeriod? existing;

  @override
  ConsumerState<PeriodFormScreen> createState() => _PeriodFormScreenState();
}

class _PeriodFormScreenState extends ConsumerState<PeriodFormScreen> {
  final _formKey = GlobalKey<FormState>();
  late final _title = TextEditingController(text: widget.existing?.title ?? '');
  DateTime? _start;
  DateTime? _end;
  DateTime? _due;
  String _lateFeeType = 'none';
  late final _lateFeeValue = TextEditingController(
    text: widget.existing == null
        ? ''
        : _trimNum(widget.existing!.lateFeeValue),
  );
  String? _apiError;
  bool _saving = false;

  static String _trimNum(double v) =>
      v == v.roundToDouble() ? v.toInt().toString() : '$v';

  @override
  void initState() {
    super.initState();
    final e = widget.existing;
    if (e != null) {
      _start = DateTime.tryParse(e.startDate);
      _end = DateTime.tryParse(e.endDate);
      _due = DateTime.tryParse(e.dueDate);
      _lateFeeType = e.lateFeeType;
    }
  }

  Future<void> _save() async {
    if (!_formKey.currentState!.validate()) return;
    if (_start == null || _end == null || _due == null) {
      setState(() => _apiError = AppLocalizations.of(context).selectDate);
      return;
    }
    final payload = BillingPeriod(
      id: widget.existing?.id ?? '',
      buildingId: widget.buildingId,
      title: _title.text.trim(),
      startDate: isoDate(_start!),
      endDate: isoDate(_end!),
      dueDate: isoDate(_due!),
      lateFeeType: _lateFeeType,
      lateFeeValue:
          double.tryParse(fromPersianDigits(_lateFeeValue.text.trim())) ?? 0,
      status: widget.existing?.status ?? 'draft',
    ).toPayload();
    try {
      await ref
          .read(periodsControllerProvider(widget.buildingId).notifier)
          .save(widget.existing, payload);
      if (mounted) context.pop();
    } on ApiException catch (e) {
      setState(() => _apiError = e.serverMessage);
    } finally {
      if (mounted) setState(() => _saving = false);
    }
  }

  @override
  void dispose() {
    _title.dispose();
    _lateFeeValue.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final feeTypes = {
      'none': l10n.lateFeeNone,
      'fixed': l10n.lateFeeFixed,
      'percent': l10n.lateFeePercent,
      'per_day': l10n.lateFeePerDay,
    };
    return Scaffold(
      appBar: AppBar(
        title: Text(widget.existing == null ? l10n.addPeriod : l10n.editPeriod),
      ),
      body: Form(
        key: _formKey,
        child: ListView(
          padding: AppTheme.pagePadding,
          children: [
            TextFormField(
              controller: _title,
              decoration:
                  InputDecoration(labelText: l10n.periodTitleLabel),
              validator: (v) =>
                  (v == null || v.trim().isEmpty) ? l10n.requiredField : null,
            ),
            const SizedBox(height: AppTheme.spaceM),
            JalaliDatePickerField(
              label: l10n.periodStart,
              initialValue: _start,
              onChanged: (d) {
                setState(() {
                  _start = d;
                  if (_end != null && _end!.isBefore(d)) _end = d;
                  if (_due != null && _due!.isBefore(d)) _due = d;
                });
              },
              validator: (_) => null,
            ),
            const SizedBox(height: AppTheme.spaceM),
            JalaliDatePickerField(
              label: l10n.periodEnd,
              initialValue: _end,
              firstDate: _start,
              onChanged: (d) => setState(() {
                _end = d;
                if (_due != null && _due!.isBefore(d)) _due = d;
              }),
            ),
            const SizedBox(height: AppTheme.spaceM),
            JalaliDatePickerField(
              label: l10n.periodDue,
              initialValue: _due,
              firstDate: _end,
              onChanged: (d) => setState(() => _due = d),
            ),
            const SizedBox(height: AppTheme.spaceM),
            DropdownButtonFormField<String>(
              initialValue: _lateFeeType,
              decoration: InputDecoration(labelText: l10n.lateFeeType),
              items: feeTypes.entries
                  .map((e) =>
                      DropdownMenuItem(value: e.key, child: Text(e.value)))
                  .toList(),
              onChanged: (v) => setState(() => _lateFeeType = v ?? 'none'),
            ),
            if (_lateFeeType != 'none') ...[
              const SizedBox(height: AppTheme.spaceM),
              TextFormField(
                controller: _lateFeeValue,
                decoration:
                    InputDecoration(labelText: l10n.lateFeeValue),
                keyboardType: TextInputType.number,
                validator: (v) {
                  final n = double.tryParse(fromPersianDigits(v ?? ''));
                  if (n == null || n <= 0) return l10n.invalidAmount;
                  return null;
                },
              ),
            ],
            if (_apiError != null) ...[
              const SizedBox(height: AppTheme.spaceM),
              Text(
                _apiError!,
                style: TextStyle(
                    color: Theme.of(context).colorScheme.error),
                textAlign: TextAlign.center,
              ),
            ],
          ],
        ),
      ),
      bottomNavigationBar: SafeArea(
        child: Padding(
          padding: const EdgeInsets.fromLTRB(
            AppTheme.spaceXl,
            AppTheme.spaceS,
            AppTheme.spaceXl,
            0,
          ),
          child: SizedBox(
            width: double.infinity,
            child: FilledButton(
              onPressed: _saving ? null : _save,
              child: Text(l10n.save),
            ),
          ),
        ),
      ),
    );
  }
}
