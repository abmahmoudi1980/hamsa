import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../core/l10n/app_localizations.dart';
import '../../../core/network/api_exception.dart';
import '../buildings_controller.dart';
import '../models/building.dart';

/// Create/edit building (US2/T033). Pass [existing] for edit mode.
class BuildingFormScreen extends ConsumerStatefulWidget {
  const BuildingFormScreen({super.key, this.existing});

  final Building? existing;

  @override
  ConsumerState<BuildingFormScreen> createState() => _BuildingFormScreenState();
}

class _BuildingFormScreenState extends ConsumerState<BuildingFormScreen> {
  final _formKey = GlobalKey<FormState>();
  late final _name = TextEditingController(text: widget.existing?.name ?? '');
  late final _address =
      TextEditingController(text: widget.existing?.address ?? '');
  late final _blocks = TextEditingController(
    text: widget.existing == null ? '' : '${widget.existing!.blockCount}',
  );
  late final _floors = TextEditingController(
    text: widget.existing == null ? '' : '${widget.existing!.floorCount}',
  );
  late final _units = TextEditingController(
    text: widget.existing == null ? '' : '${widget.existing!.unitCount}',
  );
  late final _builtYear =
      TextEditingController(text: widget.existing?.builtYear?.toString() ?? '');
  late final _managerPhone =
      TextEditingController(text: widget.existing?.managerPhone ?? '');
  late final _emergencyPhone =
      TextEditingController(text: widget.existing?.emergencyPhone ?? '');
  late final _notes = TextEditingController(text: widget.existing?.notes ?? '');

  String? _apiError;
  bool _saving = false;

  int? _parseInt(TextEditingController c) => int.tryParse(c.text.trim());

  Future<void> _save() async {
    if (!_formKey.currentState!.validate()) return;
    setState(() {
      _saving = true;
      _apiError = null;
    });
    try {
      await ref
          .read(buildingsControllerProvider.notifier)
          .save(widget.existing, {
        'name': _name.text.trim(),
        if (_address.text.trim().isNotEmpty) 'address': _address.text.trim(),
        if (_parseInt(_blocks) != null) 'block_count': _parseInt(_blocks),
        if (_parseInt(_floors) != null) 'floor_count': _parseInt(_floors),
        if (_parseInt(_units) != null) 'unit_count': _parseInt(_units),
        if (_parseInt(_builtYear) != null) 'built_year': _parseInt(_builtYear),
        if (_managerPhone.text.trim().isNotEmpty)
          'manager_phone': _managerPhone.text.trim(),
        if (_emergencyPhone.text.trim().isNotEmpty)
          'emergency_phone': _emergencyPhone.text.trim(),
        if (_notes.text.trim().isNotEmpty) 'notes': _notes.text.trim(),
      });
      if (mounted) context.pop();
    } on ApiException catch (e) {
      setState(() => _apiError = e.serverMessage);
    } finally {
      if (mounted) setState(() => _saving = false);
    }
  }

  @override
  void dispose() {
    for (final c in [
      _name,
      _address,
      _blocks,
      _floors,
      _units,
      _builtYear,
      _managerPhone,
      _emergencyPhone,
      _notes,
    ]) {
      c.dispose();
    }
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return Scaffold(
      appBar: AppBar(
        title:
            Text(widget.existing == null ? l10n.addBuilding : l10n.editBuilding),
      ),
      body: Form(
        key: _formKey,
        child: ListView(
          padding: const EdgeInsets.all(16),
          children: [
            TextFormField(
              controller: _name,
              decoration: InputDecoration(labelText: l10n.buildingName),
              validator: (v) =>
                  v == null || v.trim().isEmpty ? l10n.requiredField : null,
            ),
            TextFormField(
              controller: _address,
              decoration: InputDecoration(labelText: l10n.buildingAddress),
            ),
            Row(children: [
              Expanded(
                child: _NumField(controller: _blocks, label: l10n.blockCount),
              ),
              const SizedBox(width: 8),
              Expanded(
                child: _NumField(controller: _floors, label: l10n.floorCount),
              ),
              const SizedBox(width: 8),
              Expanded(
                child:
                    _NumField(controller: _units, label: l10n.unitCountLabel),
              ),
            ]),
            TextFormField(
              controller: _builtYear,
              keyboardType: TextInputType.number,
              inputFormatters: [FilteringTextInputFormatter.digitsOnly],
              decoration: InputDecoration(labelText: l10n.builtYear),
            ),
            TextFormField(
              controller: _managerPhone,
              keyboardType: TextInputType.phone,
              decoration: InputDecoration(labelText: l10n.managerPhoneLabel),
            ),
            TextFormField(
              controller: _emergencyPhone,
              keyboardType: TextInputType.phone,
              decoration:
                  InputDecoration(labelText: l10n.emergencyPhoneLabel),
            ),
            TextFormField(
              controller: _notes,
              maxLines: 3,
              decoration: InputDecoration(labelText: l10n.notesLabel),
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
              onPressed: _saving ? null : _save,
              child: Text(l10n.save),
            ),
          ]
              .map((w) => Padding(
                    padding: const EdgeInsets.only(bottom: 12),
                    child: w,
                  ))
              .toList(),
        ),
      ),
    );
  }
}

class _NumField extends StatelessWidget {
  const _NumField({required this.controller, required this.label});

  final TextEditingController controller;
  final String label;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return TextFormField(
      controller: controller,
      keyboardType: TextInputType.number,
      inputFormatters: [FilteringTextInputFormatter.digitsOnly],
      decoration: InputDecoration(labelText: label),
      validator: (v) {
        if (v == null || v.isEmpty) return null; // optional counts
        final n = int.tryParse(v);
        if (n == null || n < 0) return l10n.invalidNumber;
        return null;
      },
    );
  }
}
