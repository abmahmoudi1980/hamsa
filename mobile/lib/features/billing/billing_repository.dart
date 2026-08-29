import 'package:dio/dio.dart';

import 'models/billing.dart';

/// US4 billing API (T050–T053) — mirrors the backend contracts/api.md
/// "Billing" and "Invoices" sections.
class BillingRepository {
  BillingRepository(this._dio);

  final Dio _dio;

  // --- periods ---------------------------------------------------------------

  Future<List<BillingPeriod>> listPeriods(String buildingId) async {
    final res = await _dio.get<Map<String, dynamic>>(
      '/buildings/$buildingId/periods',
    );
    return (res.data?['items'] as List? ?? const [])
        .map((e) => BillingPeriod.fromJson(Map<String, dynamic>.from(e as Map)))
        .toList();
  }

  Future<BillingPeriod> createPeriod(
    String buildingId,
    Map<String, dynamic> payload,
  ) async {
    final res = await _dio.post<Map<String, dynamic>>(
      '/buildings/$buildingId/periods',
      data: payload,
    );
    return BillingPeriod.fromJson(res.data!);
  }

  Future<BillingPeriod> getPeriod(String id) async {
    final res = await _dio.get<Map<String, dynamic>>('/periods/$id');
    return BillingPeriod.fromJson(res.data!);
  }

  Future<BillingPeriod> updatePeriod(
    String id,
    Map<String, dynamic> payload,
  ) async {
    final res = await _dio.patch<Map<String, dynamic>>(
      '/periods/$id',
      data: payload,
    );
    return BillingPeriod.fromJson(res.data!);
  }

  Future<BillingPeriod> reopen(String id) async {
    final res = await _dio.post<Map<String, dynamic>>('/periods/$id/reopen');
    return BillingPeriod.fromJson(res.data!);
  }

  Future<BillingPeriod> close(String id) async {
    final res = await _dio.post<Map<String, dynamic>>('/periods/$id/close');
    return BillingPeriod.fromJson(res.data!);
  }

  // --- cost items --------------------------------------------------------------

  Future<List<CostItem>> listCostItems(String periodId) async {
    final res = await _dio.get<Map<String, dynamic>>(
      '/periods/$periodId/cost-items',
    );
    return (res.data?['items'] as List? ?? const [])
        .map((e) => CostItem.fromJson(Map<String, dynamic>.from(e as Map)))
        .toList();
  }

  Future<CostItem> createCostItem(
    String periodId,
    Map<String, dynamic> payload,
  ) async {
    final res = await _dio.post<Map<String, dynamic>>(
      '/periods/$periodId/cost-items',
      data: payload,
    );
    return CostItem.fromJson(res.data!);
  }

  Future<CostItem> updateCostItem(
    String id,
    Map<String, dynamic> payload,
  ) async {
    final res = await _dio.put<Map<String, dynamic>>(
      '/cost-items/$id',
      data: payload,
    );
    return CostItem.fromJson(res.data!);
  }

  Future<void> deleteCostItem(String id) =>
      _dio.delete<void>('/cost-items/$id');

  // --- calculation / preview ------------------------------------------------------

  /// Runs the charge engine (POST) — response doubles as the fresh preview.
  Future<PeriodPreview> calculate(String periodId) async {
    final res = await _dio.post<Map<String, dynamic>>(
      '/periods/$periodId/calculate',
    );
    return PeriodPreview.fromJson(res.data!);
  }

  Future<PeriodPreview> preview(String periodId) async {
    final res = await _dio.get<Map<String, dynamic>>(
      '/periods/$periodId/preview',
    );
    return PeriodPreview.fromJson(res.data!);
  }

  /// Issues all invoices: freezes snapshots, assigns sequential numbers,
  /// notifies residents (FR-032). Server is idempotent-guarded.
  Future<List<Invoice>> issue(String periodId) async {
    final res = await _dio.post<Map<String, dynamic>>(
      '/periods/$periodId/issue',
    );
    return (res.data?['items'] as List? ?? const [])
        .map((e) => Invoice.fromJson(Map<String, dynamic>.from(e as Map)))
        .toList();
  }

  // --- invoices ---------------------------------------------------------------------

  Future<(List<Invoice>, int)> listBuildingInvoices(
    String buildingId, {
    String? periodId,
    String? unitId,
    String status = '',
    int page = 1,
    int pageSize = 20,
  }) async {
    final res = await _dio.get<Map<String, dynamic>>(
      '/buildings/$buildingId/invoices',
      queryParameters: {
        if (periodId != null) 'period_id': periodId,
        if (unitId != null) 'unit_id': unitId,
        if (status.isNotEmpty) 'status': status,
        'page': page,
        'page_size': pageSize,
      },
    );
    final items = (res.data?['items'] as List? ?? const [])
        .map((e) => Invoice.fromJson(Map<String, dynamic>.from(e as Map)))
        .toList();
    return (items, (res.data?['total'] ?? 0) as int);
  }

  Future<Invoice> getInvoice(String id) async {
    final res = await _dio.get<Map<String, dynamic>>('/invoices/$id');
    return Invoice.fromJson(res.data!);
  }

  Future<Invoice> cancelInvoice(String id, String reason) async {
    final res = await _dio.post<Map<String, dynamic>>(
      '/invoices/$id/cancel',
      data: {'reason': reason},
    );
    return Invoice.fromJson(res.data!);
  }

  Future<void> addAdjustment(
    String invoiceId, {
    required String kind, // debit | credit
    required int amount,
    required String reason,
  }) async {
    await _dio.post<Map<String, dynamic>>(
      '/invoices/$invoiceId/adjustments',
      data: {'kind': kind, 'amount': '$amount', 'reason': reason},
    );
  }

  // --- resident scope ------------------------------------------------------------------

  Future<(List<Invoice>, int)> myInvoices({
    int page = 1,
    int pageSize = 20,
  }) async {
    final res = await _dio.get<Map<String, dynamic>>(
      '/me/invoices',
      queryParameters: {'page': page, 'page_size': pageSize},
    );
    final items = (res.data?['items'] as List? ?? const [])
        .map((e) => Invoice.fromJson(Map<String, dynamic>.from(e as Map)))
        .toList();
    return (items, (res.data?['total'] ?? 0) as int);
  }
}
