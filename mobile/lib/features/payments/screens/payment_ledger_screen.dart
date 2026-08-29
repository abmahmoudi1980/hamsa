import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:persian_datetime_picker/persian_datetime_picker.dart';

import '../../../core/theme/app_theme.dart';
import '../../../core/datetime/jalali.dart';
import '../../../core/l10n/app_localizations.dart';
import '../../../shared/formatters/money_text.dart';
import '../../../shared/widgets/empty_state.dart';
import '../../../shared/widgets/status_chip.dart';
import '../../../shared/widgets/status_labels.dart';
import '../../buildings/buildings_controller.dart';
import '../models/payment.dart';
import '../payment_controller.dart';

/// US5/T060 — manager payment ledger for one building, filterable by unit,
/// method, and Jalali date range (contracts/api.md "Payments & Balances").
class PaymentLedgerScreen extends ConsumerStatefulWidget {
  const PaymentLedgerScreen({super.key, required this.buildingId});

  final String buildingId;

  @override
  ConsumerState<PaymentLedgerScreen> createState() =>
      _PaymentLedgerScreenState();
}

class _PaymentLedgerScreenState extends ConsumerState<PaymentLedgerScreen> {
  String _method = '';
  String? _unitId;
  DateTime? _from;
  DateTime? _to;
  List<Payment>? _results;
  bool _loading = false;
  String? _error;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    final l10n = AppLocalizations.of(context);
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final (items, _) =
          await ref.read(paymentRepositoryProvider).buildingPayments(
                widget.buildingId,
                unitId: _unitId,
                method: _method,
                from: _from == null ? null : isoDate(_from!),
                to: _to == null ? null : isoDate(_to!),
              );
      if (!mounted) return;
      setState(() => _results = items);
    } catch (_) {
      if (!mounted) return;
      setState(() => _error = l10n.errorServer);
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  Future<void> _pickDate({required bool isFrom}) async {
    final picked = await showPersianDatePicker(
      context: context,
      initialDate: Jalali.now(),
      firstDate: Jalali(1380, 1, 1),
      lastDate: Jalali(1450, 12, 29),
    );
    if (picked == null) return;
    setState(() {
      if (isFrom) {
        _from = picked.toDateTime();
      } else {
        _to = picked.toDateTime();
      }
    });
    _load();
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final units = ref.watch(unitsControllerProvider(widget.buildingId));
    final unitList = units.valueOrNull?.units ?? const [];
    return Scaffold(
      appBar: AppBar(title: Text(l10n.ledgerTitle)),
      body: Column(
        children: [
          Padding(
            padding: const EdgeInsets.symmetric(
              horizontal: AppTheme.spaceXl,
              vertical: AppTheme.spaceS,
            ),
            child: Wrap(
              spacing: AppTheme.spaceS,
              runSpacing: AppTheme.spaceS,
              alignment: WrapAlignment.end,
              children: [
                DropdownButton<String>(
                  value: _method,
                  hint: Text(l10n.filterMethodLabel),
                  items: [
                    DropdownMenuItem(value: '', child: Text(l10n.filterAll)),
                    ...paymentMethodLabels.entries.map(
                      (e) => DropdownMenuItem(value: e.key, child: Text(e.value)),
                    ),
                  ],
                  onChanged: (v) {
                    setState(() => _method = v ?? '');
                    _load();
                  },
                ),
                DropdownButton<String>(
                  value: _unitId,
                  hint: Text(l10n.unitsTitle),
                  items: [
                    DropdownMenuItem(value: null, child: Text(l10n.filterAll)),
                    ...unitList.map(
                      (u) => DropdownMenuItem(
                        value: u.id,
                        child: Text(toPersianDigits(u.number)),
                      ),
                    ),
                  ],
                  onChanged: (v) {
                    setState(() => _unitId = v);
                    _load();
                  },
                ),
                ActionChip(
                  label: Text(_from == null
                      ? l10n.filterFromDate
                      : formatJalaliDate(_from!)),
                  onPressed: () => _pickDate(isFrom: true),
                ),
                ActionChip(
                  label: Text(_to == null
                      ? l10n.filterToDate
                      : formatJalaliDate(_to!)),
                  onPressed: () => _pickDate(isFrom: false),
                ),
              ],
            ),
          ),
          const Divider(height: 1),
          Expanded(child: _buildBody(l10n)),
        ],
      ),
    );
  }

  Widget _buildBody(AppLocalizations l10n) {
    if (_loading) {
      return const Center(child: CircularProgressIndicator());
    }
    if (_error != null) {
      return EmptyState(icon: Icons.error_outline, title: _error!);
    }
    final items = _results ?? const <Payment>[];
    if (items.isEmpty) {
      return EmptyState(title: l10n.noPayments);
    }
    return ListView.separated(
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
                MoneyText(
                  amount: p.amount,
                  style: Theme.of(context).textTheme.titleMedium,
                ),
                const Spacer(),
                StatusChip(kind: StatusKind.payment, value: p.status),
              ],
            ),
            subtitle: Text(
              '${p.unitNumber == null ? '' : '${l10n.unitLabel} ${toPersianDigits(p.unitNumber!)} — '}'
              '${paymentMethodLabels[p.method] ?? p.method}'
              ' — ${formatJalaliDate(DateTime.parse(p.paidAt))}'
              '${p.trackingNumber == null ? '' : ' — ${toPersianDigits(p.trackingNumber!)}'}',
            ),
          ),
        );
      },
    );
  }
}
