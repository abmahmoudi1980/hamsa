import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../auth/auth_repository.dart';
import 'announcement_repository.dart';
import 'models/announcement.dart';

final announcementRepositoryProvider = Provider<AnnouncementRepository>(
  (ref) => AnnouncementRepository(ref.watch(apiClientProvider).dio),
);

/// Manager: announcements for one building (family: buildingId).
class BuildingAnnouncementsController
    extends FamilyAsyncNotifier<List<Announcement>, String> {
  @override
  Future<List<Announcement>> build(String buildingId) async {
    final (items, _) = await ref
        .watch(announcementRepositoryProvider)
        .buildingAnnouncements(buildingId);
    return items;
  }

  Future<void> refresh() async {
    state = const AsyncLoading<List<Announcement>>().copyWithPrevious(state);
    state = await AsyncValue.guard(() => build(arg));
  }
}

final buildingAnnouncementsControllerProvider = AsyncNotifierProvider.family<
    BuildingAnnouncementsController, List<Announcement>, String>(
  BuildingAnnouncementsController.new,
);

/// Resident: own targeted announcements.
class MyAnnouncementsController extends AsyncNotifier<List<Announcement>> {
  @override
  Future<List<Announcement>> build() async {
    final (items, _) =
        await ref.watch(announcementRepositoryProvider).myAnnouncements();
    return items;
  }

  Future<void> refresh() async {
    state = const AsyncLoading<List<Announcement>>().copyWithPrevious(state);
    state = await AsyncValue.guard(() async {
      final (items, _) =
          await ref.read(announcementRepositoryProvider).myAnnouncements();
      return items;
    });
  }

  int get unreadCount =>
      (state.valueOrNull ?? []).where((a) => !a.isRead).length;
}

final myAnnouncementsControllerProvider =
    AsyncNotifierProvider<MyAnnouncementsController, List<Announcement>>(
  MyAnnouncementsController.new,
);
