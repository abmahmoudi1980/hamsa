import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../core/datetime/jalali.dart';
import '../../../shared/widgets/empty_state.dart';
import '../announcement_controller.dart';
import '../models/announcement.dart';

/// T080 — resident announcements (targeted only), unread badge, mark-read.
class ResidentAnnouncementListScreen extends ConsumerWidget {
  const ResidentAnnouncementListScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final async = ref.watch(myAnnouncementsControllerProvider);
    return Scaffold(
      appBar: AppBar(title: const Text('اطلاعیه‌های من')),
      body: async.when(
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (e, _) => Center(child: Column(mainAxisSize: MainAxisSize.min, children: [Text('خطا: $e'), const SizedBox(height: 12), FilledButton(onPressed: () => ref.read(myAnnouncementsControllerProvider.notifier).refresh(), child: const Text('تلاش دوباره'))])),
        data: (items) {
          if (items.isEmpty) {
            return const EmptyState(
              icon: Icons.campaign_outlined,
              title: 'هنوز اطلاعیه‌ای برای شما منتشر نشده است',
              subtitle: 'اطلاعیه‌های جدید اینجا نمایش داده می‌شود.',
            );
          }
          final unread = items.where((a) => !a.isRead).length;
          return Column(
            children: [
              if (unread > 0)
                Padding(
                  padding: const EdgeInsets.all(12),
                  child: Chip(label: Text('${toPersianDigits(unread.toString())} ناخوانده', style: const TextStyle(color: Colors.white)), backgroundColor: Theme.of(context).colorScheme.primary),
                ),
              Expanded(
                child: RefreshIndicator(
                  onRefresh: () => ref.read(myAnnouncementsControllerProvider.notifier).refresh(),
                  child: ListView.separated(
                    padding: const EdgeInsets.all(16),
                    itemCount: items.length,
                    separatorBuilder: (_, __) => const SizedBox(height: 12),
                    itemBuilder: (_, i) => _ResidentCard(item: items[i]),
                  ),
                ),
              ),
            ],
          );
        },
      ),
    );
  }
}

class _ResidentCard extends ConsumerWidget {
  const _ResidentCard({required this.item});
  final Announcement item;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return Card(
      color: item.isRead ? null : Theme.of(context).colorScheme.surfaceContainerHighest,
      child: ListTile(
        title: Row(children: [
          Expanded(child: Text(item.title, style: TextStyle(fontWeight: item.isRead ? FontWeight.normal : FontWeight.bold))),
          if (!item.isRead) Container(padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4), decoration: BoxDecoration(color: Theme.of(context).colorScheme.primary, borderRadius: BorderRadius.circular(12)), child: const Text('نخوانده', style: TextStyle(color: Colors.white, fontSize: 11))),
        ]),
        subtitle: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
          const SizedBox(height: 4),
          Text(item.body, maxLines: 2, overflow: TextOverflow.ellipsis),
          const SizedBox(height: 4),
          Text(item.createdAt != null ? _formatDate(item.createdAt!) : '', style: Theme.of(context).textTheme.bodySmall?.copyWith(color: Colors.grey[600])),
        ]),
        onTap: () async {
          final changed = await context.push<bool>('/home/announcements/${item.id}', extra: item);
          if (changed == true && context.mounted) {
            ref.read(myAnnouncementsControllerProvider.notifier).refresh();
          } else if (!item.isRead && context.mounted) {
            // Auto mark read on open: the detail screen does it, just refresh list.
            ref.read(myAnnouncementsControllerProvider.notifier).refresh();
          }
        },
        trailing: const Icon(Icons.chevron_left),
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
