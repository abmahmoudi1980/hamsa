import 'package:dio/dio.dart';

import 'models/dashboard.dart';

/// US9 (`/me/home`) + US10 (`/buildings/{id}/dashboard`) repository.
class DashboardRepository {
  DashboardRepository(this._dio);

  final Dio _dio;

  Future<HomeSummary> home() async {
    final res = await _dio.get<Map<String, dynamic>>('/me/home');
    return HomeSummary.fromJson(res.data!);
  }

  Future<BuildingDashboard> buildingDashboard(String buildingId) async {
    final res = await _dio.get<Map<String, dynamic>>(
      '/buildings/$buildingId/dashboard',
    );
    return BuildingDashboard.fromJson(res.data!);
  }
}
