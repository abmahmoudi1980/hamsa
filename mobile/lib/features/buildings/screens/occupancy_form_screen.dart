import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../core/datetime/jalali.dart';
import '../../../core/l10n/app_localizations.dart';
import '../../../core/network/api_exception.dart';
import '../models/people.dart';
import '../people_controller.dart';

/// Add person↔unit occupancy (US3/T041): person picker, relationship
/// selector, and the Jalali start date (T018). Server closes the previous
/// active tenant (FR-007) and returns Persian 409/400 messages.
class OccupancyFormScreen extends ConsumerStatefulWidget {
  const OccupancyFormScreen({
    super.key,
    required this.buildingId,
    required this.unitId,
  });

  final String buildingId;
  final String unitId;

  @override
  ConsumerState<OccupancyFormScreen> createState() =>
      _OccupancyFormScreenState();
}

class _OccupancyFormScreenState extends ConsumerState<OccupancyFormScreen> {
  Person? _person;
  String _relationship = 'tenant';
  DateTime? _startDate;
  String? _apiError;
  bool _saving = false;

  Future<void> _save(AppLocalizations l10n) async {
    if (_person == null || _startDate == null) {
      setState(() => _apiError = l10n.requiredField);
      return;
    }
    setState(() {
      _saving = true;
      _apiError = null;
    });
    try {
      await ref
          .read(occupancyControllerProvider(widget.unitId).notifier)
          .addOccupancy({
            'person_id': _person!.id,
            'relationship': _relationship,
            'start_date': isoDate(_startDate!),
          });
      if (mounted) context.pop();
    } on ApiException catch (e) {
      setState(() => _apiError = e.serverMessage ?? e.code);
    } finally {
      if (mounted) setState(() => _saving = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final peopleAsync = ref.watch(peopleControllerProvider(widget.buildingId));

    String relationshipLabel(String r) => switch (r) {
      'tenant' => l10n.relTenant,
      'non_resident_owner' => l10n.relNonResidentOwner,
      _ => l10n.relOwner,
    };

    return Scaffold(
      appBar: AppBar(title: Text(l10n.addOccupancy)),
      body: peopleAsync.when(
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (e, _) => Center(child: Text(l10n.errorUnknown)),
        data: (people) => ListView(
          padding: const EdgeInsets.all(16),
          children: [
            DropdownButtonFormField<Person>(
              initialValue: _person,
              decoration: InputDecoration(labelText: l10n.selectPerson),
              items: people
                  .map(
                    (p) => DropdownMenuItem(value: p, child: Text(p.fullName)),
                  )
                  .toList(),
              onChanged: (p) => setState(() => _person = p),
              validator: (p) => p == null ? l10n.requiredField : null,
            ),
            Padding(
              padding: const EdgeInsets.only(top: 12),
              child: DropdownButtonFormField<String>(
                initialValue: _relationship,
                decoration: InputDecoration(labelText: l10n.relationshipLabel),
                items: occupancyRelationships
                    .map(
                      (r) => DropdownMenuItem(
                        value: r,
                        child: Text(relationshipLabel(r)),
                      ),
                    )
                    .toList(),
                onChanged: (r) => setState(() => _relationship = r ?? 'tenant'),
              ),
            ),
            Padding(
              padding: const EdgeInsets.only(top: 12),
              child: JalaliDatePickerField(
                label: l10n.occupancyStart,
                onChanged: (d) => setState(() => _startDate = d),
                validator: (d) => d == null ? l10n.requiredField : null,
              ),
            ),
            if (_apiError != null) ...[
              const SizedBox(height: 12),
              Text(
                _apiError!,
                style: TextStyle(color: Theme.of(context).colorScheme.error),
                textAlign: TextAlign.center,
              ),
            ],
            const SizedBox(height: 24),
            FilledButton(
              onPressed: _saving ? null : () => _save(l10n),
              child: Text(l10n.save),
            ),
          ],
        ),
      ),
    );
  }
}
