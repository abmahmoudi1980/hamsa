import 'package:dio/dio.dart';

import 'models/announcement.dart';

/// US8 announcements API (T079/T080) — mirrors backend contracts/api.md
/// "Announcements": manager POST/GET /buildings/:id/announcements and
/// PUT/DELETE /announcements/:id; resident GET /me/announcements and
/// POST /announcements/:id/read.
class AnnouncementRepository {
  AnnouncementRepository(this._dio);

  final Dio _dio;

  // --- manager --------------------------------------------------------------

  Future<(List<Announcement>, int)> buildingAnnouncements(
    String buildingId, {
    int page = 1,
    int pageSize = 20,
  }) async {
    final res = await _dio.get<Map<String, dynamic>>(
      '/buildings/$buildingId/announcements',
      queryParameters: {'page': page, 'page_size': pageSize},
    );
    final data = res.data!;
    final items = (data['items'] as List)
        .map((e) => Announcement.fromJson(e as Map<String, dynamic>))
        .toList();
    return (items, (data['total'] as num).toInt());
  }

  Future<Announcement> getAnnouncement(String id) async {
    final res = await _dio.get<Map<String, dynamic>>('/announcements/$id');
    return Announcement.fromJson(res.data!);
  }

  Future<Announcement> createAnnouncement(
    String buildingId, {
    required String title,
    required String body,
    required String audienceType,
    String? audienceValue,
    String? publishAt,
    String? expireAt,
    String? attachmentFileId,
  }) async {
    final payload = <String, dynamic>{
      'title': title,
      'body': body,
      'audience_type': audienceType,
    };
    if (audienceValue != null && audienceValue.isNotEmpty) {
      payload['audience_value'] = audienceValue;
    }
    if (publishAt != null && publishAt.isNotEmpty) payload['publish_at'] = publishAt;
    if (expireAt != null && expireAt.isNotEmpty) payload['expire_at'] = expireAt;
    if (attachmentFileId != null) payload['attachment_file_id'] = attachmentFileId;
    final res = await _dio.post<Map<String, dynamic>>(
      '/buildings/$buildingId/announcements',
      data: payload,
    );
    return Announcement.fromJson(res.data!);
  }

  Future<Announcement> updateAnnouncement(
    String id, {
    String? title,
    String? body,
    String? audienceType,
    String? audienceValue,
    String? publishAt,
    String? expireAt,
    String? attachmentFileId,
  }) async {
    final payload = <String, dynamic>{};
    if (title != null) payload['title'] = title;
    if (body != null) payload['body'] = body;
    if (audienceType != null) payload['audience_type'] = audienceType;
    if (audienceValue != null) payload['audience_value'] = audienceValue;
    if (publishAt != null) payload['publish_at'] = publishAt;
    if (expireAt != null) payload['expire_at'] = expireAt;
    if (attachmentFileId != null) payload['attachment_file_id'] = attachmentFileId;
    final res = await _dio.put<Map<String, dynamic>>('/announcements/$id', data: payload);
    return Announcement.fromJson(res.data!);
  }

  Future<void> deleteAnnouncement(String id) =>
      _dio.delete<void>('/announcements/$id');

  Future<String?> uploadAttachment({
    required String path,
    required String name,
  }) async {
    final form = FormData.fromMap({
      'file': await MultipartFile.fromFile(path, filename: name),
    });
    final res = await _dio.post<Map<String, dynamic>>('/files', data: form);
    return res.data?['id'] as String?;
  }

  // --- resident -------------------------------------------------------------

  Future<(List<Announcement>, int)> myAnnouncements({
    int page = 1,
    int pageSize = 20,
  }) async {
    final res = await _dio.get<Map<String, dynamic>>(
      '/me/announcements',
      queryParameters: {'page': page, 'page_size': pageSize},
    );
    final data = res.data!;
    final items = (data['items'] as List)
        .map((e) => Announcement.fromJson(e as Map<String, dynamic>))
        .toList();
    return (items, (data['total'] as num).toInt());
  }

  Future<Announcement> myAnnouncement(String id) async {
    final res = await _dio.get<Map<String, dynamic>>('/me/announcements/$id');
    return Announcement.fromJson(res.data!);
  }

  Future<void> markRead(String id) =>
      _dio.post<void>('/announcements/$id/read');

  Future<void> markReadMe(String id) =>
      _dio.post<void>('/me/announcements/$id/read');
}
