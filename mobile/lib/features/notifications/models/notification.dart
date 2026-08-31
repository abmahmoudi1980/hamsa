/// Notification model — mirrors backend `notifications` table + contracts/api.md.
class AppNotification {
  AppNotification.fromJson(Map<String, dynamic> json)
      : id = json['id'] as String,
        type = json['type'] as String? ?? '',
        title = json['title'] as String? ?? '',
        body = json['body'] as String?,
        refType = json['ref_type'] as String?,
        refId = json['ref_id'] as String?,
        isRead = json['is_read'] as bool? ?? false,
        readAt = json['read_at'] as String?,
        createdAt = json['created_at'] as String?;

  final String id;
  final String type;
  final String title;
  final String? body;
  final String? refType;
  final String? refId;
  final bool isRead;
  final String? readAt;
  final String? createdAt;
}
