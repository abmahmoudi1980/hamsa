import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../core/l10n/app_localizations.dart';
import '../../../shared/widgets/empty_state.dart';
import '../../../shared/widgets/status_chip.dart';
import '../maintenance_controller.dart';
import '../models/maintenance.dart';

/// T074 — manager request list for one building, filterable by status / priority / category.
/// Each row → detail (assignee, cost, status actions, close).
class ManagerMaintenanceListScreen extends ConsumerStatefulWidget {
  const ManagerMaintenanceListScreen({super.key, required this.buildingId});
  final String buildingId;

  @override
  ConsumerState<ManagerMaintenanceListScreen> createState() => _ManagerMaintenanceListScreenState();
}

class _ManagerMaintenanceListScreenState extends ConsumerState<ManagerMaintenanceListScreen> {
  String _status = '';
  String _priority = '';
  String _category = '';
  List<MaintenanceRequest>? _results;
  bool _loading = false;
  String? _error;

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
      final repo = ref.read(maintenanceRepositoryProvider);
      final (items, _) = await repo.buildingRequests(
        widget.buildingId,
        status: _status,
        priority: _priority,
        category: _category,
      );
      if (!mounted) return;
      setState(() => _results = items);
    } catch (e) {
      if (!mounted) return;
      setState(() => _error = l10n.maintenanceLoadError);
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return Scaffold(
      appBar: AppBar(title: Text(l10n.maintenanceTitle)),
      body: Column(
        children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(12, 8, 12, 0),
            child: Wrap(
              spacing: 8,
              runSpacing: 8,
              children: [
                DropdownButton<String>(
                  value: _status.isEmpty ? null : _status,
                  hint: Text(l10n.maintenanceFilterAllStatuses),
                  items: [
                    DropdownMenuItem(value: '', child: Text(l10n.maintenanceFilterAllStatuses)),
                    DropdownMenuItem(value: 'new', child: Text(l10n.maintenanceStatusNew)),
                    DropdownMenuItem(value: 'under_review', child: Text(l10n.maintenanceStatusUnderReview)),
                    DropdownMenuItem(value: 'in_progress', child: Text(l10n.maintenanceStatusInProgress)),
                    DropdownMenuItem(value: 'done', child: Text(l10n.maintenanceStatusDone)),
                    DropdownMenuItem(value: 'closed', child: Text(l10n.maintenanceStatusClosed)),
                  ],
                  onChanged: (v) {
                    setState(() => _status = v ?? '');
                    _load();
                  },
                ),
                DropdownButton<String>(
                  value: _priority.isEmpty ? null : _priority,
                  hint: Text(l10n.maintenanceFilterAllPriorities),
                  items: [
                    DropdownMenuItem(value: '', child: Text(l10n.maintenanceFilterAllPriorities)),
                    DropdownMenuItem(value: 'normal', child: Text(l10n.maintenancePriorityNormal)),
                    DropdownMenuItem(value: 'important', child: Text(l10n.maintenancePriorityImportant)),
                    DropdownMenuItem(value: 'urgent', child: Text(l10n.maintenancePriorityUrgent)),
                  ],
                  onChanged: (v) {
                    setState(() => _priority = v ?? '');
                    _load();
                  },
                ),
                DropdownButton<String>(
                  value: _category.isEmpty ? null : _category,
                  hint: Text(l10n.maintenanceFilterAllCategories),
                  items: [
                    DropdownMenuItem(value: '', child: Text(l10n.maintenanceFilterAllCategories)),
                    DropdownMenuItem(value: 'elevator', child: Text(l10n.maintenanceCategoryElevator)),
                    DropdownMenuItem(value: 'utilities', child: Text(l10n.maintenanceCategoryUtilities)),
                    DropdownMenuItem(value: 'electrical', child: Text(l10n.maintenanceCategoryElectrical)),
                    DropdownMenuItem(value: 'water', child: Text(l10n.maintenanceCategoryWater)),
                    DropdownMenuItem(value: 'cleaning', child: Text(l10n.maintenanceCategoryCleaning)),
                    DropdownMenuItem(value: 'common_area', child: Text(l10n.maintenanceCategoryCommonArea)),
                    DropdownMenuItem(value: 'parking', child: Text(l10n.maintenanceCategoryParking)),
                    DropdownMenuItem(value: 'other', child: Text(l10n.maintenanceCategoryOther)),
                  ],
                  onChanged: (v) {
                    setState(() => _category = v ?? '');
                    _load();
                  },
                ),
              ],
            ),
          ),
          const Divider(height: 1),
          Expanded(child: _buildBody()),
        ],
      ),
    );
  }

  Widget _buildBody() {
    final l10n = AppLocalizations.of(context);
    if (_loading) return const Center(child: CircularProgressIndicator());
    if (_error != null) {
      return Center(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Text(_error!),
            const SizedBox(height: 8),
            FilledButton(onPressed: _load, child: Text(l10n.retry)),
          ],
        ),
      );
    }
    final items = _results ?? const <MaintenanceRequest>[];
    if (items.isEmpty) {
      return EmptyState(title: l10n.maintenanceNoRequests, subtitle: l10n.maintenanceNoRequestsSubtitle);
    }
    return RefreshIndicator(
      onRefresh: _load,
      child: ListView.separated(
        padding: const EdgeInsets.all(12),
        itemCount: items.length,
        separatorBuilder: (_, __) => const SizedBox(height: 8),
        itemBuilder: (_, i) {
          final r = items[i];
          final priLabel = _priorityLabel(l10n, r.priority);
          return Card(
            child: ListTile(
              title: Text(r.title),
              subtitle: Text('${_categoryLabel(l10n, r.category)} • ${l10n.maintenancePriorityLabel}: $priLabel'),
              trailing: StatusChip(kind: StatusKind.maintenance, value: r.status),
              onTap: () => context.push('/manager/maintenance/${widget.buildingId}/${r.id}'),
            ),
          );
        },
      ),
    );
  }
}
