import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../core/network/api_exception.dart';
import 'buildings_controller.dart';
import 'buildings_repository.dart';
import 'models/people.dart';

final peopleRepositoryProvider = Provider<BuildingsRepository>(
  (ref) => ref.watch(buildingsRepositoryProvider),
);

/// Person list (US3/T040); family argument is the building id. Loading/error
/// ride on AsyncValue; API failures surface as [ApiException] and resolve to
/// Persian copy in the screen.
class PeopleController extends FamilyAsyncNotifier<List<Person>, String> {
  @override
  Future<List<Person>> build(String buildingId) => ref
      .watch(peopleRepositoryProvider)
      .listPersons(buildingId)
      .then((r) => r.$1);

  BuildingsRepository get _repo => ref.read(peopleRepositoryProvider);

  /// Creates or updates; rethrows [ApiException] for the form to display.
  Future<void> save({
    required String buildingId,
    Person? existing,
    required Map<String, dynamic> payload,
  }) async {
    if (existing == null) {
      await _repo.createPerson(buildingId, payload);
    } else {
      await _repo.updatePerson(existing.id, payload);
    }
    ref.invalidateSelf();
  }

  Future<void> archive(String id) async {
    await _repo.archivePerson(id);
    ref.invalidateSelf();
  }
}

final peopleControllerProvider =
    AsyncNotifierProvider.family<PeopleController, List<Person>, String>(
      PeopleController.new,
    );

/// Occupancy + occupant-count state of one unit (US3/T041); family argument
/// is the unit id.
class OccupancyState {
  const OccupancyState({required this.occupancies, required this.counts});

  final List<Occupancy> occupancies;
  final List<OccupantCountEntry> counts;
}

class OccupancyController extends FamilyAsyncNotifier<OccupancyState, String> {
  @override
  Future<OccupancyState> build(String unitId) => _load(unitId);

  Future<OccupancyState> _load(String unitId) async {
    final repo = ref.watch(peopleRepositoryProvider);
    final occupancies = await repo.listOccupancies(unitId);
    final counts = await repo.listOccupantCounts(unitId);
    return OccupancyState(occupancies: occupancies, counts: counts);
  }

  /// Adds a person-unit link; replacing an active tenant closes the previous
  /// occupancy server-side (FR-007).
  Future<void> addOccupancy(Map<String, dynamic> payload) async {
    await ref.read(peopleRepositoryProvider).addOccupancy(arg, payload);
    ref.invalidateSelf();
  }

  /// End-dates an occupancy row; the row is preserved (FR-007).
  Future<void> endOccupancy(String occupancyId, String endDate) async {
    await ref.read(peopleRepositoryProvider).endOccupancy(occupancyId, endDate);
    ref.invalidateSelf();
  }

  /// Appends one occupant-count data point (FR-008/BR-04).
  Future<void> recordCount({
    required int count,
    required String effectiveFrom,
  }) async {
    await ref
        .read(peopleRepositoryProvider)
        .recordOccupantCount(arg, count: count, effectiveFrom: effectiveFrom);
    ref.invalidateSelf();
  }
}

final occupancyControllerProvider =
    AsyncNotifierProvider.family<OccupancyController, OccupancyState, String>(
      OccupancyController.new,
    );

/// Rethrows server Persian messages as [ApiException] so forms can render
/// them (mirrors BuildingsController.saveUnit error flow).
Never rethrowApi(Object error) {
  if (error is ApiException) throw error;
  throw StateError('unexpected error: $error');
}
