import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:hamsa/core/storage/token_storage.dart';
import 'package:hamsa/features/auth/auth_controller.dart';
import 'package:hamsa/features/buildings/buildings_controller.dart';
import 'package:hamsa/features/buildings/buildings_repository.dart';
import 'package:hamsa/features/buildings/models/building.dart';
import 'package:hamsa/features/buildings/screens/building_list_screen.dart';
import 'package:hamsa/features/dashboard/dashboard_controller.dart';
import 'package:hamsa/features/dashboard/dashboard_repository.dart';
import 'package:hamsa/features/dashboard/models/dashboard.dart';
import 'package:hamsa/main.dart';

/// Serves a persisted manager session without touching platform channels.
class _ManagerStorage extends TokenStorage {
  @override
  Future<StoredSession?> read() async => const StoredSession(
    accessToken: 'a',
    refreshToken: 'r',
    userJson: '{"id":"u1","name":"مدیر","role":"manager"}',
  );
}

class _FakeBuildingsRepository extends BuildingsRepository {
  _FakeBuildingsRepository() : super(Dio());

  @override
  Future<List<Building>> listBuildings() async => [
    Building(
      id: 'b1',
      name: 'برج کهان',
      address: 'خیابان آزادی، پلاک ۱۰',
      unitCount: 12,
    ),
    Building(
      id: 'b2',
      name: 'ساختمان بهار',
      address: 'میدان انقلاب، کوچه مهر',
      unitCount: 8,
    ),
  ];

  @override
  Future<(List<Unit>, int)> listUnits(
    String buildingId, {
    String q = '',
    String block = '',
    int? floor,
    String status = '',
    int page = 1,
    int pageSize = 20,
  }) async => (const <Unit>[], 0);
}

/// Minimal dashboard payload — the test only passes through the dashboard
/// on its way to the buildings list.
class _FakeDashboardRepository extends DashboardRepository {
  _FakeDashboardRepository() : super(Dio());

  @override
  Future<BuildingDashboard> buildingDashboard(String buildingId) async =>
      BuildingDashboard.fromJson(const {});
}

Future<void> _pumpBuildingList(WidgetTester tester) async {
  await tester.pumpWidget(
    ProviderScope(
      overrides: [
        tokenStorageProvider.overrideWithValue(_ManagerStorage()),
        buildingsRepositoryProvider.overrideWithValue(
          _FakeBuildingsRepository(),
        ),
        dashboardRepositoryProvider.overrideWithValue(
          _FakeDashboardRepository(),
        ),
      ],
      child: const MainApp(),
    ),
  );
  // Splash → session restore → redirect to the role shell.
  await tester.pumpAndSettle();

  // Dashboard AppBar → buildings list.
  await tester.tap(find.byTooltip('ساختمان‌ها'));
  await tester.pumpAndSettle();

  expect(find.byType(BuildingListScreen), findsOneWidget);
}

void main() {
  testWidgets('building name and address stay on one line at phone width', (
    tester,
  ) async {
    // Phone-narrow viewport: with the actions in ListTile.trailing the
    // title column collapsed to ~one character wide and every name/address
    // stacked vertically; pumpAndSettle itself throws on that overflow.
    tester.view.physicalSize = const Size(1080, 1920);
    tester.view.devicePixelRatio = 3.0;
    addTearDown(() {
      tester.view.resetPhysicalSize();
      tester.view.resetDevicePixelRatio();
    });
    await _pumpBuildingList(tester);

    // Name on its own single line…
    expect(find.text('برج کهان'), findsOneWidget);
    expect(tester.widget<Text>(find.text('برج کهان')).maxLines, 1);
    // …unit count + address together on the next single line…
    final address = find.textContaining('خیابان آزادی');
    expect(address, findsOneWidget);
    expect(tester.widget<Text>(address).maxLines, 1);
    expect(find.textContaining('۱۲ واحد'), findsOneWidget);
    // …and all four per-building actions still render below (once per
    // building row).
    expect(find.byTooltip('مدیران ساختمان'), findsNWidgets(2));
    expect(find.byTooltip('افراد و ساکنان'), findsNWidgets(2));
    expect(find.byTooltip('شارژها'), findsNWidgets(2));
    expect(find.byTooltip('هزینه‌ها'), findsNWidgets(2));
  });
}
