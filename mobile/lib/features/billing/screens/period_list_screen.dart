import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../core/datetime/jalali.dart';
import '../../../core/l10n/app_localizations.dart';
import '../../../core/network/api_exception.dart';
import '../../../core/theme/app_theme.dart';
import '../../../shared/widgets/empty_state.dart';
import '../../../shared/widgets/status_chip.dart';
import '../billing_controller.dart';
import '../models/billing.dart';

/// Manager period list (US4/T050): every billing period of the building;
/// tap → detail/calculate/preview; FAB → create form.
class PeriodListScreen extends ConsumerWidget {
  const PeriodListScreen({super.key, required this.buildingId});

  final String buildingId;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final async = ref.watch(periodsControllerProvider(buildingId));
    final scheme = Theme.of(context).colorScheme;

    return Scaffold(
      appBar: AppBar(title: Text(l10n.billingTitle)),
      floatingActionButton: FloatingActionButton.extended(
        onPressed: () => context.push('/manager/periods/$buildingId/new'),
        icon: const Icon(Icons.add),
        label: Text(l10n.addPeriod),
      ),
      body: async.when(
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (e, _) => EmptyState(
          icon: Icons.error_outline,
          title: _msg(l10n, e),
        ),
        data: (periods) => periods.isEmpty
            ? EmptyState(
                title: l10n.billingTitle,
                subtitle: l10n.emptyStateSubtitle,
                actionLabel: l10n.addPeriod,
                onAction: () => context.push('/manager/periods/$buildingId/new'),
              )
            : RefreshIndicator(
                onRefresh: () async =>
                    ref.invalidate(periodsControllerProvider(buildingId)),
                child: ListView.separated(
                  padding: AppTheme.pagePadding,
                  itemCount: periods.length,
                  separatorBuilder: (_, _) => const SizedBox(height: 10),
                  itemBuilder: (context, i) {
                    final p = periods[i];
                    return Card(
                      child: ListTile(
                        leading: Container(
                          width: 44,
                          height: 44,
                          decoration: BoxDecoration(
                            color: scheme.primary.withValues(alpha: 0.1),
                            borderRadius: BorderRadius.circular(
                              AppTheme.radiusSmall,
                            ),
                          ),
                          child: const Icon(Icons.event_note_outlined, size: 22),
                        ),
                        title: Text(p.title),
                        subtitle: Text(
                          '${formatJalaliDate(_parseIso(p.startDate))}'
                          ' — ${formatJalaliDate(_parseIso(p.endDate))}',
                        ),
                        trailing:
                            StatusChip(kind: StatusKind.period, value: p.status),
                        onTap: () => context
                            .push('/manager/periods/$buildingId/${p.id}'),
                      ),
                    );
                  },
                ),
              ),
      ),
    );
  }
}

DateTime _parseIso(String iso) => DateTime.tryParse(iso) ?? DateTime(2000);

String _msg(AppLocalizations l10n, Object e) =>
    e is ApiException ? (e.serverMessage ?? l10n.errorServer) : l10n.errorUnknown;

/// Shared payload builder for the cost-item form (T051).
Map<String, dynamic> costItemPayload({
  required String title,
  required String method,
  int? totalAmount,
  int? fixedAmountPerUnit,
  List<ComboWeight>? comboWeights,
  bool includeVacant = false,
  List<String> unitIds = const [],
}) =>
    {
      'title': title,
      'method': method,
      if (totalAmount != null) 'total_amount': '$totalAmount',
      if (fixedAmountPerUnit != null)
        'fixed_amount_per_unit': '$fixedAmountPerUnit',
      if (comboWeights != null)
        'combo_weights': comboWeights.map((w) => w.toPayload()).toList(),
      'include_vacant': includeVacant,
      if (method == 'specific_units') 'unit_ids': unitIds,
    };
