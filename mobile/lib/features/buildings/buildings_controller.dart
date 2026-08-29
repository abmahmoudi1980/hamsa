import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../core/network/api_exception.dart';
import '../auth/auth_repository.dart';
import 'buildings_repository.dart';
import 'models/building.dart';

final buildingsRepositoryProvider = Provider<BuildingsRepository>(
  (ref) => BuildingsRepository(ref.watch(apiClientProvider).dio),
);

/// Buildings list (US2/T033). Loading/error ride on AsyncValue; API failures
/// surface as [ApiException] and resolve to Persian copy in the screen.
class BuildingsController extends AsyncNotifier<List<Building>> {
  @override
  Future<List<Building>> build() =>
      ref.watch(buildingsRepositoryProvider).listBuildings();

  BuildingsRepository get _repo => ref.read(buildingsRepositoryProvider);

  /// Creates or updates; rethrows [ApiException] for the form to display.
  Future<void> save(Building? existing, Map<String, dynamic> payload) async {
    if (existing == null) {
      await _repo.createBuilding(payload);
    } else {
      await _repo.updateBuilding(existing.id, payload);
    }
    ref.invalidateSelf();
  }

  Future<void> archive(String id) async {
    await _repo.archiveBuilding(id);
    ref.invalidateSelf();
  }
}

final buildingsControllerProvider =
    AsyncNotifierProvider<BuildingsController, List<Building>>(
      BuildingsController.new,
    );

/// Unit list filters (`?q=&block=&status=` — floor arrives as its own chip
/// filter value inside [block] semantics kept separate).
class UnitsFilter {
  const UnitsFilter({this.q = '', this.block = '', this.status = ''});

  final String q;
  final String block;
  final String status;

  UnitsFilter copyWith({String? q, String? block, String? status}) =>
      UnitsFilter(
        q: q ?? this.q,
        block: block ?? this.block,
        status: status ?? this.status,
      );
}

class UnitsState {
  const UnitsState({
    required this.units,
    required this.total,
    this.filter = const UnitsFilter(),
  });

  final List<Unit> units;
  final int total;
  final UnitsFilter filter;
}

/// Per-building unit list (US2/T034); family argument is the building id.
class UnitsController extends FamilyAsyncNotifier<UnitsState, String> {
  @override
  Future<UnitsState> build(String buildingId) =>
      _load(buildingId, const UnitsFilter());

  Future<UnitsState> _load(String buildingId, UnitsFilter f) async {
    final (units, total) = await ref
        .watch(buildingsRepositoryProvider)
        .listUnits(buildingId, q: f.q, block: f.block, status: f.status);
    return UnitsState(units: units, total: total, filter: f);
  }

  Future<void> apply(UnitsFilter f) async {
    state = const AsyncLoading();
    state = await AsyncValue.guard(() => _load(arg, f));
  }

  /// Creates or updates a unit; duplicate-number conflicts throw the 409
  /// [ApiException] for the form to render (FR-003).
  Future<void> saveUnit({
    required String buildingId,
    Unit? existing,
    required Map<String, dynamic> payload,
  }) async {
    if (existing == null) {
      await ref
          .read(buildingsRepositoryProvider)
          .createUnit(buildingId, payload);
    } else {
      await ref
          .read(buildingsRepositoryProvider)
          .updateUnit(existing.id, payload);
    }
    ref.invalidateSelf();
  }

  Future<void> archiveUnit(String unitId) async {
    await ref.read(buildingsRepositoryProvider).archiveUnit(unitId);
    ref.invalidateSelf();
  }
}

final unitsControllerProvider =
    AsyncNotifierProvider.family<UnitsController, UnitsState, String>(
      UnitsController.new,
    );
