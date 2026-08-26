import 'package:dio/dio.dart';

import 'models/building.dart';

class BuildingsRepository {
  BuildingsRepository(this._dio);

  final Dio _dio;

  Future<List<Building>> listBuildings() async {
    final res = await _dio.get<Map<String, dynamic>>('/buildings');
    final items = (res.data?['items'] as List? ?? const [])
        .map((e) => Building.fromJson(Map<String, dynamic>.from(e as Map)))
        .toList();
    return items;
  }

  Future<Building> getBuilding(String id) async {
    final res = await _dio.get<Map<String, dynamic>>('/buildings/$id');
    return Building.fromJson(res.data!);
  }

  Future<Building> createBuilding(Map<String, dynamic> payload) async {
    final res = await _dio.post<Map<String, dynamic>>(
      '/buildings',
      data: payload,
    );
    return Building.fromJson(res.data!);
  }

  Future<Building> updateBuilding(String id, Map<String, dynamic> payload) async {
    final res = await _dio.put<Map<String, dynamic>>('/buildings/$id', data: payload);
    return Building.fromJson(res.data!);
  }

  Future<void> archiveBuilding(String id) => _dio.delete<void>('/buildings/$id');

  /// `GET /buildings/{id}/units?q=&block=&floor=&status=` → page envelope.
  Future<(List<Unit>, int)> listUnits(
    String buildingId, {
    String q = '',
    String block = '',
    int? floor,
    String status = '',
    int page = 1,
    int pageSize = 20,
  }) async {
    final res = await _dio.get<Map<String, dynamic>>(
      '/buildings/$buildingId/units',
      queryParameters: {
        if (q.isNotEmpty) 'q': q,
        if (block.isNotEmpty) 'block': block,
        if (floor != null) 'floor': floor,
        if (status.isNotEmpty) 'status': status,
        'page': page,
        'page_size': pageSize,
      },
    );
    final items = (res.data?['items'] as List? ?? const [])
        .map((e) => Unit.fromJson(Map<String, dynamic>.from(e as Map)))
        .toList();
    return (items, (res.data?['total'] ?? 0) as int);
  }

  Future<Unit> createUnit(String buildingId, Map<String, dynamic> payload) async {
    final res = await _dio.post<Map<String, dynamic>>(
      '/buildings/$buildingId/units',
      data: payload,
    );
    return Unit.fromJson(res.data!);
  }

  Future<Unit> updateUnit(String id, Map<String, dynamic> payload) async {
    final res = await _dio.put<Map<String, dynamic>>('/units/$id', data: payload);
    return Unit.fromJson(res.data!);
  }

  Future<void> archiveUnit(String id) => _dio.delete<void>('/units/$id');


  Future<Unit> getUnit(String id) async {
    final res = await _dio.get<Map<String, dynamic>>('/units/$id');
    return Unit.fromJson(res.data!);
  }
  Future<List<UnitHistoryEntry>> unitHistory(String id) async {
    final res = await _dio.get<Map<String, dynamic>>('/units/$id/history');
    return (res.data?['items'] as List? ?? const [])
        .map(
          (e) => UnitHistoryEntry.fromJson(
            Map<String, dynamic>.from(e as Map),
          ),
        )
        .toList();
  }
}
