import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../core/l10n/app_localizations.dart';
import '../../../core/network/api_exception.dart';
import '../../../core/theme/app_theme.dart';
import '../buildings_controller.dart';
import '../models/building.dart';
import '../../../shared/widgets/save_bar.dart';

/// Unit create/edit with all registry fields incl. parking/storage
/// (US2/T034). Duplicate numbers surface the server's 409 Persian message
/// (FR-003). Edit mode passes [existing] or [unitId] (loaded on first build).
class UnitFormScreen extends ConsumerStatefulWidget {
  const UnitFormScreen({
    super.key,
    required this.buildingId,
    this.existing,
    this.unitId,
  }) : assert(existing == null || unitId == null);

  final String buildingId;

  /// Pre-loaded unit (edit). Exactly one of [existing]/[unitId] for edits.
  final Unit? existing;
  final String? unitId;

  @override
  ConsumerState<UnitFormScreen> createState() => _UnitFormScreenState();
}

class _UnitFormScreenState extends ConsumerState<UnitFormScreen> {
  final _formKey = GlobalKey<FormState>();
  late String? _status = widget.existing?.status ?? 'active';
  String? _apiError;
  bool _saving = false;
  late final bool _loadingExisting =
      widget.existing == null && widget.unitId != null;
  // Populated after the (optional) load of the existing unit.
  final Map<String, TextEditingController> _c = {};

  late final Future<Unit?> _loadFuture = _load();

  Future<Unit?> _load() {
    if (widget.existing != null) return Future.value(widget.existing);
    if (widget.unitId == null) return Future.value(null);
    return ref.read(buildingsRepositoryProvider).getUnit(widget.unitId!);
  }

  void _populate(Unit u) {
    _c['number'] = TextEditingController(text: u.number);
    _c['block'] = TextEditingController(text: u.block ?? '');
    _c['floor'] = TextEditingController(text: '${u.floor}');
    _c['area'] = TextEditingController(text: '${u.areaM2}');
    _c['parkingCount'] = TextEditingController(text: '${u.parkingCount}');
    _c['parkingNumbers'] = TextEditingController(text: u.parkingNumbers ?? '');
    _c['storageCount'] = TextEditingController(text: '${u.storageCount}');
    _c['storageNumbers'] = TextEditingController(text: u.storageNumbers ?? '');
    _c['notes'] = TextEditingController(text: u.notes ?? '');
    _status = u.status;
  }

  TextEditingController _ctrl(String key, [String init = '']) =>
      _c.putIfAbsent(key, () => TextEditingController(text: init));

  Future<void> _save(Unit existing) async {
    if (!_formKey.currentState!.validate()) return;
    setState(() {
      _saving = true;
      _apiError = null;
    });
    try {
      await ref
          .read(unitsControllerProvider(widget.buildingId).notifier)
          .saveUnit(
            buildingId: widget.buildingId,
            existing: existing.id.isEmpty ? null : existing,
            payload: {
              'number': _ctrl('number').text.trim(),
              if (_ctrl('block').text.trim().isNotEmpty)
                'block': _ctrl('block').text.trim(),
              'floor': int.tryParse(_ctrl('floor').text) ?? 0,
              'area_m2': int.tryParse(_ctrl('area').text) ?? 0,
              if (_ctrl('parkingCount').text.isNotEmpty)
                'parking_count': int.tryParse(_ctrl('parkingCount').text),
              if (_ctrl('parkingNumbers').text.trim().isNotEmpty)
                'parking_numbers': _ctrl('parkingNumbers').text.trim(),
              if (_ctrl('storageCount').text.isNotEmpty)
                'storage_count': int.tryParse(_ctrl('storageCount').text),
              if (_ctrl('storageNumbers').text.trim().isNotEmpty)
                'storage_numbers': _ctrl('storageNumbers').text.trim(),
              'status': _status,
              if (_ctrl('notes').text.trim().isNotEmpty)
                'notes': _ctrl('notes').text.trim(),
            },
          );
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
    return FutureBuilder<Unit?>(
      future: _loadFuture,
      builder: (context, snap) {
        final loading =
            _loadingExisting && snap.connectionState != ConnectionState.done;
        final existing = snap.data ??
            Unit(id: '', buildingId: '', number: '', areaM2: 0);
        // Initialize the controllers exactly once, whether the unit came
        // from the constructor, the network (edit-by-id), or fresh-create.
        if (snap.connectionState == ConnectionState.done && _c.isEmpty) {
          _populate(existing);
        }
        final appBar = AppBar(
          title: Text(
            _loadingExisting || widget.existing != null
                ? l10n.editUnit
                : l10n.addUnit,
          ),
          actions: [
            if (_loadingExisting || widget.existing != null)
              IconButton(
                icon: const Icon(Icons.history),
                tooltip: l10n.changeHistory,
                onPressed: () => context.push(
                  '/manager/buildings/${widget.buildingId}'
                  '/units/${widget.unitId ?? widget.existing!.id}/history',
                ),
              ),
            IconButton(
              icon: const Icon(Icons.groups_outlined),
              tooltip: l10n.occupancyTitle,
              onPressed: () => context.push(
                '/manager/buildings/${widget.buildingId}'
                '/units/${widget.unitId ?? widget.existing!.id}/occupancy',
              ),
            ),
          ],
        );
        return Scaffold(
          appBar: appBar,
          body: loading
              ? const Center(child: CircularProgressIndicator())
              : _form(context, l10n),
          bottomNavigationBar: loading
              ? null
              : SaveBar(
                  child: FilledButton(
                    onPressed: _saving ? null : () => _save(existing),
                    child: Text(l10n.save),
                  ),
                ),
        );
      },
    );
  }

  Widget _form(BuildContext context, AppLocalizations l10n) {
    return Form(
      key: _formKey,
      child: ListView(
        padding: AppTheme.pagePadding,
        children: [
          TextFormField(
            controller: _ctrl('number'),
            decoration: InputDecoration(labelText: l10n.unitNumber),
            validator: (v) => v == null || v.trim().isEmpty
                ? l10n.requiredField
                : null,
          ),
          const SizedBox(height: AppTheme.spaceM),
          TextFormField(
            controller: _ctrl('block'),
            decoration: InputDecoration(labelText: l10n.unitBlock),
          ),
          const SizedBox(height: AppTheme.spaceM),
          TextFormField(
            controller: _ctrl('floor'),
            keyboardType: TextInputType.number,
            inputFormatters: [FilteringTextInputFormatter.digitsOnly],
            decoration: InputDecoration(labelText: l10n.unitFloor),
          ),
          const SizedBox(height: AppTheme.spaceM),
          TextFormField(
            controller: _ctrl('area'),
            keyboardType: TextInputType.number,
            inputFormatters: [FilteringTextInputFormatter.digitsOnly],
            decoration: InputDecoration(labelText: l10n.areaM2),
            validator: (v) {
              final n = int.tryParse(v ?? '');
              if (n == null || n <= 0) return l10n.invalidAmount;
              return null;
            },
          ),
          const SizedBox(height: AppTheme.spaceM),
          DropdownButtonFormField<String>(
            initialValue: _status,
            decoration: InputDecoration(labelText: l10n.unitStatus),
            items: [
              DropdownMenuItem(
                value: 'active',
                child: Text(l10n.statusActive),
              ),
              DropdownMenuItem(
                value: 'vacant',
                child: Text(l10n.statusVacant),
              ),
              DropdownMenuItem(
                value: 'occupied',
                child: Text(l10n.statusOccupied),
              ),
              DropdownMenuItem(
                value: 'inactive',
                child: Text(l10n.statusInactive),
              ),
            ],
            onChanged: (v) => setState(() => _status = v ?? 'active'),
          ),
          const SizedBox(height: AppTheme.spaceM),
          Row(
            children: [
              Expanded(
                child: TextFormField(
                  controller: _ctrl('parkingCount'),
                  keyboardType: TextInputType.number,
                  inputFormatters: [
                    FilteringTextInputFormatter.digitsOnly,
                  ],
                  decoration: InputDecoration(
                    labelText: l10n.parkingCount,
                  ),
                ),
              ),
              const SizedBox(width: AppTheme.spaceS),
              Expanded(
                child: TextFormField(
                  controller: _ctrl('parkingNumbers'),
                  decoration: InputDecoration(
                    labelText: l10n.parkingNumbers,
                  ),
                ),
              ),
            ],
          ),
          const SizedBox(height: AppTheme.spaceM),
          Row(
            children: [
              Expanded(
                child: TextFormField(
                  controller: _ctrl('storageCount'),
                  keyboardType: TextInputType.number,
                  inputFormatters: [
                    FilteringTextInputFormatter.digitsOnly,
                  ],
                  decoration: InputDecoration(
                    labelText: l10n.storageCount,
                  ),
                ),
              ),
              const SizedBox(width: AppTheme.spaceS),
              Expanded(
                child: TextFormField(
                  controller: _ctrl('storageNumbers'),
                  decoration: InputDecoration(
                    labelText: l10n.storageNumbers,
                  ),
                ),
              ),
            ],
          ),
          const SizedBox(height: AppTheme.spaceM),
          TextFormField(
            controller: _ctrl('notes'),
            maxLines: 3,
            decoration: InputDecoration(labelText: l10n.notesLabel),
          ),
          if (_apiError != null) ...[
            const SizedBox(height: AppTheme.spaceM),
            Text(
              _apiError!,
              style: TextStyle(
                color: Theme.of(context).colorScheme.error,
              ),
              textAlign: TextAlign.center,
            ),
          ],
        ],
      ),
    );
  }
}
