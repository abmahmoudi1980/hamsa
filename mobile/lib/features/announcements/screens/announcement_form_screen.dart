import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:image_picker/image_picker.dart';

import '../../../core/datetime/jalali.dart';
import '../../../shared/widgets/attachment_picker.dart';
import '../announcement_controller.dart';
import '../models/announcement.dart';

/// T079 — manager publish / edit form (Jalali publish/expire pickers via T018,
/// audience type/value, attachment).
class AnnouncementFormScreen extends ConsumerStatefulWidget {
  const AnnouncementFormScreen({super.key, required this.buildingId, this.announcement});

  final String buildingId;
  final Announcement? announcement;

  @override
  ConsumerState<AnnouncementFormScreen> createState() => _AnnouncementFormScreenState();
}

class _AnnouncementFormScreenState extends ConsumerState<AnnouncementFormScreen> {
  final _formKey = GlobalKey<FormState>();
  late final TextEditingController _titleCtrl;
  late final TextEditingController _bodyCtrl;
  late final TextEditingController _audienceValueCtrl;
  String _audienceType = 'all';
  DateTime? _publishAt;
  DateTime? _expireAt;
  String? _attachmentFileId;
  XFile? _pickedFile;
  bool _submitting = false;

  bool get _editing => widget.announcement != null;

  @override
  void initState() {
    super.initState();
    final a = widget.announcement;
    _titleCtrl = TextEditingController(text: a?.title ?? '');
    _bodyCtrl = TextEditingController(text: a?.body ?? '');
    _audienceType = a?.audienceType ?? 'all';
    _audienceValueCtrl = TextEditingController(text: a?.audienceValue ?? '');
    if (a?.publishAt != null) _publishAt = DateTime.tryParse(a!.publishAt!);
    if (a?.expireAt != null) _expireAt = DateTime.tryParse(a!.expireAt!);
  }

  @override
  void dispose() {
    _titleCtrl.dispose();
    _bodyCtrl.dispose();
    _audienceValueCtrl.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    if (!_formKey.currentState!.validate()) return;
    if (_audienceType != 'all' && _audienceValueCtrl.text.trim().isEmpty) {
      ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('مقدار مخاطب الزامی است')));
      return;
    }
    if (_publishAt != null && _expireAt != null && !_expireAt!.isAfter(_publishAt!)) {
      ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('تاریخ انقضا باید بعد از انتشار باشد')));
      return;
    }
    setState(() => _submitting = true);
    try {
      // Upload attachment if picked
      String? fileId = _attachmentFileId;
      if (_pickedFile != null) {
        fileId = await ref.read(announcementRepositoryProvider).uploadAttachment(path: _pickedFile!.path, name: _pickedFile!.name);
      }
      final repo = ref.read(announcementRepositoryProvider);
      if (_editing) {
        await repo.updateAnnouncement(
          widget.announcement!.id,
          title: _titleCtrl.text.trim(),
          body: _bodyCtrl.text.trim(),
          audienceType: _audienceType,
          audienceValue: _audienceType == 'all' ? null : _audienceValueCtrl.text.trim(),
          publishAt: _publishAt?.toUtc().toIso8601String(),
          expireAt: _expireAt?.toUtc().toIso8601String(),
          attachmentFileId: fileId,
        );
        if (mounted) {
          ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('اطلاعیه ویرایش شد')));
          context.pop(true);
        }
      } else {
        await repo.createAnnouncement(
          widget.buildingId,
          title: _titleCtrl.text.trim(),
          body: _bodyCtrl.text.trim(),
          audienceType: _audienceType,
          audienceValue: _audienceType == 'all' ? null : _audienceValueCtrl.text.trim(),
          publishAt: _publishAt?.toUtc().toIso8601String(),
          expireAt: _expireAt?.toUtc().toIso8601String(),
          attachmentFileId: fileId,
        );
        if (mounted) {
          ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('اطلاعیه با موفقیت منتشر شد')));
          context.pop(true);
        }
      }
    } catch (e) {
      if (mounted) ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('خطا: $e')));
    } finally {
      if (mounted) setState(() => _submitting = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: Text(_editing ? 'ویرایش اطلاعیه' : 'انتشار اطلاعیه')),
      body: Form(
        key: _formKey,
        child: ListView(
          padding: const EdgeInsets.all(16),
          children: [
            TextFormField(
              controller: _titleCtrl,
              decoration: const InputDecoration(labelText: 'عنوان اطلاعیه', hintText: 'مثلاً: جلسه هیئت مدیره'),
              validator: (v) => (v == null || v.trim().isEmpty) ? 'عنوان الزامی است' : null,
            ),
            const SizedBox(height: 12),
            TextFormField(
              controller: _bodyCtrl,
              decoration: const InputDecoration(labelText: 'متن اطلاعیه'),
              maxLines: 5,
              validator: (v) => (v == null || v.trim().isEmpty) ? 'متن الزامی است' : null,
            ),
            const SizedBox(height: 12),
            DropdownButtonFormField<String>(
              initialValue: _audienceType,
              decoration: const InputDecoration(labelText: 'مخاطب'),
              items: const [
                DropdownMenuItem(value: 'all', child: Text('همه ساکنان')),
                DropdownMenuItem(value: 'block', child: Text('بلوک')),
                DropdownMenuItem(value: 'floor', child: Text('طبقه')),
                DropdownMenuItem(value: 'unit', child: Text('واحد مشخص')),
              ],
              onChanged: (v) => setState(() => _audienceType = v ?? 'all'),
            ),
            if (_audienceType != 'all') ...[
              TextFormField(
                controller: _audienceValueCtrl,
                decoration: InputDecoration(
                  labelText: 'مقدار مخاطب',
                  hintText: _audienceType == 'block' ? 'مثلاً: A' : _audienceType == 'floor' ? 'مثلاً: ۲' : 'شناسه واحد (UUID)',
                ),
                validator: (v) => _audienceType != 'all' && (v == null || v.trim().isEmpty) ? 'مقدار الزامی است' : null,
              ),
            ],
            const SizedBox(height: 12),
            JalaliDatePickerField(
              label: 'تاریخ انتشار (اختیاری)',
              initialValue: _publishAt,
              onChanged: (d) => setState(() => _publishAt = d),
            ),
            const SizedBox(height: 12),
            JalaliDatePickerField(
              label: 'تاریخ انقضا (اختیاری)',
              initialValue: _expireAt,
              onChanged: (d) => setState(() => _expireAt = d),
            ),
            const SizedBox(height: 12),
            AttachmentPicker(
              initial: _pickedFile,
              onChanged: (file) => setState(() => _pickedFile = file),
            ),
            const SizedBox(height: 24),
            FilledButton(
              onPressed: _submitting ? null : _submit,
              child: _submitting
                  ? const SizedBox(width: 20, height: 20, child: CircularProgressIndicator(strokeWidth: 2))
                  : Text(_editing ? 'ذخیره تغییرات' : 'انتشار'),
            ),
          ],
        ),
      ),
    );
  }
}
