import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../core/datetime/jalali.dart';
import '../../../shared/widgets/empty_state.dart';
import '../notifications_controller.dart';
import '../models/notification.dart';

/// T080 — notification center (list, unread filter, deep links via go_router, mark-all-read).
class NotificationCenterScreen extends ConsumerStatefulWidget {
  const NotificationCenterScreen({super.key});

  @override
  ConsumerState<NotificationCenterScreen> createState() => _NotificationCenterScreenState();
}

class _NotificationCenterScreenState extends ConsumerState<NotificationCenterScreen> {
  bool _unreadOnly = false;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) => _load());
  }

  Future<void> _load() async {
    if (_unreadOnly) {
      await ref.read(notificationsControllerProvider.notifier).refreshUnreadOnly(true);
    } else {
      await ref.read(notificationsControllerProvider.notifier).refresh();
    }
  }

  void _onTap(AppNotification n) {
    // Deep link per ref_type/ref_id (contracts/api.md).
    switch (n.refType) {
      case 'announcement':
        if (n.refId != null) context.push('/home/announcements/${n.refId}');
        break;
      case 'invoice':
        if (n.refId != null) context.push('/invoice/${n.refId}');
        break;
      case 'maintenance_request':
        if (n.refId != null) context.push('/home/maintenance');
        break;
      default:
        break;
    }
    if (!n.isRead) {
      ref.read(notificationsControllerProvider.notifier).markRead(n.id);
    }
  }

  @override
  Widget build(BuildContext context) {
    final async = ref.watch(notificationsControllerProvider);
    return Scaffold(
      appBar: AppBar(
        title: const Text('اعلان‌ها'),
        actions: [
          IconButton(
            tooltip: 'خواندن همه',
            onPressed: () async {
              await ref.read(notificationsControllerProvider.notifier).markAllRead();
              if (context.mounted) ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('همه اعلان‌ها خوانده شد')));
            },
            icon: const Icon(Icons.done_all),
          ),
        ],
      ),
      body: Column(
        children: [
          Padding(
            padding: const EdgeInsets.all(12),
            child: Row(children: [
              const Text('فقط خوانده‌نشده'),
              const SizedBox(width: 8),
              Switch(
                value: _unreadOnly,
                onChanged: (v) async {
                  setState(() => _unreadOnly = v);
                  await _load();
                },
              ),
              const Spacer(),
              TextButton.icon(onPressed: _load, icon: const Icon(Icons.refresh), label: const Text('به‌روزرسانی')),
            ]),
          ),
          Expanded(
            child: async.when(
              loading: () => const Center(child: CircularProgressIndicator()),
              error: (e, _) => Center(child: Column(mainAxisSize: MainAxisSize.min, children: [Text('خطا: $e'), const SizedBox(height: 12), FilledButton(onPressed: _load, child: const Text('تلاش دوباره'))])),
              data: (items) {
                if (items.isEmpty) {
                  return const EmptyState(
                    icon: Icons.notifications_none,
                    title: 'اعلانی وجود ندارد',
                    subtitle: 'اعلان‌های شارژ، پرداخت و اطلاعیه‌ها اینجا نمایش داده می‌شود.',
                  );
                }
                return RefreshIndicator(
                  onRefresh: _load,
                  child: ListView.separated(
                    padding: const EdgeInsets.all(16),
                    itemCount: items.length,
                    separatorBuilder: (_, __) => const SizedBox(height: 8),
                    itemBuilder: (_, i) => _NotificationTile(item: items[i], onTap: () => _onTap(items[i])),
                  ),
                );
              },
            ),
          ),
        ],
      ),
    );
  }
}

class _NotificationTile extends StatelessWidget {
  const _NotificationTile({required this.item, required this.onTap});
  final AppNotification item;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    return Card(
      color: item.isRead ? null : Theme.of(context).colorScheme.surfaceContainerHighest,
      child: ListTile(
        leading: Icon(_iconFor(item.type), color: item.isRead ? Colors.grey : Theme.of(context).colorScheme.primary),
        title: Text(item.title, style: TextStyle(fontWeight: item.isRead ? FontWeight.normal : FontWeight.bold)),
        subtitle: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
          if (item.body != null) Text(item.body!, maxLines: 2, overflow: TextOverflow.ellipsis),
          if (item.createdAt != null) Text(_formatDate(item.createdAt!), style: Theme.of(context).textTheme.bodySmall?.copyWith(color: Colors.grey[600])),
        ]),
        trailing: item.isRead ? null : const Icon(Icons.circle, size: 10, color: Colors.blue),
        onTap: onTap,
      ),
    );
  }
}

IconData _iconFor(String type) {
  switch (type) {
    case 'invoice_issued':
      return Icons.receipt_long;
    case 'announcement_published':
      return Icons.campaign;
    case 'request_status_changed':
    case 'request_submitted':
      return Icons.build;
    case 'payment_recorded':
      return Icons.payments;
    default:
      return Icons.notifications;
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
