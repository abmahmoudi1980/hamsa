import 'package:dio/dio.dart';

import 'models/maintenance.dart';

/// US7 maintenance API (T073/T074) — mirrors backend contracts/api.md
/// "Maintenance": resident POST/GET /me/maintenance-requests plus manager
/// GET /buildings/:id/maintenance-requests and PATCH /maintenance-requests/:id.
class MaintenanceRepository {
  MaintenanceRepository(this._dio);

  final Dio _dio;

  // --- resident -------------------------------------------------------------

  Future<MaintenanceRequest> submit({
    required String title,
    required String category,
    String? description,
    String? location,
    String? photoFileId,
    String priority = 'normal',
  }) async {
    final res = await _dio.post<Map<String, dynamic>>(
      '/me/maintenance-requests',
      data: {
        'title': title,
        'category': category,
        if (description != null && description.isNotEmpty) 'description': description,
        if (location != null && location.isNotEmpty) 'location': location,
        if (photoFileId != null) 'photo_file_id': photoFileId,
        'priority': priority,
      },
    );
    return MaintenanceRequest.fromJson(res.data!);
  }

  Future<String?> uploadPhoto({required String path, required String name}) async {
    final form = FormData.fromMap({
      'file': await MultipartFile.fromFile(path, filename: name),
    });
    try {
      final res = await _dio.post<Map<String, dynamic>>('/files', data: form);
      return res.data?['id'] as String?;
    } on DioException {
      return null;
    }
  }

  Future<(List<MaintenanceRequest>, int)> myRequests({int page = 1, int pageSize = 20}) async {
    final res = await _dio.get<Map<String, dynamic>>(
      '/me/maintenance-requests',
      queryParameters: {'page': page, 'page_size': pageSize},
    );
    final data = res.data!;
    final items = (data['items'] as List? ?? [])
        .map((e) => MaintenanceRequest.fromJson(e as Map<String, dynamic>))
        .toList();
    final total = (data['total'] as num?)?.toInt() ?? items.length;
    return (items, total);
  }

  Future<MaintenanceRequest> myRequest(String id) async {
    final res = await _dio.get<Map<String, dynamic>>('/me/maintenance-requests/$id');
    return MaintenanceRequest.fromJson(res.data!);
  }

  // --- manager --------------------------------------------------------------

  Future<(List<MaintenanceRequest>, int)> buildingRequests(
    String buildingId, {
    String status = '',
    String priority = '',
    String category = '',
    int page = 1,
    int pageSize = 20,
  }) async {
    final res = await _dio.get<Map<String, dynamic>>(
      '/buildings/$buildingId/maintenance-requests',
      queryParameters: {
        if (status.isNotEmpty) 'status': status,
        if (priority.isNotEmpty) 'priority': priority,
        if (category.isNotEmpty) 'category': category,
        'page': page,
        'page_size': pageSize,
      },
    );
    final data = res.data!;
    final items = (data['items'] as List? ?? [])
        .map((e) => MaintenanceRequest.fromJson(e as Map<String, dynamic>))
        .toList();
    final total = (data['total'] as num?)?.toInt() ?? items.length;
    return (items, total);
  }

  Future<MaintenanceRequest> getRequest(String id) async {
    final res = await _dio.get<Map<String, dynamic>>('/maintenance-requests/$id');
    return MaintenanceRequest.fromJson(res.data!);
  }

  Future<MaintenanceRequest> updateRequest(
    String id, {
    String? status,
    String? assigneePersonId,
    int? recordedCost,
    bool clearCost = false,
    String? notes,
  }) async {
    final data = <String, dynamic>{};
    if (status != null) data['status'] = status;
    if (assigneePersonId != null) {
      data['assignee_person_id'] = assigneePersonId.isEmpty ? null : assigneePersonId;
    }
    if (clearCost) {
      data['recorded_cost'] = null;
    } else if (recordedCost != null) {
      data['recorded_cost'] = recordedCost;
    }
    if (notes != null) data['notes'] = notes;
    final res = await _dio.patch<Map<String, dynamic>>('/maintenance-requests/$id', data: data);
    return MaintenanceRequest.fromJson(res.data!);
  }
}
