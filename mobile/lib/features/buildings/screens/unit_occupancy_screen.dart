import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../core/datetime/jalali.dart';
import '../../../core/l10n/app_localizations.dart';
import '../../../core/network/api_exception.dart';
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
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (dialogContext) => AlertDialog(
        title: Text(l10n.endOccupancy),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Text(l10n.archiveConfirm),
            const SizedBox(height: 12),
            JalaliDatePickerField(
              label: l10n.occupancyStart,
              onChanged: (d) => picked = d,
            ),
          ],
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(dialogContext, false),
            child: Text(l10n.cancel),
          ),
          FilledButton(
            onPressed: () => Navigator.pop(dialogContext, true),
            child: Text(l10n.confirm),
          ),
        ],
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
      body: async.when(
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (e, _) => Center(child: Text(l10n.errorUnknown)),
        data: (s) => ListView(
          padding: const EdgeInsets.all(16),
          children: [
            Text(
              l10n.occupancyTitle,
              style: Theme.of(context).textTheme.titleMedium,
            ),
            if (s.occupancies.isEmpty)
              Padding(
                padding: const EdgeInsets.symmetric(vertical: 16),
                child: Text(l10n.emptyStateTitle),
              )
            else
              ...s.occupancies.map(
                (o) => ListTile(
                  contentPadding: EdgeInsets.zero,
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
            const Divider(height: 32),
            Row(
              children: [
                Expanded(
                  child: Text(
                    l10n.occupantCountTitle,
                    style: Theme.of(context).textTheme.titleMedium,
                  ),
                ),
                TextButton.icon(
                  icon: const Icon(Icons.add),
                  label: Text(l10n.recordOccupantCount),
                  onPressed: () => _recordCount(context, ref, l10n),
                ),
              ],
            ),
            if (s.counts.isEmpty)
              Padding(
                padding: const EdgeInsets.symmetric(vertical: 16),
                child: Text(l10n.emptyStateTitle),
              )
            else
              ...s.counts.map(
                (c) => ListTile(
                  contentPadding: EdgeInsets.zero,
                  leading: const Icon(Icons.groups_outlined),
                  title: Text(toPersianDigits('${c.count}')),
                  subtitle: Text(
                    '${l10n.effectiveFrom}: '
                    '${formatJalaliDate(DateTime.parse(c.effectiveFrom))}',
                  ),
                ),
              ),
          ],
        ),
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
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (dialogContext) => AlertDialog(
        title: Text(l10n.recordOccupantCount),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            TextField(
              controller: countCtrl,
              keyboardType: TextInputType.number,
              inputFormatters: [FilteringTextInputFormatter.digitsOnly],
              decoration: InputDecoration(labelText: l10n.occupantCountTitle),
            ),
            const SizedBox(height: 12),
            JalaliDatePickerField(
              label: l10n.effectiveFrom,
              onChanged: (d) => picked = d,
            ),
          ],
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(dialogContext, false),
            child: Text(l10n.cancel),
          ),
          FilledButton(
            onPressed: () => Navigator.pop(dialogContext, true),
            child: Text(l10n.confirm),
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
