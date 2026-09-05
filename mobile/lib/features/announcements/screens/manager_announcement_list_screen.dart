import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../core/datetime/jalali.dart';
import '../../../shared/widgets/empty_state.dart';
import '../announcement_controller.dart';
import '../models/announcement.dart';

/// T079 — manager announcement list for one building.
class ManagerAnnouncementListScreen extends ConsumerWidget {
  const ManagerAnnouncementListScreen({super.key, required this.buildingId});

  final String buildingId;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final async = ref.watch(
      buildingAnnouncementsControllerProvider(buildingId),
    );
    return Scaffold(
      appBar: AppBar(title: const Text('اطلاعیه‌ها')),
      floatingActionButton: FloatingActionButton.extended(
        onPressed: () => context.push('/manager/announcements/$buildingId/new'),
        icon: const Icon(Icons.add),
        label: const Text('انتشار اطلاعیه'),
      ),
      body: SafeArea(
        child: async.when(
          loading: () => const Center(child: CircularProgressIndicator()),
          error: (e, _) => _ErrorView(
            error: '$e',
            onRetry: () => ref
                .read(
                  buildingAnnouncementsControllerProvider(buildingId).notifier,
                )
                .refresh(),
          ),
          data: (items) {
            if (items.isEmpty) {
              return const EmptyState(
                icon: Icons.campaign_outlined,
                title: 'هنوز اطلاعیه‌ای منتشر نشده است',
                subtitle: 'برای اطلاع‌رسانی جدید، «انتشار اطلاعیه» را بزنید.',
              );
            }
            return RefreshIndicator(
              onRefresh: () => ref
                  .read(
                    buildingAnnouncementsControllerProvider(buildingId).notifier,
                  )
                  .refresh(),
              child: ListView.separated(
                padding: const EdgeInsets.all(16),
                itemCount: items.length,
                separatorBuilder: (_, __) => const SizedBox(height: 12),
                itemBuilder: (_, i) =>
                    _AnnouncementCard(item: items[i], buildingId: buildingId),
              ),
            );
          },
        )
      ),
    );
  }
}

class _AnnouncementCard extends ConsumerWidget {
  const _AnnouncementCard({required this.item, required this.buildingId});

  final Announcement item;
  final String buildingId;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Expanded(
                  child: Text(
                    item.title,
                    style: Theme.of(context).textTheme.titleMedium?.copyWith(
                      fontWeight: FontWeight.bold,
                    ),
                  ),
                ),
                _AudienceChip(
                  audienceType: item.audienceType,
                  audienceValue: item.audienceValue,
                ),
              ],
            ),
            if (item.body.isNotEmpty) ...[
              const SizedBox(height: 8),
              Text(
                item.body,
                maxLines: 3,
                overflow: TextOverflow.ellipsis,
                style: Theme.of(context).textTheme.bodyMedium,
              ),
            ],
            const SizedBox(height: 8),
            Row(
              children: [
                if (item.createdAt != null)
                  Text(
                    _formatDate(item.createdAt!),
                    style: Theme.of(
                      context,
                    ).textTheme.bodySmall?.copyWith(color: Colors.grey[600]),
                  ),
                const Spacer(),
                PopupMenuButton<String>(
                  onSelected: (v) async {
                    if (v == 'edit') {
                      final changed = await context.push<bool>(
                        '/manager/announcements/$buildingId/${item.id}',
                        extra: item,
                      );
                      if (changed == true && context.mounted) {
                        ref
                            .read(
                              buildingAnnouncementsControllerProvider(
                                buildingId,
                              ).notifier,
                            )
                            .refresh();
                      }
                    } else if (v == 'delete') {
                      final ok = await showDialog<bool>(
                        context: context,
                        builder: (_) => AlertDialog(
                          title: const Text('حذف اطلاعیه'),
                          content: const Text('این اطلاعیه حذف شود؟'),
                          actions: [
                            TextButton(
                              onPressed: () => Navigator.pop(context, false),
                              child: const Text('انصراف'),
                            ),
                            FilledButton(
                              onPressed: () => Navigator.pop(context, true),
                              child: const Text('حذف'),
                            ),
                          ],
                        ),
                      );
                      if (ok == true) {
                        try {
                          await ref
                              .read(announcementRepositoryProvider)
                              .deleteAnnouncement(item.id);
                          if (context.mounted) {
                            ScaffoldMessenger.of(context).showSnackBar(
                              const SnackBar(content: Text('اطلاعیه حذف شد')),
                            );
                            ref
                                .read(
                                  buildingAnnouncementsControllerProvider(
                                    buildingId,
                                  ).notifier,
                                )
                                .refresh();
                          }
                        } catch (e) {
                          if (context.mounted) {
                            ScaffoldMessenger.of(context).showSnackBar(
                              SnackBar(content: Text('حذف ناموفق: $e')),
                            );
                          }
                        }
                      }
                    }
                  },
                  itemBuilder: (_) => const [
                    PopupMenuItem(value: 'edit', child: Text('ویرایش')),
                    PopupMenuItem(value: 'delete', child: Text('حذف')),
                  ],
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }
}

class _AudienceChip extends StatelessWidget {
  const _AudienceChip({required this.audienceType, this.audienceValue});
  final String audienceType;
  final String? audienceValue;

  @override
  Widget build(BuildContext context) {
    String label;
    switch (audienceType) {
      case 'all':
        label = 'همه';
        break;
      case 'block':
        label = 'بلوک ${toPersianDigits(audienceValue ?? '')}';
        break;
      case 'floor':
        label = 'طبقه ${toPersianDigits(audienceValue ?? '')}';
        break;
      case 'unit':
        label =
            'واحد ${toPersianDigits((audienceValue ?? '').substring(0, 8))}';
        break;
      default:
        label = audienceType;
    }
    return Chip(
      label: Text(label, style: const TextStyle(fontSize: 12)),
      visualDensity: VisualDensity.compact,
    );
  }
}

class _ErrorView extends StatelessWidget {
  const _ErrorView({required this.error, required this.onRetry});
  final String error;
  final VoidCallback onRetry;
  @override
  Widget build(BuildContext context) {
    return Center(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Text(error),
          const SizedBox(height: 12),
          SizedBox(
            width: double.infinity,
            child: FilledButton(
              onPressed: onRetry,
              child: const Text('تلاش دوباره'),
            ),
          ),
        ],
      ),
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
