import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:hamsa/core/datetime/jalali.dart';
import 'package:hamsa/core/storage/token_storage.dart';
import 'package:hamsa/features/auth/auth_controller.dart';
import 'package:hamsa/features/billing/billing_controller.dart';
import 'package:hamsa/features/billing/billing_repository.dart';
import 'package:hamsa/features/billing/models/billing.dart';
import 'package:hamsa/features/buildings/buildings_controller.dart';
import 'package:hamsa/features/buildings/buildings_repository.dart';
import 'package:hamsa/features/buildings/models/building.dart';
import 'package:hamsa/features/charges/screens/invoice_detail_screen.dart';
import 'package:hamsa/features/dashboard/dashboard_controller.dart';
import 'package:hamsa/features/dashboard/dashboard_repository.dart';
import 'package:hamsa/features/dashboard/models/dashboard.dart';
import 'package:hamsa/features/dashboard/screens/manager_dashboard_screen.dart';
import 'package:hamsa/features/payments/models/payment.dart';
import 'package:hamsa/features/payments/payment_controller.dart';
import 'package:hamsa/features/payments/payment_repository.dart';
import 'package:hamsa/features/payments/screens/payment_ledger_screen.dart';
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
    Building(id: 'b1', name: 'کهان'),
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

/// Past-due invoice alerts for units ۱..[count] — one per unit so the
/// collapse test can address each row individually.
List<Map<String, dynamic>> _invoiceAlerts(int count) => [
  for (var i = 1; i <= count; i++)
    {
      'kind': 'past_due_invoice',
      'severity': 2,
      'unit_number': toPersianDigits(i.toString()),
      'amount': '500000',
      'ref_type': 'invoice',
      'ref_id': 'inv-$i',
    },
];

const List<Map<String, dynamic>> _defaultAlerts = [
  {
    'kind': 'past_due_invoice',
    'severity': 2,
    'unit_number': '۱۲',
    'amount': '500000',
    'ref_type': 'invoice',
    'ref_id': 'inv-1',
  },
  {
    'kind': 'debtor_unit',
    'severity': 2,
    'unit_id': 'u12',
    'unit_number': '۱۲',
    'amount': '2600000',
    'ref_type': 'unit',
    'ref_id': 'u12',
  },
];

class _FakeDashboardRepository extends DashboardRepository {
  _FakeDashboardRepository({this.alerts = _defaultAlerts}) : super(Dio());

  final List<Map<String, dynamic>> alerts;

  @override
  Future<BuildingDashboard> buildingDashboard(String buildingId) async =>
      BuildingDashboard.fromJson({
        'building_id': 'b1',
        'unit_count': 10,
        'occupied_unit_count': 8,
        'debtor_unit_count': 3,
        'total_debt': '2600000',
        'month_income': '1500000',
        'month_expense': '800000',
        'open_requests': 2,
        'pending_expenses': 1,
        'month': '2026-08',
        'alerts': alerts,
        'quick_actions': [
          {
            'key': 'issue_charge',
            'title': 'صدور شارژ',
            'path': '/manager/periods/b1',
          },
          {
            'key': 'record_expense',
            'title': 'ثبت هزینه',
            'path': '/manager/expenses/b1/new',
          },
          {
            'key': 'record_payment',
            'title': 'ثبت پرداخت',
            'path': '/manager/ledger/b1',
          },
          {
            'key': 'send_announcement',
            'title': 'انتشار اطلاعیه',
            'path': '/manager/announcements/b1/new',
          },
          {'key': 'open_requests', 'title': 'درخواست‌ها', 'path': '/home'},
        ],
      });
}

class _FakePaymentRepository extends PaymentRepository {
  _FakePaymentRepository() : super(Dio());

  @override
  Future<(List<Payment>, int)> buildingPayments(
    String buildingId, {
    String? unitId,
    String method = '',
    String? from,
    String? to,
    int page = 1,
    int pageSize = 20,
  }) async => (const <Payment>[], 0);
}

class _FakeBillingRepository extends BillingRepository {
  _FakeBillingRepository() : super(Dio());

  @override
  Future<Invoice> getInvoice(String id) async => Invoice(
    id: id,
    invoiceNumber: 'BLD-2026-0001',
    unitId: 'u12',
    unitNumber: '۱۲',
    periodTitle: 'مرداد',
    baseAmount: 2600000,
    priorDebt: 0,
    lateFeeAmount: 0,
    creditAmount: 0,
    finalAmount: 2600000,
    status: 'unpaid',
    paidAmount: 2100000,
    items: const [],
  );
}

Future<void> _pumpDashboard(
  WidgetTester tester, {
  List<Map<String, dynamic>> alerts = _defaultAlerts,
}) async {
  await tester.pumpWidget(
    ProviderScope(
      overrides: [
        tokenStorageProvider.overrideWithValue(_ManagerStorage()),
        buildingsRepositoryProvider.overrideWithValue(
          _FakeBuildingsRepository(),
        ),
        dashboardRepositoryProvider.overrideWithValue(
          _FakeDashboardRepository(alerts: alerts),
        ),
        paymentRepositoryProvider.overrideWithValue(_FakePaymentRepository()),
        billingRepositoryProvider.overrideWithValue(_FakeBillingRepository()),
      ],
      child: const MainApp(),
    ),
  );
  // Splash → session restore → redirect to the role shell.
  await tester.pumpAndSettle();
}

/// Drags the dashboard list forward until [finder] is actually on screen
/// (built ≠ visible: ListView cacheExtent materializes widgets up to
/// ~250px below the fold). The dashboard is taller than a phone viewport,
/// so lower sections (quick actions, finance, facts) need this too.
Future<void> _scrollTo(WidgetTester tester, Finder finder) async {
  final viewHeight =
      tester.view.physicalSize.height / tester.view.devicePixelRatio;

  bool onScreen() {
    final elements = finder.evaluate();
    if (elements.isEmpty) return false;
    final box = elements.first.renderObject! as RenderBox;
    final rect = box.localToGlobal(Offset.zero) & box.size;
    return rect.top >= 0 && rect.bottom <= viewHeight;
  }

  var attempts = 0;
  while (!onScreen() && attempts < 12) {
    await tester.drag(find.byType(ListView), const Offset(0, -250));
    await tester.pumpAndSettle();
    attempts++;
  }
  expect(finder, findsWidgets);
}

void main() {
  testWidgets('dashboard leads with collection hero, not a stat grid', (
    tester,
  ) async {
    // Phone-narrow viewport: the reported bug was actions clipped
    // off-screen, so prove every action stays inside the width here.
    tester.view.physicalSize = const Size(1080, 1920);
    tester.view.devicePixelRatio = 3.0;
    addTearDown(() {
      tester.view.resetPhysicalSize();
      tester.view.resetDevicePixelRatio();
    });
    await _pumpDashboard(tester);
    expect(find.byType(ManagerDashboardScreen), findsOneWidget);

    // Building identity.
    expect(find.text('کهان'), findsOneWidget);
    // Hero: total debt + debtor count + collection-health bar + CTA.
    expect(find.text('مجموع بدهی'), findsOneWidget);
    expect(find.textContaining('۲٬۶۰۰٬۰۰۰'), findsWidgets);
    expect(find.text('۳ واحد بدهکار'), findsOneWidget);
    expect(find.text('۷ از ۱۰ واحد تسویه'), findsOneWidget);
    // The old undifferentiated stat grid is gone…
    expect(find.text('شاخص‌ها'), findsNothing);
    // Alerts: one grouped card with a count badge, amounts intact.
    expect(find.text('هشدارها'), findsOneWidget);
    expect(find.text('۲ هشدار'), findsOneWidget);
    expect(
      find.text('صورتحساب واحد ۱۲ سررسید گذشته'),
      findsOneWidget,
    );

    // Quick access: every action visible at once, none clipped.
    await _scrollTo(tester, find.text('دسترسی سریع'));
    expect(find.text('دسترسی سریع'), findsOneWidget);
    const actionLabels = [
      'صدور شارژ',
      'ثبت هزینه',
      'ثبت پرداخت',
      'انتشار اطلاعیه',
      'درخواست‌ها',
    ];
    for (final label in actionLabels) {
      await _scrollTo(tester, find.text(label));
      final matches = find.text(label);
      expect(matches, findsWidgets, reason: label);
      for (final element in matches.evaluate()) {
        final box = element.renderObject! as RenderBox;
        final rect = box.localToGlobal(Offset.zero) & box.size;
        expect(rect.left, greaterThanOrEqualTo(0), reason: label);
        expect(rect.right, lessThanOrEqualTo(360), reason: label);
      }
    }

    // Finance: Jalali month in the title (2026-08 → مرداد), the two
    // columns, and the net punchline (1500000 − 800000).
    await _scrollTo(tester, find.text('تراز ماه'));
    expect(find.text('درآمد و هزینه مرداد'), findsOneWidget);
    expect(find.text('درآمد'), findsOneWidget);
    expect(find.text('هزینه'), findsOneWidget);
    expect(find.text('تراز ماه'), findsOneWidget);
    expect(find.textContaining('۷۰۰٬۰۰۰'), findsWidgets);

    // Facts: 2×2 glanceable grid instead of stacked rows.
    await _scrollTo(tester, find.text('تعداد واحدها'));
    expect(find.text('تعداد واحدها'), findsOneWidget);
    expect(find.text('واحدهای مسکونی'), findsOneWidget);
    expect(find.text('۱۰'), findsOneWidget);
  });

  testWidgets('hero record-payment CTA opens the payment ledger', (
    tester,
  ) async {
    await _pumpDashboard(tester);

    // FilledButton.icon builds a private _FilledButtonWithIcon subclass, so
    // byType-style finders (widgetWithText) can't see it — match on subtype.
    final heroCta = find
        .ancestor(
          of: find.text('ثبت پرداخت'),
          matching: find.bySubtype<FilledButton>(),
        )
        .first;
    await tester.tap(heroCta);
    await tester.pumpAndSettle();

    expect(find.byType(PaymentLedgerScreen), findsOneWidget);
  });

  testWidgets('past-due alert deep-links to the invoice detail', (
    tester,
  ) async {
    await _pumpDashboard(tester);

    await tester.tap(find.text('صورتحساب واحد ۱۲ سررسید گذشته'));
    await tester.pumpAndSettle();

    expect(find.byType(InvoiceDetailScreen), findsOneWidget);
  });

  testWidgets('long alert lists collapse behind a show-all toggle', (
    tester,
  ) async {
    await _pumpDashboard(tester, alerts: _invoiceAlerts(6));

    // Collapsed: the count badge shows all six, but only the first
    // four rows render; the rest wait behind the toggle.
    expect(find.text('۶ هشدار'), findsOneWidget);
    expect(find.text('صورتحساب واحد ۱ سررسید گذشته'), findsOneWidget);
    expect(find.text('صورتحساب واحد ۵ سررسید گذشته'), findsNothing);
    expect(find.text('نمایش همه (۶)'), findsOneWidget);

    await _scrollTo(tester, find.text('نمایش همه (۶)'));
    await tester.tap(find.text('نمایش همه (۶)'));
    await tester.pumpAndSettle();

    // Expanded: every alert renders and the toggle flips to collapse.
    await _scrollTo(tester, find.text('صورتحساب واحد ۵ سررسید گذشته'));
    expect(find.text('صورتحساب واحد ۵ سررسید گذشته'), findsOneWidget);
    expect(find.text('صورتحساب واحد ۶ سررسید گذشته'), findsOneWidget);
    expect(find.text('نمایش کمتر'), findsOneWidget);
  });
}
