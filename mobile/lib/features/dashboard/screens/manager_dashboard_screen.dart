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
/// Reading order stays attention-first — building identity → collection
/// hero → alerts → quick actions → month finance → building facts — but
/// every surface now encodes its number visually instead of stacking
/// another card: the hero carries a settled-units progress bar, alerts
/// collapse into one severity-coded list with a count badge, quick
/// actions form a uniform icon-tile grid, the finance card gains its
/// Jalali month plus a net-balance row, and slow-moving facts compress
/// into a 2×2 grid.
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
        body: SafeArea(
          child: EmptyState(
            icon: Icons.person_add_alt_outlined,
            title: l10n.superadminHomeHint,
            actionLabel: l10n.getInviteCode,
            onAction: () => context.push('/manager/invite'),
          )
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
      body: SafeArea(
        child: buildingsAsync.when(
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
        )
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
          const SizedBox(height: AppTheme.spaceL),
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
                _QuickActionGrid(actions: dash.quickActions),
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

/// Building identity + switcher as a plain header row (no card chrome):
/// the identity is context for the hero below it, not content of its own.
/// The name takes all free width (long names ellipsize); the switcher is a
/// compact tonal icon button so it never squeezes the identity on narrow
/// phones. One building: pure identity (no dead switcher control).
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
    return Row(
      children: [
        Container(
          width: 44,
          height: 44,
          decoration: BoxDecoration(
            color: scheme.primary.withValues(alpha: 0.1),
            borderRadius: BorderRadius.circular(AppTheme.radiusSmall),
          ),
          child: Icon(Icons.apartment, size: 22, color: scheme.primary),
        ),
        const SizedBox(width: AppTheme.spaceM),
        Expanded(
          child: Text(
            name,
            style: theme.textTheme.titleLarge,
            maxLines: 1,
            overflow: TextOverflow.ellipsis,
          ),
        ),
        if (showSwitcher) ...[
          const SizedBox(width: AppTheme.spaceM),
          IconButton.filledTonal(
            onPressed: onSwitch,
            tooltip: l10n.dashboardSwitchBuilding,
            icon: const Icon(Icons.swap_horiz),
          ),
        ],
      ],
    );
  }
}

/// The primary action surface: what is owed, how widespread the debt is
/// (settled-units progress bar), and the single CTA that moves money
/// (record-payment deep link from the server's own quick actions, ledger
/// fallback).
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

    // Collection health at a glance: the share of units that owe nothing.
    // Zero units → vacuously healthy (nothing to collect).
    final settled = dash.unitCount - dash.debtorUnitCount;
    final ratio = dash.unitCount <= 0 ? 1.0 : settled / dash.unitCount;
    final settledLabel = dash.debtorUnitCount <= 0
        ? l10n.dashboardAllUnitsSettled
        : l10n.dashboardSettledUnits(
            toPersianDigits(settled.toString()),
            toPersianDigits(dash.unitCount.toString()),
          );

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
            ClipRRect(
              borderRadius: BorderRadius.circular(999),
              child: LinearProgressIndicator(
                value: ratio.clamp(0.0, 1.0),
                minHeight: 8,
                color: scheme.primary,
                backgroundColor: onPrimary.withValues(alpha: 0.24),
              ),
            ),
            const SizedBox(height: AppTheme.spaceS),
            Text(
              settledLabel,
              style: theme.textTheme.bodySmall?.copyWith(color: onPrimary),
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

/// Needs-attention list in a single grouped card: severity-tinted icon,
/// title, amount and deep-link chevron per row — divided rows instead of
/// one card per alert so a long list stays compact. More than
/// [_collapsedCount] alerts collapse behind a show-all toggle; zero alerts
/// render as a slim positive confirmation, not an empty-state billboard.
class _AlertsSection extends StatefulWidget {
  const _AlertsSection({required this.dash});
  final BuildingDashboard dash;

  @override
  State<_AlertsSection> createState() => _AlertsSectionState();
}

class _AlertsSectionState extends State<_AlertsSection> {
  static const int _collapsedCount = 4;
  bool _expanded = false;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final theme = Theme.of(context);
    final scheme = theme.colorScheme;
    final alerts = widget.dash.alerts;
    final collapsed = !_expanded && alerts.length > _collapsedCount;
    final visible = collapsed ? alerts.take(_collapsedCount).toList() : alerts;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          children: [
            Expanded(child: SectionHeader(title: l10n.dashboardAlerts)),
            if (alerts.isNotEmpty)
              Padding(
                // SectionHeader carries bottom padding; match it so the
                // badge centers on the title text, not the padded row.
                padding: const EdgeInsets.only(bottom: AppTheme.spaceM),
                child: Container(
                  padding: const EdgeInsets.symmetric(
                    horizontal: 10,
                    vertical: 4,
                  ),
                  decoration: BoxDecoration(
                    color: scheme.errorContainer,
                    borderRadius: BorderRadius.circular(999),
                  ),
                  child: Text(
                    l10n.dashboardAlertsCount(
                      toPersianDigits(alerts.length.toString()),
                    ),
                    style: theme.textTheme.labelSmall?.copyWith(
                      color: scheme.onErrorContainer,
                      fontWeight: FontWeight.w700,
                    ),
                  ),
                ),
              ),
          ],
        ),
        if (alerts.isEmpty)
          _NoAlertsRow(message: l10n.dashboardNoAlerts)
        else ...[
          Card(
            child: Column(
              children: [
                for (var i = 0; i < visible.length; i++) ...[
                  if (i > 0)
                    const Divider(
                      indent: AppTheme.spaceL,
                      endIndent: AppTheme.spaceL,
                    ),
                  _AlertRow(
                    alert: visible[i],
                    buildingId: widget.dash.buildingId,
                  ),
                ],
              ],
            ),
          ),
          if (alerts.length > _collapsedCount)
            Align(
              alignment: AlignmentDirectional.centerStart,
              child: TextButton(
                onPressed: () => setState(() => _expanded = !_expanded),
                // Full 48dp touch target for the inline toggle.
                style: TextButton.styleFrom(
                  minimumSize: const Size(64, 48),
                ),
                child: Text(
                  _expanded
                      ? l10n.dashboardShowFewerAlerts
                      : l10n.dashboardShowAllAlerts(
                          toPersianDigits(alerts.length.toString()),
                        ),
                ),
              ),
            ),
        ],
      ],
    );
  }
}

/// Slim "all clear" strip — the good state deserves a whisper, not the
/// big centered EmptyState reserved for missing data.
class _NoAlertsRow extends StatelessWidget {
  const _NoAlertsRow({required this.message});
  final String message;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final scheme = theme.colorScheme;
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(AppTheme.spaceM),
      decoration: BoxDecoration(
        color: scheme.primary.withValues(alpha: 0.08),
        borderRadius: BorderRadius.circular(AppTheme.radius),
      ),
      child: Row(
        children: [
          Icon(Icons.check_circle_outline, size: 20, color: scheme.primary),
          const SizedBox(width: AppTheme.spaceM),
          Expanded(
            child: Text(message, style: theme.textTheme.bodyMedium),
          ),
        ],
      ),
    );
  }
}

class _AlertRow extends StatelessWidget {
  const _AlertRow({required this.alert, required this.buildingId});
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

    return InkWell(
      onTap: target == null ? null : () => context.push(target),
      child: Padding(
        padding: const EdgeInsets.symmetric(
          horizontal: AppTheme.spaceL,
          vertical: AppTheme.spaceM,
        ),
        child: Row(
          children: [
            Container(
              width: 36,
              height: 36,
              decoration: BoxDecoration(
                color: color.withValues(alpha: 0.12),
                borderRadius: BorderRadius.circular(10),
              ),
              child: Icon(icon, size: 18, color: color),
            ),
            const SizedBox(width: AppTheme.spaceM),
            Expanded(
              child: Text(
                title,
                maxLines: 2,
                overflow: TextOverflow.ellipsis,
                style: theme.textTheme.bodyMedium,
              ),
            ),
            if (alert.amount > 0) ...[
              const SizedBox(width: AppTheme.spaceS),
              MoneyText(
                amount: alert.amount,
                showCurrencySuffix: false,
                style: theme.textTheme.titleSmall?.copyWith(
                  color: color,
                  fontWeight: FontWeight.w700,
                ),
              ),
            ],
            if (target != null) ...[
              const SizedBox(width: AppTheme.spaceS),
              Icon(
                // Auto-mirrored: renders as ‹ in RTL (forward = left).
                Icons.chevron_right,
                size: 20,
                color: scheme.onSurfaceVariant,
              ),
            ],
          ],
        ),
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

/// Uniform icon-tile grid — every server-driven action is one equal-sized
/// touch target (≥48dp), so nothing hides off-screen and no action outranks
/// another. Cells size from the available width (max 88dp per column), so
/// phones get 3–4 columns and tablets more.
class _QuickActionGrid extends StatelessWidget {
  const _QuickActionGrid({required this.actions});
  final List<ManagerQuickAction> actions;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    if (actions.isEmpty) return const SizedBox.shrink();
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        SectionHeader(title: l10n.dashboardQuickActions),
        GridView(
          shrinkWrap: true,
          physics: const NeverScrollableScrollPhysics(),
          gridDelegate: const SliverGridDelegateWithMaxCrossAxisExtent(
            maxCrossAxisExtent: 88,
            mainAxisExtent: 104,
            crossAxisSpacing: AppTheme.spaceS,
            mainAxisSpacing: AppTheme.spaceS,
          ),
          children: [
            for (final qa in actions)
              _QuickActionTile(
                icon: _quickIcon(qa.key),
                label: _quickLabel(l10n, qa),
                onTap: () => context.push(qa.path),
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

class _QuickActionTile extends StatelessWidget {
  const _QuickActionTile({
    required this.icon,
    required this.label,
    required this.onTap,
  });

  final IconData icon;
  final String label;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final scheme = theme.colorScheme;
    return Card(
      child: InkWell(
        onTap: onTap,
        child: Padding(
          padding: const EdgeInsets.all(AppTheme.spaceS),
          child: Column(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              Container(
                width: 44,
                height: 44,
                decoration: BoxDecoration(
                  color: scheme.primary.withValues(alpha: 0.1),
                  borderRadius: BorderRadius.circular(AppTheme.radiusSmall),
                ),
                child: Icon(icon, size: 22, color: scheme.primary),
              ),
              const SizedBox(height: AppTheme.spaceS),
              Text(
                label,
                maxLines: 2,
                overflow: TextOverflow.ellipsis,
                textAlign: TextAlign.center,
                style: theme.textTheme.labelSmall?.copyWith(
                  color: scheme.onSurface,
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

/// This month's money in one card: the Jalali month in the title (the
/// server sends a Gregorian YYYY-MM anchor — contracts/api.md keeps the
/// calendar conversion client-side), income and expense side by side, and
/// the net balance as the punchline.
class _FinanceCard extends StatelessWidget {
  const _FinanceCard({required this.dash});
  final BuildingDashboard dash;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final theme = Theme.of(context);
    final scheme = theme.colorScheme;
    final net = dash.monthIncome - dash.monthExpense;
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(AppTheme.spaceL),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            SectionHeader(title: _monthTitle(l10n, dash.month)),
            Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Expanded(
                  child: _FinanceColumn(
                    icon: Icons.trending_up,
                    color: scheme.primary,
                    label: l10n.dashboardIncome,
                    amount: dash.monthIncome,
                  ),
                ),
                Container(
                  width: 1,
                  height: 44,
                  margin: const EdgeInsets.symmetric(
                    horizontal: AppTheme.spaceL,
                  ),
                  color: scheme.outlineVariant.withValues(alpha: 0.5),
                ),
                Expanded(
                  child: _FinanceColumn(
                    icon: Icons.trending_down,
                    color: scheme.tertiary,
                    label: l10n.dashboardExpense,
                    amount: dash.monthExpense,
                  ),
                ),
              ],
            ),
            const Divider(height: AppTheme.spaceL),
            Row(
              children: [
                Expanded(
                  child: Text(
                    l10n.dashboardNetBalance,
                    style: theme.textTheme.bodyMedium,
                  ),
                ),
                MoneyText(
                  amount: net,
                  style: theme.textTheme.titleMedium?.copyWith(
                    color: net >= 0 ? scheme.primary : scheme.error,
                    fontWeight: FontWeight.w700,
                  ),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }

  /// `month` is the server's Gregorian YYYY-MM anchor; the label renders
  /// the Jalali month at mid-month. Unparseable input falls back to the
  /// plain "this month" title rather than showing raw ISO text.
  String _monthTitle(AppLocalizations l10n, String month) {
    final parts = month.split('-');
    if (parts.length == 2) {
      final year = int.tryParse(parts[0]);
      final monthNumber = int.tryParse(parts[1]);
      if (year != null &&
          monthNumber != null &&
          monthNumber >= 1 &&
          monthNumber <= 12) {
        final j = jalaliOf(DateTime(year, monthNumber, 15));
        return l10n.dashboardMonthFinance(jalaliMonthName(j.month));
      }
    }
    return l10n.dashboardFinanceTitle;
  }
}

class _FinanceColumn extends StatelessWidget {
  const _FinanceColumn({
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
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          children: [
            Icon(icon, size: 18, color: color),
            const SizedBox(width: AppTheme.spaceS),
            Expanded(child: Text(label, style: theme.textTheme.bodySmall)),
          ],
        ),
        const SizedBox(height: AppTheme.spaceXs),
        // Scale-down instead of wrap: a seven-digit Toman amount must never
        // push the column or clip mid-glyph.
        FittedBox(
          fit: BoxFit.scaleDown,
          alignment: AlignmentDirectional.centerStart,
          child: MoneyText(
            amount: amount,
            style: theme.textTheme.titleMedium?.copyWith(
              color: color,
              fontWeight: FontWeight.w700,
            ),
          ),
        ),
      ],
    );
  }
}

/// Slow-moving building facts in one glanceable 2×2 grid — raw counts the
/// manager checks rarely, so they share a surface instead of shouting.
class _FactsCard extends StatelessWidget {
  const _FactsCard({required this.dash});
  final BuildingDashboard dash;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(AppTheme.spaceL),
        child: Column(
          children: [
            Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Expanded(
                  child: _FactCell(
                    icon: Icons.apartment_outlined,
                    label: l10n.dashboardUnitCount,
                    value: dash.unitCount,
                  ),
                ),
                const SizedBox(width: AppTheme.spaceM),
                Expanded(
                  child: _FactCell(
                    icon: Icons.people_outline,
                    label: l10n.dashboardOccupiedUnits,
                    value: dash.occupiedUnitCount,
                  ),
                ),
              ],
            ),
            const Divider(
              height: AppTheme.spaceXl,
              indent: AppTheme.spaceM,
              endIndent: AppTheme.spaceM,
            ),
            Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Expanded(
                  child: _FactCell(
                    icon: Icons.build_outlined,
                    label: l10n.dashboardOpenRequests,
                    value: dash.openRequests,
                  ),
                ),
                const SizedBox(width: AppTheme.spaceM),
                Expanded(
                  child: _FactCell(
                    icon: Icons.receipt_outlined,
                    label: l10n.dashboardPendingExpenses,
                    value: dash.pendingExpenses,
                  ),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }
}

class _FactCell extends StatelessWidget {
  const _FactCell({
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
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          children: [
            Icon(icon, size: 18, color: scheme.onSurfaceVariant),
            const SizedBox(width: AppTheme.spaceS),
            Text(
              toPersianDigits(value.toString()),
              style: theme.textTheme.titleMedium?.copyWith(
                fontWeight: FontWeight.w800,
              ),
            ),
          ],
        ),
        const SizedBox(height: 2),
        Text(
          label,
          maxLines: 2,
          overflow: TextOverflow.ellipsis,
          style: theme.textTheme.bodySmall,
        ),
      ],
    );
  }
}
