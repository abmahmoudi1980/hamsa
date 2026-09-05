import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../core/datetime/jalali.dart';
import '../../../core/l10n/app_localizations.dart';
import '../../../core/network/api_exception.dart';
import '../../../core/theme/app_theme.dart';
import '../../../shared/widgets/confirm_dialog.dart';
import '../../../shared/widgets/empty_state.dart';
import '../../../shared/widgets/menu_card.dart';
import '../models/people.dart';
import '../people_controller.dart';

/// Unit occupancy tab (US3/T041): dated person↔unit links with
/// end-dating, plus the independent occupant-count editor with history
/// (FR-007/FR-008). All dates go through the JalaliDatePickerField (T018).
class UnitOccupancyScreen extends ConsumerWidget {
  const UnitOccupancyScreen({
    super.key,
    required this.buildingId,
    required this.unitId,
  });

  final String buildingId;
  final String unitId;

  String _relationshipLabel(AppLocalizations l10n, String r) => switch (r) {
    'tenant' => l10n.relTenant,
    'non_resident_owner' => l10n.relNonResidentOwner,
    _ => l10n.relOwner,
  };

  Future<void> _confirmEnd(
    BuildContext context,
    WidgetRef ref,
    AppLocalizations l10n,
    Occupancy occ,
  ) async {
    DateTime? picked;
    final confirmed = await ConfirmDialog.show(
      context,
      title: l10n.endOccupancy,
      message: l10n.archiveConfirm,
      content: JalaliDatePickerField(
        label: l10n.occupancyStart,
        onChanged: (d) => picked = d,
      ),
    );
    if (confirmed == true && picked != null && context.mounted) {
      try {
        await ref
            .read(occupancyControllerProvider(unitId).notifier)
            .endOccupancy(occ.id, isoDate(picked!));
      } on ApiException catch (e) {
        if (context.mounted) {
          ScaffoldMessenger.of(
            context,
          ).showSnackBar(SnackBar(content: Text(e.serverMessage ?? e.code)));
        }
      }
    }
  }

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final async = ref.watch(occupancyControllerProvider(unitId));
    final scheme = Theme.of(context).colorScheme;

    return Scaffold(
      appBar: AppBar(title: Text(l10n.occupancyTitle)),
      floatingActionButton: FloatingActionButton.extended(
        onPressed: () async {
          await context.push(
            '/manager/buildings/$buildingId/units/$unitId/occupancy/new',
          );
          ref.invalidate(occupancyControllerProvider(unitId));
        },
        icon: const Icon(Icons.add),
        label: Text(l10n.addOccupancy),
      ),
      body: SafeArea(
        child: async.when(
          loading: () => const Center(child: CircularProgressIndicator()),
          error: (e, _) =>
              EmptyState(icon: Icons.error_outline, title: l10n.errorUnknown),
          data: (s) => ListView(
            padding: AppTheme.pagePadding,
            children: [
              SectionHeader(title: l10n.occupancyTitle),
              if (s.occupancies.isEmpty)
                EmptyState(
                  title: l10n.emptyStateTitle,
                  actionLabel: l10n.addOccupancy,
                  onAction: () async {
                    await context.push(
                      '/manager/buildings/$buildingId/units/$unitId/occupancy/new',
                    );
                    ref.invalidate(occupancyControllerProvider(unitId));
                  },
                )
              else
                ...s.occupancies.map(
                  (o) => Card(
                    child: ListTile(
                      title: Text(o.person?.fullName ?? o.personId),
                      subtitle: Text(
                        '${_relationshipLabel(l10n, o.relationship)}'
                        ' • ${formatJalaliDate(DateTime.parse(o.startDate))}'
                        '${o.endDate == null ? '' : ' — ${formatJalaliDate(DateTime.parse(o.endDate!))}'}',
                      ),
                      trailing: o.isActive
                          ? Row(
                              mainAxisSize: MainAxisSize.min,
                              children: [
                                Chip(label: Text(l10n.occupancyActive)),
                                IconButton(
                                  icon: const Icon(Icons.event_busy),
                                  tooltip: l10n.endOccupancy,
                                  onPressed: () =>
                                      _confirmEnd(context, ref, l10n, o),
                                ),
                              ],
                            )
                          : Chip(label: Text(l10n.occupancyEnded)),
                    ),
                  ),
                ),
              const SizedBox(height: AppTheme.spaceXl),
              SectionHeader(
                title: l10n.occupantCountTitle,
                action: TextButton.icon(
                  icon: const Icon(Icons.add),
                  label: Text(l10n.recordOccupantCount),
                  onPressed: () => _recordCount(context, ref, l10n),
                ),
              ),
              if (s.counts.isEmpty)
                EmptyState(
                  title: l10n.emptyStateTitle,
                  actionLabel: l10n.recordOccupantCount,
                  onAction: () => _recordCount(context, ref, l10n),
                )
              else
                ...s.counts.map(
                  (c) => Card(
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
                        child: const Icon(Icons.groups_outlined, size: 22),
                      ),
                      title: Text(toPersianDigits('${c.count}')),
                      subtitle: Text(
                        '${l10n.effectiveFrom}: '
                        '${formatJalaliDate(DateTime.parse(c.effectiveFrom))}',
                      ),
                    ),
                  ),
                ),
            ],
          ),
        )
      ),
    );
  }

  Future<void> _recordCount(
    BuildContext context,
    WidgetRef ref,
    AppLocalizations l10n,
  ) async {
    final countCtrl = TextEditingController();
    DateTime? picked;
    final confirmed = await ConfirmDialog.show(
      context,
      title: l10n.recordOccupantCount,
      content: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          TextField(
            controller: countCtrl,
            keyboardType: TextInputType.number,
            inputFormatters: [FilteringTextInputFormatter.digitsOnly],
            decoration: InputDecoration(labelText: l10n.occupantCountTitle),
          ),
          const SizedBox(height: AppTheme.spaceM),
          JalaliDatePickerField(
            label: l10n.effectiveFrom,
            onChanged: (d) => picked = d,
          ),
        ],
      ),
    );
    final count = int.tryParse(countCtrl.text);
    if (confirmed == true &&
        picked != null &&
        count != null &&
        context.mounted) {
      try {
        await ref
            .read(occupancyControllerProvider(unitId).notifier)
            .recordCount(count: count, effectiveFrom: isoDate(picked!));
      } on ApiException catch (e) {
        if (context.mounted) {
          ScaffoldMessenger.of(
            context,
          ).showSnackBar(SnackBar(content: Text(e.serverMessage ?? e.code)));
        }
      }
    }
  }
}
