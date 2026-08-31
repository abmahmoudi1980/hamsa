/// US7 maintenance models (T073/T074) — mirrors the backend contracts/api.md
/// "Maintenance" section. Enum tokens stay locale-neutral; all Persian labels
/// are resolved via AppLocalizations (fa.arb) in the UI layer — no hard-coded
/// Persian in the model.
class MaintenanceRequest {
  MaintenanceRequest.fromJson(Map<String, dynamic> json)
      : id = json['id'] as String,
        buildingId = json['building_id'] as String? ?? '',
        unitId = json['unit_id'] as String?,
        submittedBy = json['submitted_by'] as String? ?? '',
        title = json['title'] as String? ?? '',
        category = json['category'] as String? ?? 'other',
        description = json['description'] as String?,
        location = json['location'] as String?,
        photoFile = json['photo_file'] as String?,
        priority = json['priority'] as String? ?? 'normal',
        status = json['status'] as String? ?? 'new',
        assigneePersonId = json['assignee_person_id'] as String?,
        recordedCost = json['recorded_cost'] == null
            ? null
            : int.tryParse('${json['recorded_cost']}'),
        notes = json['notes'] as String?,
        createdAt = json['created_at'] as String?,
        updatedAt = json['updated_at'] as String?,
        closedAt = json['closed_at'] as String?;

  final String id;
  final String buildingId;
  final String? unitId;
  final String submittedBy;
  final String title;
  final String category;
  final String? description;
  final String? location;
  final String? photoFile;
  final String priority;
  final String status;
  final String? assigneePersonId;
  final int? recordedCost;
  final String? notes;
  final String? createdAt;
  final String? updatedAt;
  final String? closedAt;
}

