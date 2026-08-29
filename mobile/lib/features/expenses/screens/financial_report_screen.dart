import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:persian_datetime_picker/persian_datetime_picker.dart';

import '../../../core/datetime/jalali.dart';
import '../../../core/l10n/app_localizations.dart';
import '../../../shared/formatters/money_text.dart';
import '../expense_controller.dart';
import '../models/expense.dart';

/// US6/T067 — financial report (FR-027): Jalali month selector, summary
/// cards with Persian-digit Toman amounts. Month is derived from any date
/// picked in the T018 picker (no Gregorian input anywhere in the app).
class FinancialReportScreen extends ConsumerStatefulWidget {
  const FinancialReportScreen({super.key, required this.buildingId});

  final String buildingId;

  @override
  ConsumerState<FinancialReportScreen> createState() =>
      _FinancialReportScreenState();
}

class _FinancialReportScreenState extends ConsumerState<FinancialReportScreen> {
  DateTime? _monthRef; // any day inside the selected month
  FinancialReport? _report;
  bool _loading = false;
  String? _error;

  @override
  void initState() {
    super.initState();
    _monthRef = DateTime.now();
    WidgetsBinding.instance.addPostFrameCallback((_) => _load());
  }

  Future<void> _load() async {
    final d = _monthRef;
    if (d == null) return;
    final l10n = AppLocalizations.of(context);
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final report = await ref
          .read(expenseRepositoryProvider)
          .financialReport(widget.buildingId, _monthKey(d));
      if (!mounted) return;
      setState(() => _report = report);
    } catch (_) {
      if (!mounted) return;
      setState(() => _error = l10n.errorServer);
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  /// Wire month key (YYYY-MM, Gregorian — the calendar the API stores).
  String _monthKey(DateTime d) =>
      '${d.year.toString().padLeft(4, '0')}-${d.month.toString().padLeft(2, '0')}';

  /// `مرداد ۱۴۰۴` — Jalali month label for the selected month.
  String _monthLabel(DateTime d) {
    final j = jalaliOf(d);
    return '${jalaliMonthName(j.month)} ${toPersianDigits('${j.year}')}';
  }

  Future<void> _pickMonth() async {
    final now = Jalali.now();
    final picked = await showPersianDatePicker(
      context: context,
      initialDate:
          _monthRef == null ? now : jalaliOf(_monthRef!),
      firstDate: Jalali(1380, 1, 1),
      lastDate: now.addYears(30),
      currentDate: now,
    );
    if (picked == null) return;
    setState(() => _monthRef = picked.toDateTime());
    _load();
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final r = _report;
    return Scaffold(
      appBar: AppBar(title: Text(l10n.financialReport)),
      body: _loading
          ? const Center(child: CircularProgressIndicator())
          : _error != null
              ? Center(child: Text(_error!))
              : ListView(
                  padding: const EdgeInsets.all(16),
                  children: [
                    // Jalali month selector — reuse of the date picker is the
                    // only calendar entry point in the app (T018 constraint).
                    Card(
                      child: ListTile(
                        leading: const Icon(Icons.calendar_month_outlined),
                        title: Text(l10n.reportMonthLabel),
                        subtitle: Text(
                          _monthRef == null ? '' : _monthLabel(_monthRef!),
                          style: Theme.of(context).textTheme.titleMedium,
                        ),
                        trailing: const Icon(Icons.chevron_left),
                        onTap: _pickMonth,
                      ),
                    ),
                    const SizedBox(height: 16),
                    if (r != null) ...[
                      _SummaryCard(
                        title: l10n.reportMonthlyIncome,
                        amount: r.monthlyIncome,
                        color: Colors.green,
                      ),
                      _SummaryCard(
                        title: l10n.reportMonthlyExpense,
                        amount: r.monthlyExpense,
                        color: Colors.red,
                      ),
                      _SummaryCard(
                        title: l10n.reportNet,
                        amount: r.net,
                        color: r.net >= 0 ? Colors.green : Colors.red,
                      ),
                      const Divider(height: 24),
                      _SummaryCard(
                        title: l10n.reportTotalDebt,
                        amount: r.totalDebt,
                        color: Colors.deepOrange,
                      ),
                      _SummaryCard(
                        title: l10n.reportTotalPayments,
                        amount: r.totalPayments,
                        color: Colors.blue,
                      ),
                      _SummaryCard(
                        title: l10n.reportTotalExpenses,
                        amount: r.totalExpenses,
                        color: Colors.brown,
                      ),
                    ],
                  ],
                ),
    );
  }
}

class _SummaryCard extends StatelessWidget {
  const _SummaryCard({
    required this.title,
    required this.amount,
    required this.color,
  });

  final String title;
  final int amount;
  final Color color;

  @override
  Widget build(BuildContext context) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
        child: Row(
          children: [
            Expanded(child: Text(title)),
            MoneyText(
              amount: amount,
              style: Theme.of(context)
                  .textTheme
                  .titleMedium
                  ?.copyWith(color: color, fontWeight: FontWeight.bold),
            ),
          ],
        ),
      ),
    );
  }
}
