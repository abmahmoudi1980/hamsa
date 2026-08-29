import '../../billing/models/billing.dart';

/// US5 payment DTOs (T060/T061) — mirrors the backend contracts/api.md
/// "Payments & Balances" section. Money is integer Toman serialized as JSON
/// strings on the wire, parsed via [parseMoney].

class Payment {
  Payment({
    required this.id,
    required this.buildingId,
    required this.unitId,
    this.invoiceId,
    required this.method,
    required this.amount,
    required this.paidAt,
    this.trackingNumber,
    required this.status,
    this.invoiceNumber,
    this.unitNumber,
  });

  factory Payment.fromJson(Map<String, dynamic> j) => Payment(
        id: j['id'].toString(),
        buildingId: j['building_id'].toString(),
        unitId: j['unit_id'].toString(),
        invoiceId: j['invoice_id']?.toString(),
        method: j['method']?.toString() ?? 'manual',
        amount: parseMoney(j['amount']),
        paidAt: j['paid_at'].toString(),
        trackingNumber: j['tracking_number']?.toString(),
        status: j['status']?.toString() ?? 'recorded',
        invoiceNumber: j['invoice_number']?.toString(),
        unitNumber: j['unit_number']?.toString(),
      );

  final String id;
  final String buildingId;
  final String unitId;
  final String? invoiceId;
  final String method; // manual | gateway
  final int amount;
  final String paidAt; // ISO YYYY-MM-DD
  final String? trackingNumber;
  final String status; // recorded | verified | failed | reversed
  final String? invoiceNumber;
  final String? unitNumber;
}

/// Unit financial position (spec §9 components). All values are integer
/// Toman (already unwrapped from the wire strings).
class UnitBalance {
  UnitBalance({
    required this.unitId,
    required this.priorDebt,
    required this.currentInvoiceAmount,
    required this.lateFeeTotal,
    required this.credit,
    required this.paidTotal,
    required this.balance,
  });

  factory UnitBalance.fromJson(Map<String, dynamic> j) => UnitBalance(
        unitId: j['unit_id'].toString(),
        priorDebt: parseMoney(j['prior_debt']),
        currentInvoiceAmount: parseMoney(j['current_invoice_amount']),
        lateFeeTotal: parseMoney(j['late_fee_total']),
        credit: parseMoney(j['credit']),
        paidTotal: parseMoney(j['paid_total']),
        balance: parseMoney(j['balance']),
      );

  final String unitId;
  final int priorDebt;
  final int currentInvoiceAmount;
  final int lateFeeTotal;
  final int credit;
  final int paidTotal;
  final int balance;
}

/// Response of `POST /invoices/{id}/pay` — the gateway page the resident
/// must open.
class GatewayStart {
  GatewayStart({
    required this.paymentId,
    required this.paymentUrl,
    required this.amount,
  });

  factory GatewayStart.fromJson(Map<String, dynamic> j) => GatewayStart(
        paymentId: j['payment_id'].toString(),
        paymentUrl: j['payment_url'].toString(),
        amount: parseMoney(j['amount']),
      );

  final String paymentId;
  final String paymentUrl;
  final int amount;
}
