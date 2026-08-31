import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../auth/auth_repository.dart';
import 'dashboard_repository.dart';
import 'models/dashboard.dart';

final dashboardRepositoryProvider = Provider<DashboardRepository>(
  (ref) => DashboardRepository(ref.watch(apiClientProvider).dio),
);

/// US9 — resident home (T082/T083). Single fetch of `/me/home` covers the
/// financial card, requests card and announcements card.
class ResidentHomeController extends AsyncNotifier<HomeSummary> {
  @override
  Future<HomeSummary> build() =>
      ref.watch(dashboardRepositoryProvider).home();

  Future<void> refresh() async {
    state = const AsyncLoading<HomeSummary>().copyWithPrevious(state);
    state = await AsyncValue.guard(() =>
        ref.read(dashboardRepositoryProvider).home());
  }
}

final residentHomeControllerProvider =
    AsyncNotifierProvider<ResidentHomeController, HomeSummary>(
  ResidentHomeController.new,
);

/// US10 — manager building dashboard (T086/T087). Family argument: building id.
class BuildingDashboardController
    extends FamilyAsyncNotifier<BuildingDashboard, String> {
  @override
  Future<BuildingDashboard> build(String buildingId) =>
      ref.watch(dashboardRepositoryProvider).buildingDashboard(buildingId);

  Future<void> refresh() async {
    state = const AsyncLoading<BuildingDashboard>().copyWithPrevious(state);
    state = await AsyncValue.guard(() =>
        ref.read(dashboardRepositoryProvider).buildingDashboard(arg));
  }
}

final buildingDashboardControllerProvider = AsyncNotifierProvider.family<
    BuildingDashboardController, BuildingDashboard, String>(
  BuildingDashboardController.new,
);
