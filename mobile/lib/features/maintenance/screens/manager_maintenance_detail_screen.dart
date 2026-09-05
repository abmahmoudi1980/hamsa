import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../core/datetime/jalali.dart';
import '../../../core/l10n/app_localizations.dart';
import '../../../core/network/api_exception.dart';
import '../../../shared/formatters/toman_input.dart';
import '../../../shared/widgets/confirm_dialog.dart';
import '../../../shared/widgets/status_chip.dart';
import '../maintenance_controller.dart';
import '../models/maintenance.dart';

/// T074 — manager detail: shows request fields plus controls for assignee
/// (person picker), recorded cost, notes, status actions and close.
class ManagerMaintenanceDetailScreen extends ConsumerStatefulWidget {
  const ManagerMaintenanceDetailScreen({
    super.key,
    required this.buildingId,
    required this.requestId,
  });
  final String buildingId;
  final String requestId;

  @override
  ConsumerState<ManagerMaintenanceDetailScreen> createState() =>
      _ManagerMaintenanceDetailScreenState();
}

class _ManagerMaintenanceDetailScreenState
    extends ConsumerState<ManagerMaintenanceDetailScreen> {
  MaintenanceRequest? _item;
  bool _loading = true;
  String? _error;
  bool _saving = false;

  String? _assigneeId;
  final _costCtrl = TextEditingController();
  final _notesCtrl = TextEditingController();

  String _statusLabel(AppLocalizations l10n, String token) => switch (token) {
    'new' => l10n.maintenanceStatusNew,
    'under_review' => l10n.maintenanceStatusUnderReview,
    'in_progress' => l10n.maintenanceStatusInProgress,
    'done' => l10n.maintenanceStatusDone,
    'closed' => l10n.maintenanceStatusClosed,
    _ => token,
  };

  String _priorityLabel(AppLocalizations l10n, String token) => switch (token) {
    'normal' => l10n.maintenancePriorityNormal,
    'important' => l10n.maintenancePriorityImportant,
    'urgent' => l10n.maintenancePriorityUrgent,
    _ => token,
  };

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

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) => _load());
  }

  @override
  void dispose() {
    _costCtrl.dispose();
    _notesCtrl.dispose();
    super.dispose();
  }

  Future<void> _load() async {
    final l10n = AppLocalizations.of(context);
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final repo = ref.read(maintenanceRepositoryProvider);
      final item = await repo.getRequest(widget.requestId);
      if (!mounted) return;
      setState(() {
        _item = item;
        _assigneeId = item.assigneePersonId;
        _costCtrl.text = item.recordedCost?.toString() ?? '';
        _notesCtrl.text = item.notes ?? '';
      });
    } on ApiException catch (e) {
      if (!mounted) return;
      setState(() => _error = e.serverMessage ?? l10n.maintenanceLoadError);
    } catch (_) {
      if (!mounted) return;
      setState(() => _error = l10n.maintenanceLoadError);
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  Future<void> _saveMeta() async {
    final l10n = AppLocalizations.of(context);
    setState(() => _saving = true);
    try {
      final repo = ref.read(maintenanceRepositoryProvider);
      final raw = _costCtrl.text.trim().replaceAll(',', '').replaceAll('٬', '');
      final cost = raw.isEmpty ? null : int.tryParse(raw);
      final updated = await repo.updateRequest(
        widget.requestId,
        assigneePersonId: _assigneeId ?? '',
        recordedCost: cost,
        notes: _notesCtrl.text.trim(),
      );
      if (!mounted) return;
      setState(() => _item = updated);
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(SnackBar(content: Text(l10n.maintenanceSaved)));
    } on ApiException catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(e.serverMessage ?? l10n.maintenanceSaveError)),
      );
    } finally {
      if (mounted) setState(() => _saving = false);
    }
  }

  Future<void> _transition(String next) async {
    final l10n = AppLocalizations.of(context);
    final statusText = _statusLabel(l10n, next);
    final confirmed = await ConfirmDialog.show(
      context,
      title: l10n.maintenanceChangeStatus,
      message: l10n.maintenanceChangeStatusConfirm(statusText),
      confirmLabel: l10n.confirm,
    );
    if (confirmed != true) return;
    setState(() => _saving = true);
    try {
      final repo = ref.read(maintenanceRepositoryProvider);
      final updated = await repo.updateRequest(widget.requestId, status: next);
      if (!mounted) return;
      setState(() => _item = updated);
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(l10n.maintenanceStatusChangedTo(statusText))),
      );
    } on ApiException catch (e) {
      if (!mounted) return;
      final msg = e.statusCode == 409
          ? l10n.maintenanceStatusChangeNotAllowed
          : (e.serverMessage ?? l10n.maintenanceSaveError);
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(msg)));
    } finally {
      if (mounted) setState(() => _saving = false);
    }
  }

  List<String> _nextStatuses(String current) {
    return switch (current) {
      'new' => ['under_review'],
      'under_review' => ['in_progress', 'closed'],
      'in_progress' => ['done', 'closed'],
      'done' => ['closed'],
      _ => const [],
    };
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    if (_loading) {
      return Scaffold(
        appBar: AppBar(title: Text(l10n.maintenanceDetailTitle)),
        body: const SafeArea(
          child: Center(child: CircularProgressIndicator())
        ),
      );
    }
    if (_error != null || _item == null) {
      return Scaffold(
        appBar: AppBar(title: Text(l10n.maintenanceDetailTitle)),
        body: SafeArea(
          child: Center(
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                Text(_error ?? l10n.errorNotFound),
                const SizedBox(height: 8),
                SizedBox(
                  width: double.infinity,
                  child: FilledButton(onPressed: _load, child: Text(l10n.retry)),
                ),
              ],
            ),
          )
        ),
      );
    }
    final item = _item!;
    final next = _nextStatuses(item.status);
    return Scaffold(
      appBar: AppBar(title: Text(item.title)),
      body: SafeArea(
        child: ListView(
          padding: const EdgeInsets.all(16),
          children: [
            Row(
              children: [
                StatusChip(kind: StatusKind.maintenance, value: item.status),
                const SizedBox(width: 8),
                Chip(label: Text(_priorityLabel(l10n, item.priority))),
                const SizedBox(width: 8),
                Chip(label: Text(_categoryLabel(l10n, item.category))),
              ],
            ),
            const SizedBox(height: 12),
            if (item.createdAt != null)
              Text(
                l10n.maintenanceCreatedAt(_tryJalali(item.createdAt!)),
                style: Theme.of(context).textTheme.bodySmall,
              ),
            if (item.location != null && item.location!.isNotEmpty)
              Text(l10n.maintenanceLocation(item.location!)),
            if (item.description != null && item.description!.isNotEmpty) ...[
              const SizedBox(height: 8),
              Text(item.description!),
            ],
            const Divider(height: 32),
            if (next.isNotEmpty) ...[
              Text(
                l10n.maintenanceChangeStatus,
                style: Theme.of(context).textTheme.titleMedium,
              ),
              const SizedBox(height: 8),
              Wrap(
                spacing: 8,
                children: next
                    .map(
                      (s) => FilledButton(
                        onPressed: _saving ? null : () => _transition(s),
                        child: Text(_statusLabel(l10n, s)),
                      ),
                    )
                    .toList(),
              ),
              const SizedBox(height: 16),
            ] else
              Chip(label: Text(l10n.maintenanceClosedChip)),
            const Divider(height: 32),
            Text(
              l10n.maintenanceMetaTitle,
              style: Theme.of(context).textTheme.titleMedium,
            ),
            const SizedBox(height: 8),
            TextFormField(
              initialValue: _assigneeId ?? '',
              decoration: InputDecoration(
                labelText: l10n.maintenanceAssigneeLabel,
                hintText: l10n.maintenanceAssigneeHint,
              ),
              onChanged: (v) => _assigneeId = v.trim().isEmpty ? null : v.trim(),
            ),
            const SizedBox(height: 12),
            TextFormField(
              controller: _costCtrl,
              decoration: InputDecoration(
                labelText: l10n.maintenanceCostLabel,
                hintText: l10n.maintenanceCostHint,
              ),
              keyboardType: TextInputType.number,
              inputFormatters: const [TomanInputFormatter()],
            ),
            const SizedBox(height: 12),
            TextFormField(
              controller: _notesCtrl,
              decoration: InputDecoration(labelText: l10n.maintenanceNotesLabel),
              maxLines: 3,
            ),
            const SizedBox(height: 12),
            FilledButton.icon(
              onPressed: _saving ? null : _saveMeta,
              icon: _saving
                  ? const SizedBox(
                      width: 16,
                      height: 16,
                      child: CircularProgressIndicator(strokeWidth: 2),
                    )
                  : const Icon(Icons.save),
              label: Text(l10n.maintenanceSaveMeta),
            ),
            const SizedBox(height: 24),
            OutlinedButton.icon(
              onPressed: () => context.pop(),
              icon: const Icon(Icons.arrow_back),
              label: Text(l10n.maintenanceBack),
            ),
          ],
        )
      ),
    );
  }

  String _tryJalali(String iso) {
    try {
      return formatJalaliDate(DateTime.parse(iso));
    } catch (_) {
      return iso;
    }
  }
}
