import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../core/l10n/app_localizations.dart';
import '../../../core/theme/app_theme.dart';
import '../../../shared/formatters/money_text.dart';
import '../../../shared/widgets/empty_state.dart';
import '../../../shared/widgets/menu_card.dart';
import '../../buildings/buildings_controller.dart';
import '../../buildings/models/building.dart';
import '../dashboard_controller.dart';
import '../models/dashboard.dart';

/// US10 (T087) — Manager dashboard: cards (unit count, debtor units,
/// total debt, month income/expense, open requests, pending expenses),
/// alerts list (debtor / past-due / open-request / pending-expense), and
/// quick-action buttons deep-linking to issue charge / record expense /
/// record payment / send announcement / open requests. Building picker is
/// a fallback for managers with multiple buildings.
class ManagerDashboardScreen extends ConsumerWidget {
  const ManagerDashboardScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final buildingsAsync = ref.watch(buildingsControllerProvider);

    return Scaffold(
      appBar: AppBar(
        title: Text(l10n.dashboardTitle),
        actions: [
          IconButton(
            icon: const Icon(Icons.person_add_alt_outlined),
            tooltip: l10n.inviteTitle,
            onPressed: () => context.push('/manager/invite'),
          ),
          IconButton(
            icon: const Icon(Icons.apartment),
            tooltip: l10n.buildingsTitle,
            onPressed: () => context.push('/manager/buildings'),
          ),
        ],
      ),
      body: buildingsAsync.when(
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (_, _) => Center(
          child: EmptyState(
            icon: Icons.error_outline,
            title: l10n.errorServer,
          ),
        ),
        data: (buildings) {
          if (buildings.isEmpty) {
            return EmptyState(
              icon: Icons.apartment,
              title: l10n.dashboardNoBuilding,
              actionLabel: l10n.addBuilding,
              onAction: () => context.push('/manager/buildings/new'),
            );
          }
          return _BuildingDashboardList(buildings: buildings);
        },
      ),
    );
  }
}

class _BuildingDashboardList extends ConsumerStatefulWidget {
  const _BuildingDashboardList({required this.buildings});
  final List<Building> buildings;

  @override
  ConsumerState<_BuildingDashboardList> createState() =>
      _BuildingDashboardListState();
}

class _BuildingDashboardListState
    extends ConsumerState<_BuildingDashboardList> {
  // The manager's explicitly picked building (null = first building).
  // Re-checked against the current list so a stale pick (e.g. the building
  // was removed elsewhere) falls back to the first instead of watching a
  // family member nobody else renders.
  String? _pickedId;

  @override
  Widget build(BuildContext context) {
    final buildings = widget.buildings;
    final buildingId =
        _pickedId != null && buildings.any((b) => b.id == _pickedId)
            ? _pickedId!
            : buildings.first.id;
    final async = ref.watch(buildingDashboardControllerProvider(buildingId));

    return RefreshIndicator(
      onRefresh: () => ref
          .read(buildingDashboardControllerProvider(buildingId).notifier)
          .refresh(),
      child: ListView(
        padding: AppTheme.pagePadding,
        children: [
          _BuildingPicker(
            buildings: buildings
                .map((b) => (id: b.id, name: b.name))
                .toList(),
            selectedId: buildingId,
            onSelect: (id) {
              // Force a fresh fetch for the target building, then actually
              // render it by updating the watched selection.
              ref.invalidate(buildingDashboardControllerProvider(id));
              setState(() => _pickedId = id);
            },
          ),
          const SizedBox(height: AppTheme.spaceM),
          async.when(
            loading: () => const Padding(
              padding: EdgeInsets.symmetric(vertical: AppTheme.spaceXl),
              child: Center(child: CircularProgressIndicator()),
            ),
            error: (_, _) => EmptyState(
              icon: Icons.error_outline,
              title: l10nFor(context).errorServer,
            ),
            data: (dash) => Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                _CardsSection(dash: dash),
                const SizedBox(height: AppTheme.spaceL),
                _AlertsSection(dash: dash),
                const SizedBox(height: AppTheme.spaceL),
                _QuickActionsSection(dash: dash),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

AppLocalizations l10nFor(BuildContext context) => AppLocalizations.of(context);

class _BuildingPicker extends StatelessWidget {
  const _BuildingPicker({
    required this.buildings,
    required this.selectedId,
    required this.onSelect,
  });
  final List<({String id, String name})> buildings;
  final String selectedId;
  final void Function(String) onSelect;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final selected = buildings.firstWhere(
      (b) => b.id == selectedId,
      orElse: () => buildings.first,
    );
    return Align(
      alignment: AlignmentDirectional.centerStart,
      child: TextButton.icon(
        onPressed: () async {
          final picked = await showModalBottomSheet<String>(
            context: context,
            builder: (_) => SafeArea(
              child: ListView(
                shrinkWrap: true,
                children: [
                  Padding(
                    padding: const EdgeInsets.all(AppTheme.spaceL),
                    child: Text(l10n.dashboardBuildingPicker,
                        style: Theme.of(context).textTheme.titleMedium),
                  ),
                  for (final b in buildings)
                    ListTile(
                      leading: const Icon(Icons.apartment),
                      title: Text(b.name),
                      onTap: () => Navigator.of(context).pop(b.id),
                    ),
                ],
              ),
            ),
          );
          if (picked != null) onSelect(picked);
        },
        icon: const Icon(Icons.apartment),
        label: Text('${l10n.dashboardBuildingPicker}: ${selected.name}'),
      ),
    );
  }
}

class _CardsSection extends StatelessWidget {
  const _CardsSection({required this.dash});
  final BuildingDashboard dash;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final theme = Theme.of(context);
    final scheme = theme.colorScheme;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        SectionHeader(title: l10n.dashboardCards),
        GridView.count(
          crossAxisCount: 2,
          shrinkWrap: true,
          physics: const NeverScrollableScrollPhysics(),
          mainAxisSpacing: AppTheme.spaceM,
          crossAxisSpacing: AppTheme.spaceM,
          childAspectRatio: 1.4,
          children: [
            _StatCard(
              icon: Icons.apartment,
              title: l10n.dashboardUnitCount,
              value: dash.unitCount.toString(),
              color: scheme.primary,
            ),
            _StatCard(
              icon: Icons.people,
              title: l10n.dashboardOccupiedUnits,
              value: dash.occupiedUnitCount.toString(),
              color: scheme.tertiary,
            ),
            _StatCard(
              icon: Icons.warning_amber,
              title: l10n.dashboardDebtorUnits,
              value: dash.debtorUnitCount.toString(),
              color: scheme.error,
            ),
            _StatCard(
              icon: Icons.account_balance_wallet,
              title: l10n.dashboardTotalDebt,
              valueWidget: MoneyText(
                amount: dash.totalDebt,
                showCurrencySuffix: false,
                style: theme.textTheme.titleLarge?.copyWith(
                  color: scheme.error,
                  fontWeight: FontWeight.w800,
                ),
              ),
              color: scheme.error,
            ),
            _StatCard(
              icon: Icons.trending_up,
              title: l10n.dashboardMonthIncome,
              valueWidget: MoneyText(
                amount: dash.monthIncome,
                showCurrencySuffix: false,
                style: theme.textTheme.titleLarge?.copyWith(
                  color: scheme.primary,
                  fontWeight: FontWeight.w800,
                ),
              ),
              color: scheme.primary,
            ),
            _StatCard(
              icon: Icons.trending_down,
              title: l10n.dashboardMonthExpense,
              valueWidget: MoneyText(
                amount: dash.monthExpense,
                showCurrencySuffix: false,
                style: theme.textTheme.titleLarge?.copyWith(
                  color: scheme.tertiary,
                  fontWeight: FontWeight.w800,
                ),
              ),
              color: scheme.tertiary,
            ),
            _StatCard(
              icon: Icons.build,
              title: l10n.dashboardOpenRequests,
              value: dash.openRequests.toString(),
              color: scheme.secondary,
            ),
            _StatCard(
              icon: Icons.receipt,
              title: l10n.dashboardPendingExpenses,
              value: dash.pendingExpenses.toString(),
              color: scheme.secondary,
            ),
          ],
        ),
      ],
    );
  }
}

class _StatCard extends StatelessWidget {
  const _StatCard({
    required this.icon,
    required this.title,
    this.value,
    this.valueWidget,
    required this.color,
  }) : assert(value != null || valueWidget != null,
            'either value or valueWidget must be provided');

  final IconData icon;
  final String title;
  final String? value;
  final Widget? valueWidget;
  final Color color;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(AppTheme.spaceM),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Icon(icon, size: 18, color: color),
                const SizedBox(width: AppTheme.spaceS),
                Expanded(
                  child: Text(
                    title,
                    style: theme.textTheme.bodySmall,
                    maxLines: 2,
                    overflow: TextOverflow.ellipsis,
                  ),
                ),
              ],
            ),
            const Spacer(),
            valueWidget ??
                Text(
                  value!,
                  style: theme.textTheme.headlineSmall?.copyWith(
                    color: color,
                    fontWeight: FontWeight.w800,
                  ),
                ),
          ],
        ),
      ),
    );
  }
}

class _AlertsSection extends StatelessWidget {
  const _AlertsSection({required this.dash});
  final BuildingDashboard dash;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        SectionHeader(title: l10n.dashboardAlerts),
        if (dash.alerts.isEmpty)
          EmptyState(icon: Icons.check_circle, title: l10n.dashboardNoAlerts)
        else
          ...dash.alerts.map((a) => _AlertTile(alert: a)),
      ],
    );
  }
}

class _AlertTile extends StatelessWidget {
  const _AlertTile({required this.alert});
  final DashboardAlert alert;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final theme = Theme.of(context);
    final scheme = theme.colorScheme;
    final color = switch (alert.severity) {
      2 => scheme.error,
      1 => scheme.tertiary,
      _ => scheme.primary,
    };
    final icon = switch (alert.kind) {
      'debtor_unit' => Icons.warning_amber,
      'past_due_invoice' => Icons.schedule,
      'open_request' => Icons.build,
      'pending_expense' => Icons.receipt,
      _ => Icons.info_outline,
    };
    final title = switch (alert.kind) {
      'debtor_unit' => l10n.dashboardAlertDebtor(alert.unitNumber),
      'past_due_invoice' => l10n.dashboardAlertPastDue(alert.unitNumber),
      'open_request' => l10n.dashboardAlertOpenRequest(alert.title),
      'pending_expense' => l10n.dashboardAlertPendingExpense(alert.title),
      _ => alert.title,
    };

    return Card(
      child: ListTile(
        leading: Icon(icon, color: color),
        title: Text(title),
        trailing: alert.amount > 0
            ? MoneyText(
                amount: alert.amount,
                showCurrencySuffix: false,
                style: theme.textTheme.titleSmall?.copyWith(color: color),
              )
            : null,
        onTap: () {
          // Deep-link the alert to its underlying detail when possible.
          final ref = alert.refType;
          final id = alert.refId;
          if (ref == 'invoice' && id.isNotEmpty) {
            context.push('/invoice/$id');
          }
          // unit / maintenance_request / expense: navigated by manager
          // through their existing list screens; no per-alert screen in P0.
        },
      ),
    );
  }
}

class _QuickActionsSection extends StatelessWidget {
  const _QuickActionsSection({required this.dash});
  final BuildingDashboard dash;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        SectionHeader(title: l10n.dashboardQuickActions),
        GridView.count(
          crossAxisCount: 2,
          shrinkWrap: true,
          physics: const NeverScrollableScrollPhysics(),
          mainAxisSpacing: AppTheme.spaceM,
          crossAxisSpacing: AppTheme.spaceM,
          childAspectRatio: 2.0,
          children: dash.quickActions.map((qa) {
            final label = _quickLabel(l10n, qa.key);
            return _QuickActionButton(
              label: label,
              path: qa.path,
              icon: _quickIcon(qa.key),
            );
          }).toList(),
        ),
      ],
    );
  }

  String _quickLabel(AppLocalizations l10n, String key) {
    return switch (key) {
      'issue_charge' => l10n.dashboardActionIssueCharge,
      'record_expense' => l10n.dashboardActionRecordExpense,
      'record_payment' => l10n.dashboardActionRecordPayment,
      'send_announcement' => l10n.dashboardActionSendAnnouncement,
      'open_requests' => l10n.dashboardActionOpenRequests,
      _ => key,
    };
  }

  IconData _quickIcon(String key) {
    return switch (key) {
      'issue_charge' => Icons.receipt_long,
      'record_expense' => Icons.attach_money,
      'record_payment' => Icons.payments,
      'send_announcement' => Icons.campaign,
      'open_requests' => Icons.build,
      _ => Icons.arrow_forward,
    };
  }
}

class _QuickActionButton extends StatelessWidget {
  const _QuickActionButton({
    required this.label,
    required this.path,
    required this.icon,
  });
  final String label;
  final String path;
  final IconData icon;

  @override
  Widget build(BuildContext context) {
    return FilledButton.icon(
      onPressed: () => context.push(path),
      icon: Icon(icon),
      label: Text(label, overflow: TextOverflow.ellipsis),
    );
  }
}
