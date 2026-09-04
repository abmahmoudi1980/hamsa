import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:hamsa/core/storage/token_storage.dart';
import 'package:hamsa/features/auth/auth_controller.dart';
import 'package:hamsa/features/billing/billing_controller.dart';
import 'package:hamsa/features/billing/billing_repository.dart';
import 'package:hamsa/features/billing/models/billing.dart';
import 'package:hamsa/features/buildings/buildings_controller.dart';
import 'package:hamsa/features/buildings/buildings_repository.dart';
import 'package:hamsa/features/buildings/models/building.dart';
import 'package:hamsa/features/payments/models/payment.dart';
import 'package:hamsa/features/payments/payment_controller.dart';
import 'package:hamsa/features/payments/payment_repository.dart';
import 'package:hamsa/features/payments/screens/payment_ledger_screen.dart';
import 'package:hamsa/features/payments/screens/record_payment_screen.dart';
import 'package:hamsa/features/payments/screens/select_invoice_screen.dart';
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

Invoice _invoice({
  required String id,
  required String number,
  required String unitNumber,
  required int finalAmount,
  required int paidAmount,
  required String status,
  bool numberPresent = true,
}) => Invoice(
  id: id,
  invoiceNumber: number,
  invoiceNumberPresent: numberPresent,
  unitId: 'unit-$unitNumber',
  unitNumber: unitNumber,
  periodTitle: 'مرداد',
  baseAmount: finalAmount,
  priorDebt: 0,
  lateFeeAmount: 0,
  creditAmount: 0,
  finalAmount: finalAmount,
  status: status,
  paidAmount: paidAmount,
  items: const [],
);

/// Building invoices mixing payable and non-payable rows.
class _FakeBillingRepository extends BillingRepository {
  _FakeBillingRepository() : super(Dio());

  @override
  Future<(List<Invoice>, int)> listBuildingInvoices(
    String buildingId, {
    String? periodId,
    String? unitId,
    String status = '',
    int page = 1,
    int pageSize = 20,
  }) async => (
    [
      _invoice(
        id: 'inv-unpaid',
        number: 'BLD-2026-0001',
        unitNumber: '12',
        finalAmount: 2600000,
        paidAmount: 0,
        status: 'unpaid',
      ),
      _invoice(
        id: 'inv-partial',
        number: 'BLD-2026-0002',
        unitNumber: '14',
        finalAmount: 2600000,
        paidAmount: 1500000,
        status: 'partial',
      ),
      _invoice(
        id: 'inv-paid',
        number: 'BLD-2026-0003',
        unitNumber: '99',
        finalAmount: 2600000,
        paidAmount: 2600000,
        status: 'paid',
      ),
      _invoice(
        id: 'inv-cancelled',
        number: 'BLD-2026-0004',
        unitNumber: '50',
        finalAmount: 2600000,
        paidAmount: 0,
        status: 'cancelled',
      ),
      _invoice(
        id: 'inv-draft',
        number: '',
        unitNumber: '7',
        finalAmount: 2600000,
        paidAmount: 0,
        status: 'unpaid',
        numberPresent: false,
      ),
    ],
    5,
  );
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

class _FakeBuildingsRepository extends BuildingsRepository {
  _FakeBuildingsRepository() : super(Dio());

  @override
  Future<(List<Unit>, int)> listUnits(
    String buildingId, {
    String q = '',
    String block = '',
    int? floor,
    String status = '',
    int page = 1,
    int pageSize = 20,
  }) async => (
    [
      Unit(id: 'u12', buildingId: 'b1', number: '12', areaM2: 80),
    ],
    1,
  );
}

Future<void> _pumpAuthenticatedManager(WidgetTester tester) async {
  await tester.pumpWidget(
    ProviderScope(
      overrides: [
        tokenStorageProvider.overrideWithValue(_ManagerStorage()),
        billingRepositoryProvider.overrideWithValue(_FakeBillingRepository()),
        paymentRepositoryProvider.overrideWithValue(_FakePaymentRepository()),
        buildingsRepositoryProvider.overrideWithValue(
          _FakeBuildingsRepository(),
        ),
      ],
      child: const MainApp(),
    ),
  );
  // Splash → session restore → redirect to the role shell.
  await tester.pumpAndSettle();
}

void _go(WidgetTester tester, String location) {
  tester.element(find.byType(Scaffold).first).go(location);
}

void main() {
  testWidgets(
    'ledger offers record-payment and the picker lists only payable invoices',
    (tester) async {
      await _pumpAuthenticatedManager(tester);

      _go(tester, '/manager/ledger/b1');
      await tester.pumpAndSettle();
      expect(find.byType(PaymentLedgerScreen), findsOneWidget);

      // The reported gap: the ledger had no way to start a manual payment.
      expect(find.byType(FloatingActionButton), findsOneWidget);
      await tester.tap(find.byType(FloatingActionButton));
      await tester.pumpAndSettle();

      expect(find.byType(SelectInvoiceScreen), findsOneWidget);
      // Payable rows surface with Persian unit numbers…
      expect(find.text('شماره واحد ۱۲'), findsOneWidget);
      expect(find.text('شماره واحد ۱۴'), findsOneWidget);
      // …partial shows the remaining balance (۲٬۶۰۰٬۰۰۰ − ۱٬۵۰۰٬۰۰۰)…
      expect(find.textContaining('۱٬۱۰۰٬۰۰۰'), findsOneWidget);
      // …and paid/cancelled/unissued rows are never offered (server 409s
      // payments against anything but unpaid/partial).
      expect(find.textContaining('۹۹'), findsNothing);
      expect(find.textContaining('۵۰'), findsNothing);
      expect(find.textContaining('۷'), findsNothing);
    },
  );
  testWidgets('picking an invoice opens the record-payment form', (
    tester,
  ) async {
    await _pumpAuthenticatedManager(tester);

    _go(tester, '/manager/select-invoice/b1');
    await tester.pumpAndSettle();

    await tester.tap(find.text('شماره واحد ۱۲'));
    await tester.pumpAndSettle();

    expect(find.byType(RecordPaymentScreen), findsOneWidget);
    expect(find.text('ثبت پرداخت'), findsOneWidget);
    // Regression: the amount field once shipped without a label.
    expect(find.text('مبلغ پرداخت (تومان)'), findsOneWidget);
  });
}
