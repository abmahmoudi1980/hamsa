import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../core/l10n/app_localizations.dart';
import '../../../core/network/api_exception.dart';
import '../../../core/theme/app_theme.dart';
import '../models/people.dart';
import '../people_controller.dart';
import '../../../shared/widgets/save_bar.dart';

/// Person create/edit (US3/T040). Server 400/409 Persian messages surface
/// via [ApiException.serverMessage] (e.g. کد ملی checksum, FR-005).
class PersonFormScreen extends ConsumerStatefulWidget {
  const PersonFormScreen({
    super.key,
    required this.buildingId,
    this.personId,
    this.existing,
  });

  final String buildingId;

  /// Edit mode: pass [existing] (pre-loaded) or [personId] (loaded on open).
  final String? personId;
  final Person? existing;

  @override
  ConsumerState<PersonFormScreen> createState() => _PersonFormScreenState();
}

class _PersonFormScreenState extends ConsumerState<PersonFormScreen> {
  final _formKey = GlobalKey<FormState>();
  final _name = TextEditingController();
  final _phone = TextEditingController();
  final _nationalId = TextEditingController();
  String? _apiError;
  bool _saving = false;
  Person? _loaded;
  bool _loading = false;

  @override
  void initState() {
    super.initState();
    final e = widget.existing;
    if (e != null) {
      _populate(e);
    } else if (widget.personId != null) {
      _loading = true;
      _loadExisting();
    }
  }

  void _populate(Person p) {
    _loaded = p;
    _name.text = p.fullName;
    _phone.text = p.phone ?? '';
    _nationalId.text = p.nationalId ?? '';
  }

  Future<void> _loadExisting() async {
    try {
      final (people, _) = await ref
          .read(peopleRepositoryProvider)
          .listPersons(widget.buildingId);
      Person? match;
      for (final p in people) {
        if (p.id == widget.personId) match = p;
      }
      if (match != null && mounted) {
        setState(() {
          _populate(match!);
          _loading = false;
        });
      }
    } on ApiException catch (e) {
      if (mounted) {
        setState(() {
          _apiError = e.serverMessage ?? e.code;
          _loading = false;
        });
      }
    }
  }

  @override
  void dispose() {
    _name.dispose();
    _phone.dispose();
    _nationalId.dispose();
    super.dispose();
  }

  Future<void> _save() async {
    if (!_formKey.currentState!.validate()) return;
    setState(() {
      _saving = true;
      _apiError = null;
    });
    try {
      await ref
          .read(peopleControllerProvider(widget.buildingId).notifier)
          .save(
            buildingId: widget.buildingId,
            existing: _loaded,
            payload: {
              'full_name': _name.text.trim(),
              if (_phone.text.trim().isNotEmpty) 'phone': _phone.text.trim(),
              if (_nationalId.text.trim().isNotEmpty)
                'national_id': _nationalId.text.trim(),
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
    final isEdit = _loaded != null || widget.personId != null;
    if (_loading) {
      return const Scaffold(body: Center(child: CircularProgressIndicator()));
    }
    return Scaffold(
      appBar: AppBar(title: Text(isEdit ? l10n.editPerson : l10n.addPerson)),
      body: SafeArea(
        child: Form(
          key: _formKey,
          child: ListView(
            padding: AppTheme.pagePadding,
            children: [
              TextFormField(
                controller: _name,
                decoration: InputDecoration(labelText: l10n.personName),
                validator: (v) =>
                    v == null || v.trim().isEmpty ? l10n.requiredField : null,
              ),
              const SizedBox(height: AppTheme.spaceM),
              TextFormField(
                controller: _phone,
                keyboardType: TextInputType.phone,
                inputFormatters: [FilteringTextInputFormatter.digitsOnly],
                maxLength: 11,
                decoration: InputDecoration(
                  labelText: l10n.personPhone,
                  counterText: '',
                ),
              ),
              const SizedBox(height: AppTheme.spaceM),
              TextFormField(
                controller: _nationalId,
                keyboardType: TextInputType.number,
                inputFormatters: [FilteringTextInputFormatter.digitsOnly],
                maxLength: 10,
                decoration: InputDecoration(
                  labelText: l10n.personNationalId,
                  counterText: '',
                ),
                validator: (v) {
                  final t = v?.trim() ?? '';
                  if (t.isNotEmpty && t.length != 10) {
                    return l10n.invalidNationalId;
                  }
                  return null;
                },
              ),
              if (_apiError != null) ...[
                const SizedBox(height: AppTheme.spaceM),
                Text(
                  _apiError!,
                  style: TextStyle(color: Theme.of(context).colorScheme.error),
                  textAlign: TextAlign.center,
                ),
              ],
            ],
          ),
        )
      ),
      bottomNavigationBar: SaveBar(
        child: FilledButton(
          onPressed: _saving ? null : _save,
          child: Text(l10n.save),
        ),
      ),
    );
  }
}
