import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../core/datetime/jalali.dart';
import '../../../core/l10n/app_localizations.dart';
import '../../../shared/widgets/empty_state.dart';
import '../../../shared/widgets/status_chip.dart';
import '../../../shared/widgets/status_labels.dart';
import '../maintenance_controller.dart';
import '../models/maintenance.dart';

/// T073 — resident my-requests list with status timeline.
/// Submitting is via the FAB → /home/maintenance/new.
class ResidentMaintenanceListScreen extends ConsumerWidget {
  const ResidentMaintenanceListScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final async = ref.watch(myMaintenanceControllerProvider);
    return Scaffold(
      appBar: AppBar(title: Text(l10n.myMaintenanceTitle)),
      floatingActionButton: FloatingActionButton.extended(
        onPressed: () => context.push('/home/maintenance/new'),
        icon: const Icon(Icons.add),
        label: Text(l10n.addMaintenanceRequest),
      ),
      body: async.when(
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (e, _) => Center(child: Text(l10n.maintenanceLoadError)),
        data: (items) {
          if (items.isEmpty) {
            return EmptyState(
              title: l10n.maintenanceEmptyTitle,
              subtitle: l10n.maintenanceEmptySubtitle,
            );
          }
          return RefreshIndicator(
            onRefresh: () => ref.read(myMaintenanceControllerProvider.notifier).refresh(),
            child: ListView.separated(
              padding: const EdgeInsets.all(12),
              itemCount: items.length,
              separatorBuilder: (_, __) => const SizedBox(height: 8),
              itemBuilder: (_, i) => _Card(item: items[i]),
            ),
          );
        },
      ),
    );
  }
}

class _Card extends StatelessWidget {
  const _Card({required this.item});
  final MaintenanceRequest item;

  String _categoryLabel(AppLocalizations l10n, String token) => switch (token) {
        'elevator' => l10n.maintenanceCategoryElevator,
        'utilities' => l10n.maintenanceCategoryUtilities,
        'electrical' => l10n.maintenanceCategoryElectrical,
        'water' => l10n.maintenanceCategoryWater,
        'cleaning' => l10n.maintenanceCategoryCleaning,
        'common_area' => l10n.maintenanceCategoryCommonArea,
        'parking' => l10n.maintenanceCategoryParking,
        _ => l10n.maintenanceCategoryOther,
      };

  String _priorityLabel(AppLocalizations l10n, String token) => switch (token) {
        'normal' => l10n.maintenancePriorityNormal,
        'important' => l10n.maintenancePriorityImportant,
        'urgent' => l10n.maintenancePriorityUrgent,
        _ => token,
      };

  String _statusLabel(AppLocalizations l10n, String token) => switch (token) {
        'new' => l10n.maintenanceStatusNew,
        'under_review' => l10n.maintenanceStatusUnderReview,
        'in_progress' => l10n.maintenanceStatusInProgress,
        'done' => l10n.maintenanceStatusDone,
        'closed' => l10n.maintenanceStatusClosed,
        _ => token,
      };

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final priLabel = _priorityLabel(l10n, item.priority);
    final catLabel = _categoryLabel(l10n, item.category);
    String dateStr = '';
    if (item.createdAt != null) {
      try {
        dateStr = formatJalaliDate(DateTime.parse(item.createdAt!));
      } catch (_) {
        dateStr = item.createdAt!;
      }
    }

    return Card(
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Expanded(
                  child: Text(item.title, style: Theme.of(context).textTheme.titleMedium),
                ),
                StatusChip(kind: StatusKind.maintenance, value: item.status),
              ],
            ),
            const SizedBox(height: 6),
            Wrap(
              spacing: 8,
              children: [
                Chip(label: Text(catLabel)),
                Chip(label: Text(priLabel)),
                if (dateStr.isNotEmpty) Chip(label: Text(dateStr)),
              ],
            ),
            if (item.location != null && item.location!.isNotEmpty) ...[
              const SizedBox(height: 6),
              Row(
                children: [
                  const Icon(Icons.place_outlined, size: 16),
                  const SizedBox(width: 4),
                  Expanded(child: Text(l10n.maintenanceLocation(item.location!))),
                ],
              ),
            ],
            if (item.description != null && item.description!.isNotEmpty) ...[
              const SizedBox(height: 6),
              Text(item.description!, style: Theme.of(context).textTheme.bodySmall),
            ],
            const SizedBox(height: 8),
            // Status timeline (quick visual of all 5 states — highlight current).
            Row(
              children: ['new', 'under_review', 'in_progress', 'done', 'closed'].map((s) {
                final isPast = _statusOrder(item.status) >= _statusOrder(s);
                final isCurrent = item.status == s;
                return Expanded(
                  child: Column(
                    children: [
                      Container(
                        height: 6,
                        decoration: BoxDecoration(
                          color: isPast ? (maintenanceStatusColors[s] ?? Colors.grey) : Colors.grey.shade300,
                          borderRadius: BorderRadius.circular(3),
                        ),
                      ),
                      const SizedBox(height: 4),
                      Text(
                        _statusLabel(l10n, s),
                        textAlign: TextAlign.center,
                        style: TextStyle(
                          fontSize: 9,
                          fontWeight: isCurrent ? FontWeight.bold : FontWeight.normal,
                          color: isCurrent ? Theme.of(context).colorScheme.primary : Colors.grey.shade700,
                        ),
                      ),
                    ],
                  ),
                );
              }).toList(),
            ),
            if (item.recordedCost != null) ...[
              const SizedBox(height: 8),
              Text(
                l10n.maintenanceRecordedCost('${item.recordedCost}'),
                style: Theme.of(context).textTheme.bodySmall?.copyWith(color: Colors.green.shade700),
              ),
            ],
          ],
        ),
      ),
    );
  }
}

int _statusOrder(String s) => switch (s) {
      'new' => 0,
      'under_review' => 1,
      'in_progress' => 2,
      'done' => 3,
      'closed' => 4,
      _ => -1,
    };
