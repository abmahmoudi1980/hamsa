import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../core/datetime/jalali.dart';
import '../../../core/l10n/app_localizations.dart';
import '../../../core/theme/app_theme.dart';
import '../../../shared/formatters/money_text.dart';
import '../../../shared/widgets/empty_state.dart';
import '../../../shared/widgets/menu_card.dart';
import '../../../shared/widgets/status_chip.dart';
import '../../dashboard/dashboard_controller.dart';
import '../../dashboard/models/dashboard.dart';

/// US9 (T083) — Resident home: financial card + maintenance summary +
/// announcements summary with unread badge. The single `GET /me/home` call
/// powers every section so the panel renders in one round trip.
class ResidentHomeScreen extends ConsumerWidget {
  const ResidentHomeScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final async = ref.watch(residentHomeControllerProvider);

    return Scaffold(
      appBar: AppBar(title: Text(l10n.homeTitle)),
      body: async.when(
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (_, _) => Center(
          child: EmptyState(
            icon: Icons.error_outline,
            title: l10n.errorServer,
          ),
        ),
        data: (home) => RefreshIndicator(
          onRefresh: () =>
              ref.read(residentHomeControllerProvider.notifier).refresh(),
          child: ListView(
            padding: AppTheme.pagePadding,
            children: [
              _FinancialCard(home: home),
              const SizedBox(height: AppTheme.spaceL),
              _RequestsCard(home: home),
              const SizedBox(height: AppTheme.spaceL),
              _AnnouncementsCard(home: home),
              const SizedBox(height: AppTheme.spaceXxl),
              // Quick menu entries — the US9 spec lists these alongside the
              // home summary so residents reach charges/payments/maintenance
              // without re-navigating through the bottom nav.
              MenuCard(
                icon: Icons.receipt_long,
                title: l10n.myCharges,
                onTap: () => context.push('/home/charges'),
              ),
              const SizedBox(height: AppTheme.spaceM),
              MenuCard(
                icon: Icons.payments_outlined,
                title: l10n.paymentHistoryTitle,
                onTap: () => context.push('/home/payments'),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _FinancialCard extends ConsumerWidget {
  const _FinancialCard({required this.home});
  final HomeSummary home;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final theme = Theme.of(context);
    final scheme = theme.colorScheme;
    final invoice = home.latestInvoice;
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(AppTheme.spaceL),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            SectionHeader(title: l10n.homeFinancialCard),
            Row(
              crossAxisAlignment: CrossAxisAlignment.end,
              children: [
                Expanded(
                  child: MoneyText(
                    amount: home.payableAmount,
                    style: theme.textTheme.headlineSmall?.copyWith(
                      color: scheme.primary,
                      fontWeight: FontWeight.w800,
                    ),
                  ),
                ),
                if (home.payableAmount > 0 && invoice != null)
                  FilledButton.icon(
                    onPressed: () => context.push('/invoice/${invoice.id}/pay'),
                    icon: const Icon(Icons.payment),
                    label: Text(l10n.homePayInvoice),
                  ),
              ],
            ),
            const SizedBox(height: AppTheme.spaceS),
            Text(l10n.homePayableAmount, style: theme.textTheme.bodySmall),
            if (invoice != null) ...[
              const SizedBox(height: AppTheme.spaceM),
              InkWell(
                onTap: () => context.push('/invoice/${invoice.id}'),
                borderRadius: BorderRadius.circular(AppTheme.radiusSmall),
                child: Padding(
                  padding: const EdgeInsets.symmetric(
                    vertical: AppTheme.spaceS,
                  ),
                  child: Row(
                    children: [
                      Expanded(
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Text(
                              l10n.homeLatestInvoice,
                              style: theme.textTheme.bodySmall,
                            ),
                            Text(
                              invoice.periodTitle.isNotEmpty
                                  ? invoice.periodTitle
                                  : invoice.invoiceNumber,
                              style: theme.textTheme.titleMedium,
                            ),
                            if (invoice.hasDueDate)
                              Text(
                                l10n.homeDueDate(_formatDue(invoice.dueDate)),
                                style: theme.textTheme.bodySmall,
                              ),
                          ],
                        ),
                      ),
                      const SizedBox(width: AppTheme.spaceS),
                      StatusChip(
                          kind: StatusKind.invoice, value: invoice.status),
                    ],
                  ),
                ),
              ),
            ] else ...[
              const SizedBox(height: AppTheme.spaceM),
              Text(l10n.homeNoInvoice, style: theme.textTheme.bodyMedium),
            ],
          ],
        ),
      ),
    );
  }
}

class _RequestsCard extends StatelessWidget {
  const _RequestsCard({required this.home});
  final HomeSummary home;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final theme = Theme.of(context);
    final req = home.latestRequest;
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(AppTheme.spaceL),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            SectionHeader(title: l10n.homeRequestsCard),
            Text(
              l10n.homeOpenRequests(home.openRequestCount),
              style: theme.textTheme.titleMedium?.copyWith(
                color: theme.colorScheme.primary,
                fontWeight: FontWeight.w700,
              ),
            ),
            const SizedBox(height: AppTheme.spaceM),
            if (req == null)
              Text(l10n.homeNoOpenRequest, style: theme.textTheme.bodyMedium)
            else
              InkWell(
                onTap: () => context.push('/home/maintenance'),
                child: Row(
                  children: [
                    Expanded(
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text(l10n.homeLatestRequest,
                              style: theme.textTheme.bodySmall),
                          Text(req.title, style: theme.textTheme.titleMedium),
                        ],
                      ),
                    ),
                    const SizedBox(width: AppTheme.spaceS),
                    StatusChip(
                        kind: StatusKind.maintenance, value: req.status),
                  ],
                ),
              ),
          ],
        ),
      ),
    );
  }
}

class _AnnouncementsCard extends StatelessWidget {
  const _AnnouncementsCard({required this.home});
  final HomeSummary home;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final theme = Theme.of(context);
    final scheme = theme.colorScheme;
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(AppTheme.spaceL),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Expanded(child: SectionHeader(title: l10n.homeAnnouncementsCard)),
                if (home.unreadAnnouncementCount > 0)
                  Container(
                    padding: const EdgeInsets.symmetric(
                      horizontal: 10,
                      vertical: 4,
                    ),
                    decoration: BoxDecoration(
                      color: scheme.errorContainer,
                      borderRadius: BorderRadius.circular(999),
                    ),
                    child: Text(
                      l10n.homeUnreadAnnouncements(home.unreadAnnouncementCount),
                      style: theme.textTheme.labelSmall?.copyWith(
                        color: scheme.onErrorContainer,
                        fontWeight: FontWeight.w700,
                      ),
                    ),
                  ),
              ],
            ),
            const SizedBox(height: AppTheme.spaceS),
            if (home.latestAnnouncements.isEmpty)
              Text(l10n.emptyStateTitle, style: theme.textTheme.bodyMedium)
            else
              ...home.latestAnnouncements.take(5).map(
                    (a) => _AnnouncementRow(item: a),
                  ),
            const SizedBox(height: AppTheme.spaceM),
            Align(
              alignment: AlignmentDirectional.centerStart,
              child: TextButton.icon(
                onPressed: () => context.push('/home/announcements'),
                icon: const Icon(Icons.list_alt),
                label: Text(l10n.homeViewAnnouncements),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _AnnouncementRow extends StatelessWidget {
  const _AnnouncementRow({required this.item});
  final LatestAnnouncement item;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: AppTheme.spaceS),
      child: Row(
        children: [
          Icon(
            item.isRead ? Icons.mark_email_read : Icons.mark_email_unread,
            size: 18,
            color: item.isRead
                ? theme.colorScheme.onSurfaceVariant
                : theme.colorScheme.primary,
          ),
          const SizedBox(width: AppTheme.spaceS),
          Expanded(
            child: Text(
              item.title,
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
              style: theme.textTheme.bodyLarge,
            ),
          ),
        ],
      ),
    );
  }
}

String _formatDue(String iso) {
  try {
    final parts = iso.split('-');
    if (parts.length != 3) return iso;
    final dt = DateTime(
      int.parse(parts[0]),
      int.parse(parts[1]),
      int.parse(parts[2]),
    );
    return formatJalaliDate(dt);
  } catch (_) {
    return iso;
  }
}

// Helper kept so Announcement model export is not pruned (other call-sites
