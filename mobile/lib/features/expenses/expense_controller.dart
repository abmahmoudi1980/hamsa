import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../auth/auth_repository.dart';
import 'expense_repository.dart';
import 'models/expense.dart';

/// US6 expense DI + list controller (T066/T067).

final expenseRepositoryProvider = Provider<ExpenseRepository>(
  (ref) => ExpenseRepository(ref.watch(apiClientProvider).dio),
);

/// Manager expense list for one building (family argument: building id).
class BuildingExpensesController
    extends FamilyAsyncNotifier<List<Expense>, String> {
  @override
  Future<List<Expense>> build(String buildingId) async {
    final (items, _) = await ref
        .watch(expenseRepositoryProvider)
        .buildingExpenses(buildingId);
    return items;
  }

  Future<void> refresh() async {
    state = const AsyncLoading<List<Expense>>().copyWithPrevious(state);
    state = await AsyncValue.guard(() => build(arg));
  }
}

final buildingExpensesControllerProvider = AsyncNotifierProvider.family<
    BuildingExpensesController, List<Expense>, String>(
  BuildingExpensesController.new,
);
