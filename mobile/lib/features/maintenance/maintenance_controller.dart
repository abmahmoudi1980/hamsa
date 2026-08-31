import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../auth/auth_repository.dart';
import 'maintenance_repository.dart';
import 'models/maintenance.dart';

final maintenanceRepositoryProvider = Provider<MaintenanceRepository>(
  (ref) => MaintenanceRepository(ref.watch(apiClientProvider).dio),
);

/// Resident own requests (no building scoping).
class MyMaintenanceController extends AsyncNotifier<List<MaintenanceRequest>> {
  @override
  Future<List<MaintenanceRequest>> build() async {
    final (items, _) = await ref.watch(maintenanceRepositoryProvider).myRequests();
    return items;
  }

  Future<void> refresh() async {
    state = const AsyncLoading<List<MaintenanceRequest>>().copyWithPrevious(state);
    state = await AsyncValue.guard(() async {
      final (items, _) = await ref.read(maintenanceRepositoryProvider).myRequests();
      return items;
    });
  }
}

final myMaintenanceControllerProvider =
    AsyncNotifierProvider<MyMaintenanceController, List<MaintenanceRequest>>(
  MyMaintenanceController.new,
);

/// Manager: requests for one building (family argument: building id).
class BuildingMaintenanceController
    extends FamilyAsyncNotifier<List<MaintenanceRequest>, String> {
  @override
  Future<List<MaintenanceRequest>> build(String buildingId) async {
    final (items, _) = await ref.watch(maintenanceRepositoryProvider).buildingRequests(buildingId);
    return items;
  }

  Future<void> refresh() async {
    state = const AsyncLoading<List<MaintenanceRequest>>().copyWithPrevious(state);
    state = await AsyncValue.guard(() => build(arg));
  }

  Future<void> applyFilter({String status = '', String priority = '', String category = ''}) async {
    state = const AsyncLoading<List<MaintenanceRequest>>().copyWithPrevious(state);
    state = await AsyncValue.guard(() async {
      final (items, _) = await ref.read(maintenanceRepositoryProvider).buildingRequests(
            arg,
            status: status,
            priority: priority,
            category: category,
          );
      return items;
    });
  }
}

final buildingMaintenanceControllerProvider = AsyncNotifierProvider.family<
    BuildingMaintenanceController, List<MaintenanceRequest>, String>(
  BuildingMaintenanceController.new,
);
