import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../auth/auth_repository.dart';
import 'models/notification.dart';
import 'notifications_repository.dart';

final notificationsRepositoryProvider = Provider<NotificationsRepository>(
  (ref) => NotificationsRepository(ref.watch(apiClientProvider).dio),
);

class NotificationsController extends AsyncNotifier<List<AppNotification>> {
  @override
  Future<List<AppNotification>> build() async {
    final (items, _) =
        await ref.watch(notificationsRepositoryProvider).list();
    return items;
  }

  Future<void> refresh() async {
    state = const AsyncLoading<List<AppNotification>>().copyWithPrevious(state);
    state = await AsyncValue.guard(() async {
      final (items, _) = await ref.read(notificationsRepositoryProvider).list();
      return items;
    });
  }

  Future<void> refreshUnreadOnly(bool unreadOnly) async {
    state = const AsyncLoading<List<AppNotification>>().copyWithPrevious(state);
    state = await AsyncValue.guard(() async {
      final (items, _) = await ref
          .read(notificationsRepositoryProvider)
          .list(unreadOnly: unreadOnly);
      return items;
    });
  }

  Future<void> markRead(String id) async {
    await ref.read(notificationsRepositoryProvider).markRead(id);
    await refresh();
  }

  Future<void> markAllRead() async {
    await ref.read(notificationsRepositoryProvider).markAllRead();
    await refresh();
  }

  int get unreadCount =>
      (state.valueOrNull ?? []).where((n) => !n.isRead).length;
}

final notificationsControllerProvider =
    AsyncNotifierProvider<NotificationsController, List<AppNotification>>(
  NotificationsController.new,
);
