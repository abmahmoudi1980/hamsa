/// US8 announcements model (T079/T080) — mirrors backend contracts/api.md
/// "Announcements". Enum tokens stay locale-neutral.
class Announcement {
  Announcement.fromJson(Map<String, dynamic> json)
      : id = json['id'] as String,
        buildingId = json['building_id'] as String? ?? '',
        title = json['title'] as String? ?? '',
        body = json['body'] as String? ?? '',
        audienceType = json['audience_type'] as String? ?? 'all',
        audienceValue = json['audience_value'] as String?,
        publishAt = json['publish_at'] as String?,
        expireAt = json['expire_at'] as String?,
        attachmentFile = json['attachment_file'] as String?,
        createdBy = json['created_by'] as String?,
        createdAt = json['created_at'] as String?,
        updatedAt = json['updated_at'] as String?,
        isRead = json['is_read'] as bool? ?? false,
        readAt = json['read_at'] as String?;

  final String id;
  final String buildingId;
  final String title;
  final String body;
  final String audienceType;
  final String? audienceValue;
  final String? publishAt;
  final String? expireAt;
  final String? attachmentFile;
  final String? createdBy;
  final String? createdAt;
  final String? updatedAt;
  final bool isRead;
  final String? readAt;
}
