import 'package:hamsa/core/theme/app_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:hamsa/core/l10n/app_localizations.dart';
import 'package:hamsa/features/buildings/buildings_controller.dart';
import 'package:hamsa/features/buildings/buildings_repository.dart';
import 'package:hamsa/features/buildings/screens/building_managers_screen.dart';

import '../stub_http_adapter.dart';

const _managerRow =
    '{"user_id":"u1","phone":"09120000001","name":"رضا مدیر",'
    '"role":"manager","granted_at":"2026-08-20T10:00:00Z"}';
void main() {
  StubAdapter adapterWith(Map<String, (int, String)> extra) => StubAdapter({
    'GET /buildings/b1/managers': (200, '{"items":[$_managerRow]}'),
    ...extra,
  });

  Future<void> pumpManagers(WidgetTester tester, StubAdapter adapter) async {
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          buildingsRepositoryProvider.overrideWithValue(
            BuildingsRepository(stubDio(adapter)),
          ),
        ],
        child: const MaterialApp(
          locale: Locale('fa'),
          supportedLocales: AppLocalizations.supportedLocales,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          home: BuildingManagersScreen(buildingId: 'b1'),
        ),
      ),
    );
    await tester.pumpAndSettle();
  }

  // Drain the success/error snackbar timers so no pending timer outlives the
  // test; call before the final `expect` of transient snackbar copy is gone.
  Future<void> settleSnackbars(WidgetTester tester) =>
      tester.pumpAndSettle(const Duration(seconds: 9));

  testWidgets('renders the manager list with phone and Jalali date', (
    tester,
  ) async {
    await pumpManagers(tester, adapterWith({}));

    expect(find.text('مدیران ساختمان'), findsOneWidget);
    expect(find.text('رضا مدیر'), findsOneWidget);
    expect(find.textContaining('۰۹۱۲۰۰۰۰۰۰۱'), findsOneWidget);
    // 2026-08-20 (UTC) → mid-1405 Jalali; the day itself is TZ-tolerant.
    expect(find.textContaining('از تاریخ ۱۴۰۵'), findsOneWidget);
  });

  testWidgets('empty list renders the empty state', (tester) async {
    await pumpManagers(
      tester,
      StubAdapter({'GET /buildings/b1/managers': (200, '{"items":[]}')}),
    );

    expect(
      find.text('هنوز مدیری برای این ساختمان ثبت نشده است.'),
      findsOneWidget,
    );
  });

  testWidgets('add posts the phone and reloads the list', (tester) async {
    final adapter = adapterWith({
      'POST /buildings/b1/managers': (
        201,
        '{"user_id":"u2","phone":"09120000002","name":"سارا",'
            '"role":"manager","granted_at":"2026-08-21T10:00:00Z"}',
      ),
    });
    await pumpManagers(tester, adapter);

    await tester.enterText(find.byType(TextFormField), '09120000002');
    await tester.tap(find.text('افزودن مدیر'));
    // Let the Dio chain complete (adapter await needs more than one frame).
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 50));

    final post = adapter.requests.where((r) => r.method == 'POST').single;
    expect(post.path, '/buildings/b1/managers');
    expect(post.body, {'phone': '09120000002'});
    expect(find.text('مدیر با موفقیت اضافه شد.'), findsOneWidget);
    await settleSnackbars(tester);
    // The add invalidated the list: a second GET has been issued.
    expect(
      adapter.requests.where((r) => r.method == 'GET').length,
      greaterThan(1),
    );
  });

  testWidgets('duplicate grant surfaces the server 409 verbatim', (
    tester,
  ) async {
    final adapter = adapterWith({
      'POST /buildings/b1/managers': (
        409,
        '{"error":{"code":"CONFLICT",'
            '"message":"این مدیر از قبل دسترسی دارد."}}',
      ),
    });
    await pumpManagers(tester, adapter);

    await tester.enterText(find.byType(TextFormField), '09120000001');
    await tester.tap(find.text('افزودن مدیر'));
    // Let the Dio chain complete (adapter await needs more than one frame).
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 50));

    expect(find.text('این مدیر از قبل دسترسی دارد.'), findsOneWidget);
    await settleSnackbars(tester);
  });

  testWidgets('remove asks confirm before issuing DELETE', (tester) async {
    final adapter = adapterWith({
      'DELETE /buildings/b1/managers/u1': (204, ''),
    });
    await pumpManagers(tester, adapter);

    await tester.tap(find.byIcon(Icons.person_remove_outlined).first);
    await tester.pumpAndSettle();

    // The confirm dialog is open — nothing revoked yet.
    expect(
      find.text('آیا مطمئنید می‌خواهید دسترسی این مدیر از ساختمان حذف شود؟'),
      findsOneWidget,
    );
    expect(adapter.requests.where((r) => r.method == 'DELETE'), isEmpty);

    await tester.tap(find.text('حذف'));
    // Let the Dio chain complete (adapter await needs more than one frame).
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 50));

    final del = adapter.requests.where((r) => r.method == 'DELETE').single;
    expect(del.path, '/buildings/b1/managers/u1');
    expect(find.text('دسترسی مدیر با موفقیت حذف شد.'), findsOneWidget);
    await settleSnackbars(tester);
  });

  testWidgets('last-manager conflict surfaces the server 409 verbatim', (
    tester,
  ) async {
    final adapter = adapterWith({
      'DELETE /buildings/b1/managers/u1': (
        409,
        '{"error":{"code":"CONFLICT",'
            '"message":"حداقل یک مدیر باید برای ساختمان باقی بماند."}}',
      ),
    });
    await pumpManagers(tester, adapter);

    await tester.tap(find.byIcon(Icons.person_remove_outlined).first);
    await tester.pumpAndSettle();
    await tester.tap(find.text('حذف'));
    // Let the Dio chain complete (adapter await needs more than one frame).
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 50));

    expect(
      find.text('حداقل یک مدیر باید برای ساختمان باقی بماند.'),
      findsOneWidget,
    );
    await settleSnackbars(tester);
  });

  // Regression: AppTheme button styles previously used Size.fromHeight as
  // minimumSize (infinite minimum width), which threw "BoxConstraints forces
  // an infinite width" for the add-manager FilledButton sitting in the form
  // Row — every screen test passed because it pumped the default theme.
  testWidgets('app theme renders the add-manager form at desktop width', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(1400, 900);
    tester.view.devicePixelRatio = 1.0;
    addTearDown(tester.view.reset);
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          buildingsRepositoryProvider.overrideWithValue(
            BuildingsRepository(stubDio(adapterWith({}))),
          ),
        ],
        child: MaterialApp(
          theme: AppTheme.light,
          locale: const Locale('fa'),
          supportedLocales: AppLocalizations.supportedLocales,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          home: const BuildingManagersScreen(buildingId: 'b1'),
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(tester.takeException(), isNull);
    final addFinder = find.widgetWithText(FilledButton, 'افزودن مدیر');
    expect(addFinder, findsOneWidget);
    // Hug-content width, not the crashed infinite/stretched layout.
    expect(tester.getSize(addFinder).width, lessThan(300));
  });
}
