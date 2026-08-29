import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../auth/auth_repository.dart';
import 'models/payment.dart';
import 'payment_repository.dart';

/// US5 payment DI + list controllers (T060/T061).

final paymentRepositoryProvider = Provider<PaymentRepository>(
  (ref) => PaymentRepository(ref.watch(apiClientProvider).dio),
);

/// Manager payment ledger for one building (family argument: building id).
class BuildingPaymentsController
    extends FamilyAsyncNotifier<List<Payment>, String> {
  @override
  Future<List<Payment>> build(String buildingId) async {
    final (items, _) = await ref
        .watch(paymentRepositoryProvider)
        .buildingPayments(buildingId);
    return items;
  }

  Future<void> refresh() async {
    state = const AsyncLoading<List<Payment>>().copyWithPrevious(state);
    state = await AsyncValue.guard(() => build(arg));
  }
}

final buildingPaymentsControllerProvider = AsyncNotifierProvider.family<
    BuildingPaymentsController, List<Payment>, String>(
  BuildingPaymentsController.new,
);

/// Resident payment history (/me/payments).
class MyPaymentsController extends AsyncNotifier<List<Payment>> {
  @override
  Future<List<Payment>> build() async {
    final (items, _) = await ref.watch(paymentRepositoryProvider).myPayments();
    return items;
  }

  Future<void> refresh() async {
    state = const AsyncLoading<List<Payment>>().copyWithPrevious(state);
    state = await AsyncValue.guard(() => build());
  }
}

final myPaymentsControllerProvider =
    AsyncNotifierProvider<MyPaymentsController, List<Payment>>(
  MyPaymentsController.new,
);

/// Unit balance (spec §9 components) — family argument: unit id.
class UnitBalanceController extends FamilyAsyncNotifier<UnitBalance?, String> {
  @override
  Future<UnitBalance?> build(String unitId) async {
    try {
      return await ref.watch(paymentRepositoryProvider).unitBalance(unitId);
    } on DioException catch (e) {
      if (e.response?.statusCode == 404) return null; // no ledger history yet
      rethrow;
    }
  }
}

final unitBalanceControllerProvider = AsyncNotifierProvider.family<
    UnitBalanceController, UnitBalance?, String>(
  UnitBalanceController.new,
);
