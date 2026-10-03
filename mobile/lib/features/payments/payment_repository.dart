import 'package:dio/dio.dart';

import 'models/payment.dart';

/// US5 payments API (T060/T061) — mirrors the backend contracts/api.md
/// "Payments & Balances" section.
class PaymentRepository {
  PaymentRepository(this._dio);

  final Dio _dio;

  /// Manager records a manual payment against an issued invoice.
  Future<Payment> recordManualPayment(
    String invoiceId, {
    required int amount,
    required String paidAt, // ISO YYYY-MM-DD
    String? trackingNumber,
  }) async {
    final res = await _dio.post<Map<String, dynamic>>(
      '/invoices/$invoiceId/payments',
      data: {
        'amount': '$amount',
        'paid_at': paidAt,
        if (trackingNumber != null && trackingNumber.isNotEmpty)
          'tracking_number': trackingNumber,
      },
    );
    return Payment.fromJson(res.data!);
  }

  /// Manager payment ledger for one building (filter by unit/date/method).
  ///
  /// 003-web-frontend F1: both payment lists now answer the standard
  /// `{items, page, page_size, total}` envelope. `payments` is retained by the
  /// server as a deprecated alias for one release cycle, so this reads `items`.
  Future<(List<Payment>, int)> buildingPayments(
    String buildingId, {
    String? unitId,
    String method = '',
    String? from,
    String? to,
    int page = 1,
    int pageSize = 20,
  }) async {
    final res = await _dio.get<Map<String, dynamic>>(
      '/buildings/$buildingId/payments',
      queryParameters: {
        if (unitId != null) 'unit_id': unitId,
        if (method.isNotEmpty) 'method': method,
        if (from != null) 'from': from,
        if (to != null) 'to': to,
        'page': page,
        'page_size': pageSize,
      },
    );
    return _readPage(res.data);
  }

  /// Resident payment history.
  Future<(List<Payment>, int)> myPayments({
    int page = 1,
    int pageSize = 20,
  }) async {
    final res = await _dio.get<Map<String, dynamic>>(
      '/me/payments',
      queryParameters: {'page': page, 'page_size': pageSize},
    );
    return _readPage(res.data);
  }

  /// Unpacks the shared pagination envelope into a list + total.
  ///
  /// The `?? const []` fallbacks are deliberate: an unexpected key must render
  /// an empty list rather than crash the screen, which is exactly how the
  /// `items`-vs-`payments` mismatch went unnoticed (F1).
  (List<Payment>, int) _readPage(Map<String, dynamic>? body) {
    final items = (body?['items'] as List? ?? const [])
        .map((e) => Payment.fromJson(Map<String, dynamic>.from(e as Map)))
        .toList();
    final total = (body?['total'] as num?)?.toInt() ?? items.length;
    return (items, total);
  }

  /// Unit balance (spec §9 components).
  Future<UnitBalance> unitBalance(String unitId) async {
    final res = await _dio.get<Map<String, dynamic>>('/units/$unitId/balance');
    return UnitBalance.fromJson(res.data!);
  }

  /// Resident starts an online payment (amount defaults to the outstanding).
  Future<GatewayStart> startGatewayPayment(String invoiceId, {int? amount}) async {
    final res = await _dio.post<Map<String, dynamic>>(
      '/invoices/$invoiceId/pay',
      data: amount == null ? <String, dynamic>{} : {'amount': '$amount'},
    );
    return GatewayStart.fromJson(res.data!);
  }
}
