import 'package:dio/dio.dart';

import 'models/notification.dart';

/// Notifications API — mirrors contracts/api.md "Notifications".
class NotificationsRepository {
  NotificationsRepository(this._dio);

  final Dio _dio;

  Future<(List<AppNotification>, int)> list({
    bool unreadOnly = false,
    int page = 1,
    int pageSize = 20,
  }) async {
    final res = await _dio.get<Map<String, dynamic>>(
      '/me/notifications',
      queryParameters: {
        if (unreadOnly) 'unreadOnly': 'true',
        'page': page,
        'page_size': pageSize,
      },
    );
    final data = res.data!;
    final items = (data['items'] as List)
        .map((e) => AppNotification.fromJson(e as Map<String, dynamic>))
        .toList();
    return (items, (data['total'] as num).toInt());
  }

  Future<void> markRead(String id) =>
      _dio.post<void>('/me/notifications/$id/read');

  Future<void> markAllRead() =>
      _dio.post<void>('/me/notifications/read-all');
}
