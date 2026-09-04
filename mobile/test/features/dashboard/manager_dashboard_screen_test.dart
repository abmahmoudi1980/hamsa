import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
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

class _FakeDashboardRepository extends DashboardRepository {
  _FakeDashboardRepository() : super(Dio());

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
        'alerts': [
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
        ],
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

Future<void> _pumpDashboard(WidgetTester tester) async {
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
        paymentRepositoryProvider.overrideWithValue(_FakePaymentRepository()),
        billingRepositoryProvider.overrideWithValue(_FakeBillingRepository()),
      ],
      child: const MainApp(),
    ),
  );
  // Splash → session restore → redirect to the role shell.
  await tester.pumpAndSettle();
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
    // Hero: total debt + debtor count + the single money-moving CTA.
    expect(find.text('مجموع بدهی'), findsOneWidget);
    expect(find.textContaining('۲٬۶۰۰٬۰۰۰'), findsWidgets);
    expect(find.text('۳ واحد بدهکار'), findsOneWidget);
    // The old undifferentiated stat grid is gone…
    expect(find.text('شاخص‌ها'), findsNothing);
    // …replaced by glanceable finance/facts rows.
    expect(find.text('درآمد ماه'), findsOneWidget);
    expect(find.text('هزینه ماه'), findsOneWidget);
    expect(find.text('تعداد واحدها'), findsOneWidget);
    expect(find.textContaining('۱۰'), findsWidgets);
    // Quick access: every action visible at once, none clipped.
    expect(find.text('دسترسی سریع'), findsOneWidget);
    const actionLabels = [
      'صدور شارژ',
      'ثبت هزینه',
      'ثبت پرداخت',
      'انتشار اطلاعیه',
      'درخواست‌ها',
    ];
    for (final label in actionLabels) {
      final matches = find.widgetWithText(FilledButton, label);
      expect(matches, findsWidgets, reason: label);
      for (final element in matches.evaluate()) {
        final box = element.renderObject! as RenderBox;
        final rect = box.localToGlobal(Offset.zero) & box.size;
        expect(rect.left, greaterThanOrEqualTo(0), reason: label);
        expect(rect.right, lessThanOrEqualTo(360), reason: label);
      }
    }
    // …and alerts surface with their amounts.
    expect(find.text('هشدارها'), findsOneWidget);
    expect(
      find.text('صورتحساب واحد ۱۲ سررسید گذشته'),
      findsOneWidget,
    );
  });

  testWidgets('hero record-payment CTA opens the payment ledger', (
    tester,
  ) async {
    await _pumpDashboard(tester);

    await tester.tap(find.widgetWithText(FilledButton, 'ثبت پرداخت').first);
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
}
