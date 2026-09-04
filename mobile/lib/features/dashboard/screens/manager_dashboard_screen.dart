import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../core/datetime/jalali.dart';
import '../../../core/l10n/app_localizations.dart';
import '../../../core/theme/app_theme.dart';
import '../../../shared/formatters/money_text.dart';
import '../../../shared/widgets/empty_state.dart';
import '../../../shared/widgets/menu_card.dart';
import '../../buildings/buildings_controller.dart';
import '../../buildings/models/building.dart';
import '../../auth/auth_controller.dart';
import '../../auth/models/user_session.dart';
import '../dashboard_controller.dart';
import '../models/dashboard.dart';

/// US10 (T087) — Manager dashboard, redesigned around one question per
/// glance: "does this building need my attention, and what do I do next?"
///
/// Reading order is attention-first: building identity → collection hero
/// (total debt + debtors + record-payment CTA) → alerts → quick actions →
/// month finance → building facts. The old 8-card stat grid treated trivia
/// (unit count) and criticals (total debt) as equals and buried the alerts
/// below the fold; every number here either acts (hero CTA, alert tiles,
/// quick actions) or informs at a glance (finance/facts rows).
class ManagerDashboardScreen extends ConsumerWidget {
  const ManagerDashboardScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);

    // 002-multi-manager-support (T030): the superadmin manages no buildings
    // (`GET /buildings` is manager-only server-side) — its home is just the
    // invite tool, so the buildings list is never fetched here.
    if (ref.watch(authControllerProvider).user?.role == UserRole.superadmin) {
      return Scaffold(
        appBar: AppBar(
          title: Text(l10n.superadminHomeTitle),
          actions: [
            IconButton(
              icon: const Icon(Icons.logout),
              tooltip: l10n.logout,
              onPressed: () =>
                  ref.read(authControllerProvider.notifier).logout(),
            ),
          ],
        ),
        body: EmptyState(
          icon: Icons.person_add_alt_outlined,
          title: l10n.superadminHomeHint,
          actionLabel: l10n.getInviteCode,
          onAction: () => context.push('/manager/invite'),
        ),
      );
    }

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
            actionLabel: l10n.retry,
            onAction: () => ref.invalidate(buildingsControllerProvider),
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
    final building = _pickedId != null &&
            buildings.any((b) => b.id == _pickedId)
        ? buildings.firstWhere((b) => b.id == _pickedId)
        : buildings.first;
    final async = ref.watch(buildingDashboardControllerProvider(building.id));

    return RefreshIndicator(
      onRefresh: () => ref
          .read(buildingDashboardControllerProvider(building.id).notifier)
          .refresh(),
      child: ListView(
        padding: AppTheme.pagePadding,
        children: [
          _BuildingHeader(
            name: building.name,
            showSwitcher: buildings.length > 1,
            onSwitch: () => _switchBuilding(context, buildings, building.id),
          ),
          const SizedBox(height: AppTheme.spaceM),
          async.when(
            loading: () => const Padding(
              padding: EdgeInsets.symmetric(vertical: AppTheme.spaceXl),
              child: Center(child: CircularProgressIndicator()),
            ),
            error: (_, _) => EmptyState(
              icon: Icons.error_outline,
              title: AppLocalizations.of(context).errorServer,
              actionLabel: AppLocalizations.of(context).retry,
              onAction: () => ref
                  .read(
                    buildingDashboardControllerProvider(building.id).notifier,
                  )
                  .refresh(),
            ),
            data: (dash) => Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                _CollectionHero(dash: dash),
                const SizedBox(height: AppTheme.spaceL),
                _AlertsSection(dash: dash),
                const SizedBox(height: AppTheme.spaceL),
                _QuickActionsRow(actions: dash.quickActions),
                const SizedBox(height: AppTheme.spaceL),
                _FinanceCard(dash: dash),
                const SizedBox(height: AppTheme.spaceM),
                _FactsCard(dash: dash),
              ],
            ),
          ),
        ],
      ),
    );
  }

  Future<void> _switchBuilding(
    BuildContext context,
    List<Building> buildings,
    String selectedId,
  ) async {
    final l10n = AppLocalizations.of(context);
    final picked = await showModalBottomSheet<String>(
      context: context,
      builder: (_) => SafeArea(
        child: ListView(
          shrinkWrap: true,
          children: [
            Padding(
              padding: const EdgeInsets.all(AppTheme.spaceL),
              child: Text(
                l10n.dashboardBuildingPicker,
                style: Theme.of(context).textTheme.titleMedium,
              ),
            ),
            for (final b in buildings)
              ListTile(
                leading: const Icon(Icons.apartment),
                title: Text(b.name),
                trailing: b.id == selectedId
                    ? Icon(
                        Icons.check,
                        color: Theme.of(context).colorScheme.primary,
                      )
                    : null,
                onTap: () => Navigator.of(context).pop(b.id),
              ),
          ],
        ),
      ),
    );
    if (picked != null && picked != selectedId) {
      // Force a fresh fetch for the target building, then actually
      // render it by updating the watched selection.
      ref.invalidate(buildingDashboardControllerProvider(picked));
      setState(() => _pickedId = picked);
    }
  }
}

/// Building identity + switcher. One building: pure identity (no dead
/// switcher control); several: the whole card switches.
class _BuildingHeader extends StatelessWidget {
  const _BuildingHeader({
    required this.name,
    required this.showSwitcher,
    required this.onSwitch,
  });

  final String name;
  final bool showSwitcher;
  final VoidCallback onSwitch;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final scheme = theme.colorScheme;
    final l10n = AppLocalizations.of(context);
    return Card(
      child: InkWell(
        onTap: showSwitcher ? onSwitch : null,
        borderRadius: BorderRadius.circular(AppTheme.radius),
        child: Padding(
          padding: const EdgeInsets.all(AppTheme.spaceL),
          child: Row(
            children: [
              Container(
                width: 44,
                height: 44,
                decoration: BoxDecoration(
                  color: scheme.primary.withValues(alpha: 0.1),
                  borderRadius: BorderRadius.circular(AppTheme.radiusSmall),
                ),
                child: Icon(
                  Icons.apartment,
                  size: 22,
                  color: scheme.primary,
                ),
              ),
              const SizedBox(width: AppTheme.spaceL),
              Expanded(
                child: Text(
                  name,
                  style: theme.textTheme.titleLarge,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                ),
              ),
              if (showSwitcher) ...[
                const SizedBox(width: AppTheme.spaceS),
                Text(
                  l10n.dashboardBuildingPicker,
                  style: theme.textTheme.bodySmall?.copyWith(
                    color: scheme.primary,
                  ),
                ),
                Icon(Icons.swap_horiz, size: 20, color: scheme.primary),
              ],
            ],
          ),
        ),
      ),
    );
  }
}

/// The primary action surface: what is owed, by how many units, and the
/// single CTA that moves money (record-payment deep link from the server's
/// own quick actions, ledger fallback).
class _CollectionHero extends StatelessWidget {
  const _CollectionHero({required this.dash});
  final BuildingDashboard dash;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final scheme = theme.colorScheme;
    final l10n = AppLocalizations.of(context);
    final onPrimary = scheme.onPrimaryContainer;
    final recordPayment = dash.quickActions
        .where((qa) => qa.key == 'record_payment')
        .firstOrNull;
    return Card(
      color: scheme.primaryContainer,
      child: Padding(
        padding: const EdgeInsets.all(AppTheme.spaceL),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              l10n.dashboardTotalDebt,
              style: theme.textTheme.bodyMedium?.copyWith(color: onPrimary),
            ),
            const SizedBox(height: AppTheme.spaceXs),
            MoneyText(
              amount: dash.totalDebt,
              style: theme.textTheme.headlineMedium?.copyWith(
                color: onPrimary,
                fontWeight: FontWeight.w800,
              ),
            ),
            const SizedBox(height: AppTheme.spaceXs),
            Text(
              l10n.dashboardDebtorCount(
                toPersianDigits(dash.debtorUnitCount.toString()),
              ),
              style: theme.textTheme.bodyMedium?.copyWith(color: onPrimary),
            ),
            const SizedBox(height: AppTheme.spaceM),
            SizedBox(
              width: double.infinity,
              child: FilledButton.icon(
                onPressed: () => context.push(
                  recordPayment?.path.isNotEmpty == true
                      ? recordPayment!.path
                      : '/manager/ledger/${dash.buildingId}',
                ),
                icon: const Icon(Icons.payments),
                label: Text(
                  recordPayment?.title.isNotEmpty == true
                      ? recordPayment!.title
                      : l10n.dashboardActionRecordPayment,
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

/// Compact wrapping action chips — every server-driven action is visible at
/// once, so nothing ever hides off-screen (the old fixed grid left orphan
/// cells; a scroll strip hid trailing actions with no affordance).
class _QuickActionsRow extends StatelessWidget {
  const _QuickActionsRow({required this.actions});
  final List<ManagerQuickAction> actions;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    if (actions.isEmpty) return const SizedBox.shrink();
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        SectionHeader(title: l10n.dashboardQuickActions),
        Wrap(
          spacing: AppTheme.spaceS,
          runSpacing: AppTheme.spaceS,
          children: [
            for (final qa in actions)
              FilledButton.tonalIcon(
                style: FilledButton.styleFrom(
                  minimumSize: const Size(64, 48),
                ),
                onPressed: () => context.push(qa.path),
                icon: Icon(_quickIcon(qa.key)),
                label: Text(_quickLabel(l10n, qa)),
              ),
          ],
        ),
      ],
    );
  }

  String _quickLabel(AppLocalizations l10n, ManagerQuickAction qa) {
    if (qa.title.isNotEmpty) return qa.title;
    return switch (qa.key) {
      'issue_charge' => l10n.dashboardActionIssueCharge,
      'record_expense' => l10n.dashboardActionRecordExpense,
      'record_payment' => l10n.dashboardActionRecordPayment,
      'send_announcement' => l10n.dashboardActionSendAnnouncement,
      'open_requests' => l10n.dashboardActionOpenRequests,
      _ => qa.key,
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

/// This month's money in two rows — no cards-per-number grid.
class _FinanceCard extends StatelessWidget {
  const _FinanceCard({required this.dash});
  final BuildingDashboard dash;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final scheme = theme.colorScheme;
    final l10n = AppLocalizations.of(context);
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(AppTheme.spaceL),
        child: Column(
          children: [
            _FinanceLine(
              icon: Icons.trending_up,
              color: scheme.primary,
              label: l10n.dashboardMonthIncome,
              amount: dash.monthIncome,
            ),
            const Divider(height: AppTheme.spaceL),
            _FinanceLine(
              icon: Icons.trending_down,
              color: scheme.tertiary,
              label: l10n.dashboardMonthExpense,
              amount: dash.monthExpense,
            ),
          ],
        ),
      ),
    );
  }
}

class _FinanceLine extends StatelessWidget {
  const _FinanceLine({
    required this.icon,
    required this.color,
    required this.label,
    required this.amount,
  });

  final IconData icon;
  final Color color;
  final String label;
  final int amount;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Row(
      children: [
        Icon(icon, size: 20, color: color),
        const SizedBox(width: AppTheme.spaceM),
        Expanded(
          child: Text(label, style: theme.textTheme.bodyMedium),
        ),
        MoneyText(
          amount: amount,
          style: theme.textTheme.titleMedium?.copyWith(
            color: color,
            fontWeight: FontWeight.w700,
          ),
        ),
      ],
    );
  }
}

/// Slow-moving building facts in one glanceable card — raw counts the
/// manager checks rarely, so they share a surface instead of shouting.
class _FactsCard extends StatelessWidget {
  const _FactsCard({required this.dash});
  final BuildingDashboard dash;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final facts = [
      (Icons.apartment, l10n.dashboardUnitCount, dash.unitCount),
      (Icons.people, l10n.dashboardOccupiedUnits, dash.occupiedUnitCount),
      (Icons.build, l10n.dashboardOpenRequests, dash.openRequests),
      (Icons.receipt, l10n.dashboardPendingExpenses, dash.pendingExpenses),
    ];
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(AppTheme.spaceL),
        child: Column(
          children: [
            for (var i = 0; i < facts.length; i++) ...[
              if (i > 0) const SizedBox(height: AppTheme.spaceM),
              _FactLine(
                icon: facts[i].$1,
                label: facts[i].$2,
                value: facts[i].$3,
              ),
            ],
          ],
        ),
      ),
    );
  }
}

class _FactLine extends StatelessWidget {
  const _FactLine({
    required this.icon,
    required this.label,
    required this.value,
  });

  final IconData icon;
  final String label;
  final int value;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final scheme = theme.colorScheme;
    return Row(
      children: [
        Icon(icon, size: 20, color: scheme.onSurfaceVariant),
        const SizedBox(width: AppTheme.spaceM),
        Expanded(
          child: Text(label, style: theme.textTheme.bodyMedium),
        ),
        Text(
          toPersianDigits(value.toString()),
          style: theme.textTheme.titleMedium?.copyWith(
            fontWeight: FontWeight.w700,
          ),
        ),
      ],
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
          ...dash.alerts.map(
            (a) => _AlertTile(alert: a, buildingId: dash.buildingId),
          ),
      ],
    );
  }
}

class _AlertTile extends StatelessWidget {
  const _AlertTile({required this.alert, required this.buildingId});
  final DashboardAlert alert;
  final String buildingId;

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
    final target = _target(alert);

    return Card(
      child: ListTile(
        leading: Icon(icon, color: color),
        title: Text(title),
        trailing: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            if (alert.amount > 0)
              MoneyText(
                amount: alert.amount,
                showCurrencySuffix: false,
                style: theme.textTheme.titleSmall?.copyWith(color: color),
              ),
            if (target != null)
              Icon(
                // Auto-mirrored: renders as ‹ in RTL (forward = left).
                Icons.chevron_right,
                size: 20,
                color: scheme.onSurfaceVariant,
              ),
          ],
        ),
        onTap: target == null ? null : () => context.push(target),
      ),
    );
  }

  /// Deep-link the alert to its underlying detail. Invoice, maintenance
  /// request, expense and unit alerts all have real destinations; anything
  /// unrecognized stays a plain (non-tappable) row.
  String? _target(DashboardAlert alert) {
    final id = alert.refId;
    if (id.isEmpty) return null;
    return switch (alert.refType) {
      'invoice' => '/invoice/$id',
      'maintenance_request' => '/manager/maintenance/$buildingId/$id',
      'expense' => '/manager/expenses/$buildingId/$id',
      'unit' => '/manager/buildings/$buildingId/units/$id/history',
      _ => null,
    };
  }
}
