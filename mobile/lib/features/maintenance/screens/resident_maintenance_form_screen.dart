import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:image_picker/image_picker.dart';

import '../../../core/l10n/app_localizations.dart';
import '../../../core/network/api_exception.dart';
import '../../../shared/widgets/attachment_picker.dart';
import '../../../shared/widgets/save_bar.dart';
import '../maintenance_controller.dart';

/// T073 — resident submit form: title, category & priority Persian selectors,
/// description, location, optional photo attach. Mirrors the contract POST
/// /me/maintenance-requests (FR-028).
class ResidentMaintenanceFormScreen extends ConsumerStatefulWidget {
  const ResidentMaintenanceFormScreen({super.key});

  @override
  ConsumerState<ResidentMaintenanceFormScreen> createState() => _ResidentMaintenanceFormScreenState();
}

class _ResidentMaintenanceFormScreenState extends ConsumerState<ResidentMaintenanceFormScreen> {
  final _formKey = GlobalKey<FormState>();
  final _titleCtrl = TextEditingController();
  final _descCtrl = TextEditingController();
  final _locationCtrl = TextEditingController();
  String _category = 'other';
  String _priority = 'normal';
  XFile? _photo;
  bool _saving = false;

  @override
  void dispose() {
    _titleCtrl.dispose();
    _descCtrl.dispose();
    _locationCtrl.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    final l10n = AppLocalizations.of(context);
    if (!_formKey.currentState!.validate()) return;
    setState(() => _saving = true);
    try {
      final repo = ref.read(maintenanceRepositoryProvider);
      final photoId = _photo == null ? null : await repo.uploadPhoto(path: _photo!.path, name: _photo!.name);
      await repo.submit(
        title: _titleCtrl.text.trim(),
        category: _category,
        description: _descCtrl.text.trim().isEmpty ? null : _descCtrl.text.trim(),
        location: _locationCtrl.text.trim().isEmpty ? null : _locationCtrl.text.trim(),
        photoFileId: photoId,
        priority: _priority,
      );
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(l10n.maintenanceSubmitSuccess)));
      context.pop();
      ref.invalidate(myMaintenanceControllerProvider);
    } catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text(describeError(l10n, e))));
    } finally {
      if (mounted) setState(() => _saving = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return Scaffold(
      appBar: AppBar(title: Text(l10n.newMaintenanceTitle)),
      body: SafeArea(
        child: Form(
          key: _formKey,
          child: ListView(
            padding: const EdgeInsets.all(16),
            children: [
              TextFormField(
                controller: _titleCtrl,
                decoration: InputDecoration(labelText: l10n.maintenanceTitleLabel, hintText: l10n.maintenanceTitleHint),
                validator: (v) => (v == null || v.trim().isEmpty) ? l10n.maintenanceTitleRequired : null,
                maxLength: 150,
              ),
              const SizedBox(height: 12),
              DropdownButtonFormField<String>(
                initialValue: _category,
                decoration: InputDecoration(labelText: l10n.maintenanceCategoryLabel),
                items: [
                  DropdownMenuItem(value: 'elevator', child: Text(l10n.maintenanceCategoryElevator)),
                  DropdownMenuItem(value: 'utilities', child: Text(l10n.maintenanceCategoryUtilities)),
                  DropdownMenuItem(value: 'electrical', child: Text(l10n.maintenanceCategoryElectrical)),
                  DropdownMenuItem(value: 'water', child: Text(l10n.maintenanceCategoryWater)),
                  DropdownMenuItem(value: 'cleaning', child: Text(l10n.maintenanceCategoryCleaning)),
                  DropdownMenuItem(value: 'common_area', child: Text(l10n.maintenanceCategoryCommonArea)),
                  DropdownMenuItem(value: 'parking', child: Text(l10n.maintenanceCategoryParking)),
                  DropdownMenuItem(value: 'other', child: Text(l10n.maintenanceCategoryOther)),
                ],
                onChanged: (v) => setState(() => _category = v ?? 'other'),
              ),
              const SizedBox(height: 12),
              DropdownButtonFormField<String>(
                initialValue: _priority,
                decoration: InputDecoration(labelText: l10n.maintenancePriorityLabel),
                items: [
                  DropdownMenuItem(value: 'normal', child: Text(l10n.maintenancePriorityNormal)),
                  DropdownMenuItem(value: 'important', child: Text(l10n.maintenancePriorityImportant)),
                  DropdownMenuItem(value: 'urgent', child: Text(l10n.maintenancePriorityUrgent)),
                ],
                onChanged: (v) => setState(() => _priority = v ?? 'normal'),
              ),
              const SizedBox(height: 12),
              TextFormField(
                controller: _locationCtrl,
                decoration: InputDecoration(labelText: l10n.maintenanceLocationLabel, hintText: l10n.maintenanceLocationHint),
              ),
              const SizedBox(height: 12),
              TextFormField(
                controller: _descCtrl,
                decoration: InputDecoration(labelText: l10n.maintenanceDescriptionLabel),
                maxLines: 4,
              ),
              const SizedBox(height: 12),
              AttachmentPicker(
                onChanged: (f) => setState(() => _photo = f),
              ),
            ],
          ),
        )
      ),
      bottomNavigationBar: SaveBar(
        child: FilledButton(
          onPressed: _saving ? null : _submit,
          child: _saving
              ? const SizedBox(width: 18, height: 18, child: CircularProgressIndicator(strokeWidth: 2))
              : Text(l10n.addMaintenanceRequest),
        ),
      ),
    );
  }
}
