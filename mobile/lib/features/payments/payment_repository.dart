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
        'size': pageSize,
      },
    );
    final items = (res.data?['payments'] as List? ?? const [])
        .map((e) => Payment.fromJson(Map<String, dynamic>.from(e as Map)))
        .toList();
    return (items, (res.data?['total'] ?? 0) as int);
  }

  /// Resident payment history.
  Future<(List<Payment>, int)> myPayments({
    int page = 1,
    int pageSize = 20,
  }) async {
    final res = await _dio.get<Map<String, dynamic>>(
      '/me/payments',
      queryParameters: {'page': page, 'size': pageSize},
    );
    final items = (res.data?['items'] as List? ?? const [])
        .map((e) => Payment.fromJson(Map<String, dynamic>.from(e as Map)))
        .toList();
    return (items, (res.data?['total'] ?? 0) as int);
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
