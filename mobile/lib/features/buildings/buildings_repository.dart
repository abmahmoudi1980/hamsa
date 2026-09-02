import 'package:dio/dio.dart';

import 'models/building.dart';
import 'models/people.dart';

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

  Future<Building> updateBuilding(
    String id,
    Map<String, dynamic> payload,
  ) async {
    final res = await _dio.put<Map<String, dynamic>>(
      '/buildings/$id',
      data: payload,
    );
    return Building.fromJson(res.data!);
  }

  Future<void> archiveBuilding(String id) =>
      _dio.delete<void>('/buildings/$id');

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

  Future<Unit> createUnit(
    String buildingId,
    Map<String, dynamic> payload,
  ) async {
    final res = await _dio.post<Map<String, dynamic>>(
      '/buildings/$buildingId/units',
      data: payload,
    );
    return Unit.fromJson(res.data!);
  }

  Future<Unit> updateUnit(String id, Map<String, dynamic> payload) async {
    final res = await _dio.put<Map<String, dynamic>>(
      '/units/$id',
      data: payload,
    );
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
          (e) => UnitHistoryEntry.fromJson(Map<String, dynamic>.from(e as Map)),
        )
        .toList();
  }

  // --- US3: people, occupancies, occupant counts -----------------------------

  /// `GET /buildings/{id}/persons?q=` → page envelope.
  Future<(List<Person>, int)> listPersons(
    String buildingId, {
    String q = '',
    int page = 1,
    int pageSize = 50,
  }) async {
    final res = await _dio.get<Map<String, dynamic>>(
      '/buildings/$buildingId/persons',
      queryParameters: {
        if (q.isNotEmpty) 'q': q,
        'page': page,
        'page_size': pageSize,
      },
    );
    final items = (res.data?['items'] as List? ?? const [])
        .map((e) => Person.fromJson(Map<String, dynamic>.from(e as Map)))
        .toList();
    return (items, (res.data?['total'] ?? 0) as int);
  }

  Future<Person> createPerson(
    String buildingId,
    Map<String, dynamic> payload,
  ) async {
    final res = await _dio.post<Map<String, dynamic>>(
      '/buildings/$buildingId/persons',
      data: payload,
    );
    return Person.fromJson(res.data!);
  }

  Future<Person> updatePerson(String id, Map<String, dynamic> payload) async {
    final res = await _dio.put<Map<String, dynamic>>(
      '/persons/$id',
      data: payload,
    );
    return Person.fromJson(res.data!);
  }

  Future<void> archivePerson(String id) => _dio.delete<void>('/persons/$id');

  /// `GET /units/{id}/occupancies` — full dated history (FR-007).
  Future<List<Occupancy>> listOccupancies(String unitId) async {
    final res = await _dio.get<Map<String, dynamic>>(
      '/units/$unitId/occupancies',
    );
    return (res.data?['items'] as List? ?? const [])
        .map((e) => Occupancy.fromJson(Map<String, dynamic>.from(e as Map)))
        .toList();
  }

  /// `POST /units/{id}/occupancies` — adding a new active tenant closes the
  /// previous one server-side (FR-007).
  Future<Occupancy> addOccupancy(
    String unitId,
    Map<String, dynamic> payload,
  ) async {
    final res = await _dio.post<Map<String, dynamic>>(
      '/units/$unitId/occupancies',
      data: payload,
    );
    return Occupancy.fromJson(res.data!);
  }

  /// `PATCH /occupancies/{id}` — end-date, preserving history (FR-007).
  Future<Occupancy> endOccupancy(String id, String endDate) async {
    final res = await _dio.patch<Map<String, dynamic>>(
      '/occupancies/$id',
      data: {'end_date': endDate},
    );
    return Occupancy.fromJson(res.data!);
  }

  /// `GET /units/{id}/occupant-count` — full history, newest first (FR-008).
  Future<List<OccupantCountEntry>> listOccupantCounts(String unitId) async {
    final res = await _dio.get<Map<String, dynamic>>(
      '/units/$unitId/occupant-count',
    );
    return (res.data?['items'] as List? ?? const [])
        .map(
          (e) =>
              OccupantCountEntry.fromJson(Map<String, dynamic>.from(e as Map)),
        )
        .toList();
  }

  /// `POST /units/{id}/occupant-count` — append one data point (BR-04).
  Future<void> recordOccupantCount(
    String unitId, {
    required int count,
    required String effectiveFrom,
  }) async {
    await _dio.post<Map<String, dynamic>>(
      '/units/$unitId/occupant-count',
      data: {'occupant_count': count, 'effective_from': effectiveFrom},
    );
  }

  // --- US5: per-building managers (002-multi-manager-support) ---------------

  /// `GET /buildings/{id}/managers` → granted managers, oldest grant first.
  Future<List<BuildingManager>> listManagers(String buildingId) async {
    final res = await _dio.get<Map<String, dynamic>>(
      '/buildings/$buildingId/managers',
    );
    return (res.data?['items'] as List? ?? const [])
        .map(
          (e) => BuildingManager.fromJson(Map<String, dynamic>.from(e as Map)),
        )
        .toList();
  }

  /// `POST /buildings/{id}/managers` → 201 with the created grant row.
  /// Duplicate (409) / unknown-phone (404) / superadmin-target (400) errors
  /// propagate for [describeError] to surface the server Persian copy.
  Future<BuildingManager> addManager(String buildingId, String phone) async {
    final res = await _dio.post<Map<String, dynamic>>(
      '/buildings/$buildingId/managers',
      data: {'phone': phone},
    );
    return BuildingManager.fromJson(res.data!);
  }

  /// `DELETE /buildings/{id}/managers/{userId}` → 204 on success.
  Future<void> removeManager(String buildingId, String userId) =>
      _dio.delete<void>('/buildings/$buildingId/managers/$userId');
}
