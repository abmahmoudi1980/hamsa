import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../core/datetime/jalali.dart';
import '../../../core/l10n/app_localizations.dart';
import '../../../core/network/api_exception.dart';
import '../../buildings/buildings_controller.dart';
import '../../buildings/models/building.dart';
import '../billing_controller.dart';
import '../models/billing.dart';
import '../../../shared/widgets/calc_method.dart';
import 'period_list_screen.dart' show costItemPayload;

/// Cost-item editor (US4/T051): title, amount (Persian digits), one of the
/// six methods — including the combined-weights editor and the unit picker
/// for specific_units. Draft periods only (server 409s otherwise).
class CostItemFormScreen extends ConsumerStatefulWidget {
  const CostItemFormScreen({
    super.key,
    required this.buildingId,
    required this.periodId,
    this.existing,
  });

  final String buildingId;
  final String periodId;
  final CostItem? existing;

  @override
  ConsumerState<CostItemFormScreen> createState() =>
      _CostItemFormScreenState();
}

class _CostItemFormScreenState extends ConsumerState<CostItemFormScreen> {
  final _formKey = GlobalKey<FormState>();
  late final _title = TextEditingController(text: widget.existing?.title ?? '');
  late final _amount = TextEditingController(
    text: widget.existing == null ? '' : '${widget.existing!.totalAmount}',
  );
  late final _fixedPerUnit = TextEditingController(
    text: widget.existing?.fixedAmountPerUnit == null
        ? ''
        : '${widget.existing!.fixedAmountPerUnit}',
  );
  late String _method = widget.existing?.method ?? 'equal';
  late bool _includeVacant = widget.existing?.includeVacant ?? false;
  late List<ComboWeight> _weights =
      widget.existing?.comboWeights.toList() ?? const [];
  late final List<String> _unitIds =
      widget.existing?.unitIds.toList() ?? const [];
  String? _apiError;
  bool _saving = false;

  int _weightSum() => _weights.fold(0, (s, w) => s + w.weight);

  Future<void> _save() async {
    if (!_formKey.currentState!.validate()) return;
    if (_method == 'combined' && _weightSum() != 100) {
      setState(
          () => _apiError = AppLocalizations.of(context).comboWeightsInvalid);
      return;
    }
    setState(() {
      _saving = true;
      _apiError = null;
    });
    final amount =
        int.tryParse(fromPersianDigits(_amount.text.trim())) ?? 0;
    final fixed =
        int.tryParse(fromPersianDigits(_fixedPerUnit.text.trim())) ?? 0;
    try {
      await ref
          .read(costItemsControllerProvider(widget.periodId).notifier)
          .save(
            widget.existing,
            costItemPayload(
              title: _title.text.trim(),
              method: _method,
              totalAmount: _method == 'fixed' ? null : amount,
              fixedAmountPerUnit: _method == 'fixed' ? fixed : null,
              comboWeights: _method == 'combined' ? _weights : null,
              includeVacant: _includeVacant,
              unitIds: _unitIds,
            ),
          );
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
    _amount.dispose();
    _fixedPerUnit.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final unitsAsync = ref.watch(unitsControllerProvider(widget.buildingId));
    final showAmount = _method != 'fixed';
    final showFixed = _method == 'fixed';
    final showUnits = _method == 'specific_units';
    final showWeights = _method == 'combined';

    return Scaffold(
      appBar: AppBar(
        title: Text(widget.existing == null
            ? l10n.addCostItem
            : l10n.editCostItem),
      ),
      body: Form(
        key: _formKey,
        child: ListView(
          padding: const EdgeInsets.all(16),
          children: [
            TextFormField(
              controller: _title,
              decoration: InputDecoration(labelText: l10n.costItemTitle),
              validator: (v) =>
                  (v == null || v.trim().isEmpty) ? l10n.requiredField : null,
            ),
            const SizedBox(height: 12),
            if (showAmount)
              TextFormField(
                controller: _amount,
                decoration:
                    InputDecoration(labelText: l10n.costItemAmount),
                keyboardType: TextInputType.number,
                validator: (v) {
                  final n = int.tryParse(fromPersianDigits(v ?? ''));
                  if (n == null || n <= 0) return l10n.invalidAmount;
                  return null;
                },
              ),
            if (showFixed) ...[
              const SizedBox(height: 12),
              TextFormField(
                controller: _fixedPerUnit,
                decoration: InputDecoration(labelText: l10n.fixedPerUnit),
                keyboardType: TextInputType.number,
                validator: (v) {
                  final n = int.tryParse(fromPersianDigits(v ?? ''));
                  if (n == null || n <= 0) return l10n.invalidAmount;
                  return null;
                },
              ),
            ],
            const SizedBox(height: 12),
            DropdownButtonFormField<String>(
              initialValue: _method,
              decoration: InputDecoration(labelText: l10n.calcMethod),
              items: calcMethods
                  .map((m) => DropdownMenuItem(
                        value: m,
                        child: Text(calcMethodLabel(l10n, m)),
                      ))
                  .toList(),
              onChanged: (v) => setState(() => _method = v ?? 'equal'),
            ),
            if (_method != 'specific_units') ...[
              const SizedBox(height: 8),
              SwitchListTile(
                title: Text(l10n.includeVacant),
                value: _includeVacant,
                onChanged: (v) => setState(() => _includeVacant = v),
                contentPadding: EdgeInsets.zero,
              ),
            ],
            if (showUnits) ...[
              const SizedBox(height: 8),
              Text(l10n.selectUnits,
                  style: Theme.of(context).textTheme.labelLarge),
              unitsAsync.when(
                loading: () => const Padding(
                  padding: EdgeInsets.all(16),
                  child: Center(child: CircularProgressIndicator()),
                ),
                error: (e, _) => Text(l10n.errorServer),
                data: (state) => Column(
                  children: state.units
                      .map<Widget>((Unit u) => CheckboxListTile(
                            title: Text('${l10n.unitNumber} ${u.number}'),
                            value: _unitIds.contains(u.id),
                            onChanged: (on) => setState(() {
                              on == true
                                  ? _unitIds.add(u.id)
                                  : _unitIds.remove(u.id);
                            }),
                            contentPadding: EdgeInsets.zero,
                          ))
                      .toList(),
                ),
              ),
            ],
            if (showWeights) ...[
              const SizedBox(height: 8),
              Row(
                mainAxisAlignment: MainAxisAlignment.spaceBetween,
                children: [
                  Text(l10n.comboWeights,
                      style: Theme.of(context).textTheme.labelLarge),
                  Text('${toPersianDigits('$_weightSum')} / ۱۰۰'),
                ],
              ),
              ..._weights.asMap().entries.map((entry) {
                final i = entry.key;
                final w = entry.value;
                return Row(
                  children: [
                    Expanded(
                      child: DropdownButtonFormField<String>(
                        initialValue: w.method,
                        items: calcMethods
                            .where((m) => m != 'combined')
                            .map((m) => DropdownMenuItem(
                                  value: m,
                                  child: Text(calcMethodLabel(l10n, m)),
                                ))
                            .toList(),
                        onChanged: (v) => setState(() =>
                            _weights[i] = ComboWeight(
                                method: v ?? 'equal',
                                weight: _weights[i].weight)),
                      ),
                    ),
                    const SizedBox(width: 8),
                    SizedBox(
                      width: 88,
                      child: TextFormField(
                        initialValue: '${w.weight}',
                        keyboardType: TextInputType.number,
                        decoration:
                            const InputDecoration(hintText: '۰–۱۰۰'),
                        onChanged: (v) => setState(() {
                          _weights[i] = ComboWeight(
                            method: _weights[i].method,
                            weight:
                                int.tryParse(fromPersianDigits(v)) ?? 0,
                          );
                        }),
                      ),
                    ),
                    IconButton(
                      icon: const Icon(Icons.remove_circle_outline),
                      onPressed: () =>
                          setState(() => _weights.removeAt(i)),
                    ),
                  ],
                );
              }),
              TextButton.icon(
                onPressed: () => setState(() => _weights =
                    [..._weights, ComboWeight(method: 'equal', weight: 0)]),
                icon: const Icon(Icons.add),
                label: Text(l10n.addWeight),
              ),
            ],
            if (_apiError != null) ...[
              const SizedBox(height: 12),
              Text(_apiError!,
                  style:
                      TextStyle(color: Theme.of(context).colorScheme.error)),
            ],
            const SizedBox(height: 24),
            FilledButton(
              onPressed: _saving ? null : _save,
              child: Text(l10n.save),
            ),
          ],
        ),
      ),
    );
  }
}

