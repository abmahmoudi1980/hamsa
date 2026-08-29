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
import 'package:hamsa/features/charges/screens/invoice_detail_screen.dart';
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

/// Unpaid issued invoice; no network involved.
class _FakeBillingRepository extends BillingRepository {
  _FakeBillingRepository() : super(Dio());

  @override
  Future<Invoice> getInvoice(String id) async => Invoice(
    id: id,
    invoiceNumber: 'INV-1',
    unitId: 'unit-1',
    baseAmount: 2600000,
    priorDebt: 0,
    lateFeeAmount: 0,
    creditAmount: 0,
    finalAmount: 2600000,
    status: 'unpaid',
    paidAmount: 0,
    items: const [],
  );
}

Future<void> _pumpAuthenticatedManager(WidgetTester tester) async {
  await tester.pumpWidget(
    ProviderScope(
      overrides: [
        tokenStorageProvider.overrideWithValue(_ManagerStorage()),
        billingRepositoryProvider.overrideWithValue(_FakeBillingRepository()),
      ],
      child: const MainApp(),
    ),
  );
  // Splash → session restore → redirect to the role shell.
  await tester.pumpAndSettle();
}

void main() {
  testWidgets('manager reaches invoice detail and sees record-payment action', (
    tester,
  ) async {
    await _pumpAuthenticatedManager(tester);

    // Open the shared invoice detail route (same push as the period
    // invoice list uses).
    tester.element(find.byType(Scaffold).first).go('/invoice/inv-1');
    await tester.pumpAndSettle();

    // The redirect must NOT bounce the manager off /invoice/:id.
    expect(find.byType(InvoiceDetailScreen), findsOneWidget);
    // US5 (T060): the manual record-payment entry point.
    expect(find.byIcon(Icons.payments_outlined), findsOneWidget);
  });

  testWidgets('authenticated manager on /login/otp lands on manager shell', (
    tester,
  ) async {
    await _pumpAuthenticatedManager(tester);

    // Simulates the post-OTP-verify state: session became authenticated
    // while the router still shows the login screen. The redirect must
    // hop to the role shell (regression: manager stuck on OTP screen).
    tester.element(find.byType(Scaffold).first).go('/login/otp?phone=0912');
    await tester.pumpAndSettle();

    expect(find.byType(InvoiceDetailScreen), findsNothing);
    final context = tester.element(find.byType(Scaffold).first);
    expect(GoRouter.of(context).routerDelegate.currentConfiguration.uri.path,
        startsWith('/manager'));
  });
}
