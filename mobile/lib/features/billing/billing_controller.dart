import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../auth/auth_repository.dart';
import 'billing_repository.dart';
import 'models/billing.dart';

final billingRepositoryProvider = Provider<BillingRepository>(
  (ref) => BillingRepository(ref.watch(apiClientProvider).dio),
);

/// Period list of one building (US4/T050) — family argument is the building id.
class PeriodsController extends FamilyAsyncNotifier<List<BillingPeriod>, String> {
  @override
  Future<List<BillingPeriod>> build(String buildingId) =>
      ref.watch(billingRepositoryProvider).listPeriods(buildingId);

  BillingRepository get _repo => ref.read(billingRepositoryProvider);

  /// Creates or updates a period; rethrows [ApiException] for the form.
  Future<BillingPeriod> save(BillingPeriod? existing, Map<String, dynamic> payload) async {
    final saved = existing == null
        ? await _repo.createPeriod(arg, payload)
        : await _repo.updatePeriod(existing.id, payload);
    ref.invalidateSelf();
    return saved;
  }

  /// calculated → draft, discarding previews (409 after issue).
  Future<void> reopen(String periodId) async {
    await _repo.reopen(periodId);
    ref.invalidateSelf();
  }

  /// issued → closed.
  Future<void> close(String periodId) async {
    await _repo.close(periodId);
    ref.invalidateSelf();
  }
}

final periodsControllerProvider = AsyncNotifierProvider.family<
    PeriodsController, List<BillingPeriod>, String>(
  PeriodsController.new,
);

/// Cost items of one period (T051) — family argument is the period id.
class CostItemsController
    extends FamilyAsyncNotifier<List<CostItem>, String> {
  @override
  Future<List<CostItem>> build(String periodId) =>
      ref.watch(billingRepositoryProvider).listCostItems(periodId);

  BillingRepository get _repo => ref.read(billingRepositoryProvider);

  Future<void> save(CostItem? existing, Map<String, dynamic> payload) async {
    if (existing == null) {
      await _repo.createCostItem(arg, payload);
    } else {
      await _repo.updateCostItem(existing.id, payload);
    }
    ref.invalidateSelf();
    // The period detail screen renders cost items from the preview payload;
    // refresh it so the new/edited item shows up when the user pops back.
    ref.invalidate(previewControllerProvider(arg));
  }

  Future<void> delete(String id) async {
    await _repo.deleteCostItem(id);
    ref.invalidateSelf();
    ref.invalidate(previewControllerProvider(arg));
  }
}

final costItemsControllerProvider = AsyncNotifierProvider.family<
    CostItemsController, List<CostItem>, String>(
  CostItemsController.new,
);

/// Calculate/preview state for one period (T052). Mutations run through
/// [actions] so the screen shows progress and Persian error copy.
class PreviewState {
  const PreviewState({required this.preview, this.busy = false});

  final PeriodPreview preview;
  final bool busy;

  PreviewState copyWith({PeriodPreview? preview, bool? busy}) =>
      PreviewState(
        preview: preview ?? this.preview,
        busy: busy ?? this.busy,
      );
}

class PreviewController
    extends FamilyAsyncNotifier<PreviewState, String> {
  @override
  Future<PreviewState> build(String periodId) async {
    final p = await ref.watch(billingRepositoryProvider).preview(periodId);
    return PreviewState(preview: p);
  }

  BillingRepository get _repo => ref.read(billingRepositoryProvider);

  /// Runs the charge engine and refreshes the preview.
  Future<void> calculate() async {
    await _setBusy(() async {
      final p = await _repo.calculate(arg);
      state = AsyncData(PreviewState(preview: p));
    });
  }

  /// Issues all invoices (confirmation lives in the UI).
  Future<List<Invoice>> issue() async {
    final issued = await _repo.issue(arg);
    final p = await _repo.preview(arg);
    state = AsyncData(PreviewState(preview: p));
    return issued;
  }

  Future<void> reopen() async {
    await _setBusy(() async {
      await _repo.reopen(arg);
      ref.invalidateSelf();
    });
  }

  Future<void> close() async {
    await _setBusy(() async {
      await _repo.close(arg);
      ref.invalidateSelf();
    });
  }

  Future<void> _setBusy(Future<void> Function() action) async {
    final current = state.value;
    if (current != null) {
      state = AsyncData(current.copyWith(busy: true));
    }
    try {
      await action();
    } finally {
      // Always clear busy — even when the action throws (e.g. calculate
      // rejected) — otherwise the action buttons stay disabled forever.
      final after = state.value;
      if (after != null && after.busy) {
        state = AsyncData(after.copyWith(busy: false));
      }
    }
  }
}

final previewControllerProvider = AsyncNotifierProvider.family<
    PreviewController, PreviewState, String>(
  PreviewController.new,
);

/// Resident charges list (T053) — /me/invoices, newest first.
class MyInvoicesController extends AsyncNotifier<List<Invoice>> {
  @override
  Future<List<Invoice>> build() async {
    final (items, _) = await ref.watch(billingRepositoryProvider).myInvoices();
    return items;
  }
}

final myInvoicesControllerProvider =
    AsyncNotifierProvider<MyInvoicesController, List<Invoice>>(
      MyInvoicesController.new,
    );

/// Manager invoice list for one building (manager ledger entry point).
class BuildingInvoicesController
    extends FamilyAsyncNotifier<List<Invoice>, String> {
  @override
  Future<List<Invoice>> build(String buildingId) async {
    final (items, _) = await ref
        .watch(billingRepositoryProvider)
        .listBuildingInvoices(buildingId);
    return items;
  }
}

final buildingInvoicesControllerProvider =
    AsyncNotifierProvider.family<BuildingInvoicesController, List<Invoice>, String>(
      BuildingInvoicesController.new,
    );
