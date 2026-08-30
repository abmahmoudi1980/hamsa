import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../core/theme/app_theme.dart';
import '../../../core/datetime/jalali.dart';
import '../../../core/l10n/app_localizations.dart';
import '../../../shared/formatters/money_text.dart';
import '../../../shared/widgets/empty_state.dart';
import '../../../shared/widgets/status_chip.dart';
import '../../../shared/widgets/status_labels.dart';
import '../expense_controller.dart';
import '../models/expense.dart';

/// US6/T066 — manager expense list for one building, filterable by category
/// and approval status (contracts/api.md "Expenses & Financial Report").
/// The financial report (T067) is reachable from the app bar.
class ExpenseListScreen extends ConsumerStatefulWidget {
  const ExpenseListScreen({super.key, required this.buildingId});

  final String buildingId;

  @override
  ConsumerState<ExpenseListScreen> createState() => _ExpenseListScreenState();
}

class _ExpenseListScreenState extends ConsumerState<ExpenseListScreen> {
  String _category = '';
  String _approval = '';
  List<Expense>? _results;
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
          .read(expenseRepositoryProvider)
          .buildingExpenses(
            widget.buildingId,
            category: _category,
            approval: _approval,
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

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return Scaffold(
      appBar: AppBar(
        title: Text(l10n.expensesTitle),
        actions: [
          IconButton(
            icon: const Icon(Icons.assessment_outlined),
            tooltip: l10n.financialReport,
            onPressed: () =>
                context.push('/manager/report/${widget.buildingId}'),
          ),
        ],
      ),
      floatingActionButton: FloatingActionButton.extended(
        onPressed: () async {
          await context.push('/manager/expenses/${widget.buildingId}/new');
          if (mounted) _load();
        },
        icon: const Icon(Icons.add),
        label: Text(l10n.addExpense),
      ),
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
                  isDense: true,
                  value: _category.isEmpty ? null : _category,
                  hint: Text(l10n.filterCategoryLabel),
                  items: [
                    DropdownMenuItem(value: '', child: Text(l10n.filterAll)),
                    ...expenseCategoryLabels.entries.map(
                      (e) => DropdownMenuItem(value: e.key, child: Text(e.value)),
                    ),
                  ],
                  onChanged: (v) {
                    setState(() => _category = v ?? '');
                    _load();
                  },
                ),
                DropdownButton<String>(
                  isDense: true,
                  value: _approval.isEmpty ? null : _approval,
                  hint: Text(l10n.filterApprovalLabel),
                  items: [
                    DropdownMenuItem(value: '', child: Text(l10n.filterAll)),
                    ...expenseApprovalLabels.entries.map(
                      (e) => DropdownMenuItem(value: e.key, child: Text(e.value)),
                    ),
                  ],
                  onChanged: (v) {
                    setState(() => _approval = v ?? '');
                    _load();
                  },
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
    final items = _results ?? const <Expense>[];
    if (items.isEmpty) {
      return EmptyState(title: l10n.noExpenses);
    }
    return ListView.separated(
      padding: AppTheme.pagePadding,
      itemCount: items.length,
      separatorBuilder: (_, _) => const SizedBox(height: 10),
      itemBuilder: (context, i) {
        final e = items[i];
        return Card(
          child: ListTile(
            leading: const Icon(Icons.receipt_outlined),
            title: Row(
              children: [
                Expanded(
                  child: Text(e.title, style: Theme.of(context).textTheme.titleMedium),
                ),
                MoneyText(amount: e.amount),
              ],
            ),
            subtitle: Text(
              '${expenseCategoryLabels[e.category] ?? e.category}'
              ' — ${formatJalaliDate(DateTime.parse(e.expenseDate))}',
            ),
            trailing: StatusChip(kind: StatusKind.expense, value: e.approvalStatus),
            onTap: () async {
              await context.push(
                '/manager/expenses/${widget.buildingId}/${e.id}',
              );
              if (mounted) _load();
            },
          ),
        );
      },
    );
  }
}
