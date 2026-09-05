import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/datetime/jalali.dart';
import '../announcement_controller.dart';
import '../models/announcement.dart';

/// T080 — announcement detail (targeted only), mark-read.
class ResidentAnnouncementDetailScreen extends ConsumerStatefulWidget {
  const ResidentAnnouncementDetailScreen({
    super.key,
    required this.announcementId,
    this.announcement,
  });

  final String announcementId;
  final Announcement? announcement;

  @override
  ConsumerState<ResidentAnnouncementDetailScreen> createState() =>
      _ResidentAnnouncementDetailScreenState();
}

class _ResidentAnnouncementDetailScreenState
    extends ConsumerState<ResidentAnnouncementDetailScreen> {
  Announcement? _item;
  bool _loading = false;
  String? _error;
  bool _marking = false;

  @override
  void initState() {
    super.initState();
    _item = widget.announcement;
    if (_item == null) {
      _load();
    } else if (!(_item!.isRead)) {
      // Best-effort auto mark read when opening an unread announcement.
      WidgetsBinding.instance.addPostFrameCallback(
        (_) => _markRead(silent: true),
      );
    }
  }

  Future<void> _load() async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final a = await ref
          .read(announcementRepositoryProvider)
          .myAnnouncement(widget.announcementId);
      setState(() => _item = a);
      if (!a.isRead) await _markRead(silent: true);
    } catch (e) {
      setState(() => _error = '$e');
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  Future<void> _markRead({bool silent = false}) async {
    setState(() => _marking = true);
    try {
      await ref
          .read(announcementRepositoryProvider)
          .markRead(widget.announcementId);
      if (mounted && _item != null) {
        setState(
          () => _item = Announcement.fromJson({
            ..._toJson(_item!),
            'is_read': true,
          }),
        );
      }
      if (!silent && mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('به‌عنوان خوانده‌شده ثبت شد')),
        );
        Navigator.of(context).pop(true);
      }
    } catch (e) {
      if (!silent && mounted) {
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(SnackBar(content: Text('خطا: $e')));
      }
    } finally {
      if (mounted) setState(() => _marking = false);
    }
  }

  Map<String, dynamic> _toJson(Announcement a) => {
    'id': a.id,
    'building_id': a.buildingId,
    'title': a.title,
    'body': a.body,
    'audience_type': a.audienceType,
    'audience_value': a.audienceValue,
    'publish_at': a.publishAt,
    'expire_at': a.expireAt,
    'attachment_file': a.attachmentFile,
    'created_at': a.createdAt,
    'is_read': a.isRead,
  };

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('جزئیات اطلاعیه')),
      body: SafeArea(
        child: _loading
            ? const Center(child: CircularProgressIndicator())
            : _error != null
            ? Center(
                child: Column(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    Text(_error!),
                    const SizedBox(height: 12),
                    SizedBox(
                      width: double.infinity,
                      child: FilledButton(
                        onPressed: _load,
                        child: const Text('تلاش دوباره'),
                      ),
                    ),
                  ],
                ),
              )
            : _item == null
            ? const Center(child: Text('اطلاعیه یافت نشد'))
            : ListView(
                padding: const EdgeInsets.all(16),
                children: [
                  Row(
                    children: [
                      Expanded(
                        child: Text(
                          _item!.title,
                          style: Theme.of(context).textTheme.titleLarge?.copyWith(
                            fontWeight: FontWeight.bold,
                          ),
                        ),
                      ),
                      Chip(
                        label: Text(
                          _item!.isRead ? 'خوانده‌شده' : 'نخوانده',
                          style: TextStyle(
                            color: _item!.isRead
                                ? Colors.grey[700]
                                : Colors.white,
                          ),
                        ),
                        backgroundColor: _item!.isRead
                            ? Colors.grey[300]
                            : Theme.of(context).colorScheme.primary,
                      ),
                    ],
                  ),
                  const SizedBox(height: 8),
                  _AudienceLabel(
                    audienceType: _item!.audienceType,
                    audienceValue: _item!.audienceValue,
                  ),
                  const SizedBox(height: 12),
                  if (_item!.createdAt != null)
                    Text(
                      'تاریخ: ${_formatDate(_item!.createdAt!)}',
                      style: Theme.of(context).textTheme.bodySmall,
                    ),
                  if (_item!.publishAt != null)
                    Text(
                      'انتشار: ${_formatDate(_item!.publishAt!)}',
                      style: Theme.of(context).textTheme.bodySmall,
                    ),
                  if (_item!.expireAt != null)
                    Text(
                      'انقضا: ${_formatDate(_item!.expireAt!)}',
                      style: Theme.of(context).textTheme.bodySmall,
                    ),
                  const Divider(height: 24),
                  Text(_item!.body, style: Theme.of(context).textTheme.bodyLarge),
                  if (_item!.attachmentFile != null) ...[
                    const SizedBox(height: 16),
                    OutlinedButton.icon(
                      onPressed: () {},
                      icon: const Icon(Icons.attach_file),
                      label: const Text('مشاهده پیوست'),
                    ),
                  ],
                  const SizedBox(height: 24),
                  if (!_item!.isRead)
                    FilledButton.icon(
                      onPressed: _marking ? null : () => _markRead(),
                      icon: _marking
                          ? const SizedBox(
                              width: 16,
                              height: 16,
                              child: CircularProgressIndicator(
                                strokeWidth: 2,
                                color: Colors.white,
                              ),
                            )
                          : const Icon(Icons.check),
                      label: const Text('علامت‌گذاری به‌عنوان خوانده‌شده'),
                    ),
                ],
              )
      ),
    );
  }
}

class _AudienceLabel extends StatelessWidget {
  const _AudienceLabel({required this.audienceType, this.audienceValue});
  final String audienceType;
  final String? audienceValue;
  @override
  Widget build(BuildContext context) {
    String label;
    switch (audienceType) {
      case 'all':
        label = 'مخاطب: همه ساکنان';
        break;
      case 'block':
        label = 'مخاطب: بلوک ${toPersianDigits(audienceValue ?? '')}';
        break;
      case 'floor':
        label = 'مخاطب: طبقه ${toPersianDigits(audienceValue ?? '')}';
        break;
      case 'unit':
        label = 'مخاطب: واحد ${toPersianDigits(audienceValue ?? '')}';
        break;
      default:
        label = 'مخاطب: $audienceType';
    }
    return Text(
      label,
      style: Theme.of(
        context,
      ).textTheme.bodySmall?.copyWith(color: Colors.grey[700]),
    );
  }
}

String _formatDate(String iso) {
  try {
    final dt = DateTime.parse(iso).toLocal();
    return formatJalaliDate(dt);
  } catch (_) {
    return iso;
  }
}
