/// US4 billing DTOs (T050). Money is integer Toman serialized as JSON
/// strings on the wire (contracts/api.md) — parsed to int via [parseMoney].
library;

/// Parses an integer Toman wire value that may arrive as `"2600000"` or 2600000.
int parseMoney(dynamic v) =>
    v is int ? v : int.tryParse(v?.toString() ?? '') ?? 0;

class BillingPeriod {
  BillingPeriod({
    required this.id,
    required this.buildingId,
    required this.title,
    required this.startDate,
    required this.endDate,
    required this.dueDate,
    required this.lateFeeType,
    required this.lateFeeValue,
    required this.status,
  });

  factory BillingPeriod.fromJson(Map<String, dynamic> j) => BillingPeriod(
        id: j['id'].toString(),
        buildingId: j['building_id'].toString(),
        title: j['title']?.toString() ?? '',
        startDate: j['start_date'].toString(),
        endDate: j['end_date'].toString(),
        dueDate: j['due_date'].toString(),
        lateFeeType: j['late_fee_type']?.toString() ?? 'none',
        lateFeeValue:
            (j['late_fee_value'] as num? ?? 0).toDouble(),
        status: j['status']?.toString() ?? 'draft',
      );

  final String id;
  final String buildingId;
  final String title;
  final String startDate; // ISO YYYY-MM-DD (Jalali rendered at the edge)
  final String endDate;
  final String dueDate;
  final String lateFeeType; // none | fixed | percent | per_day
  final double lateFeeValue;
  final String status; // draft | calculated | issued | closed

  Map<String, dynamic> toPayload() => {
        'title': title,
        'start_date': startDate,
        'end_date': endDate,
        'due_date': dueDate,
        'late_fee_type': lateFeeType,
        'late_fee_value': lateFeeValue,
      };
}

class ComboWeight {
  ComboWeight({required this.method, required this.weight});

  factory ComboWeight.fromJson(Map<String, dynamic> j) => ComboWeight(
        method: j['method'].toString(),
        weight: (j['weight'] as num? ?? 0).toInt(),
      );

  final String method;
  final int weight;

  Map<String, dynamic> toPayload() => {'method': method, 'weight': weight};
}

class CostItem {
  CostItem({
    required this.id,
    required this.periodId,
    required this.title,
    required this.totalAmount,
    required this.method,
    this.fixedAmountPerUnit,
    this.comboWeights = const [],
    this.includeVacant = false,
    this.unitIds = const [],
  });

  factory CostItem.fromJson(Map<String, dynamic> j) => CostItem(
        id: j['id'].toString(),
        periodId: j['period_id'].toString(),
        title: j['title']?.toString() ?? '',
        totalAmount: parseMoney(j['total_amount']),
        method: j['method'].toString(),
        fixedAmountPerUnit: j['fixed_amount_per_unit'] == null
            ? null
            : parseMoney(j['fixed_amount_per_unit']),
        comboWeights: (j['combo_weights'] as List? ?? const [])
            .map((e) => ComboWeight.fromJson(Map<String, dynamic>.from(e as Map)))
            .toList(),
        includeVacant: j['include_vacant'] as bool? ?? false,
        unitIds: (j['unit_ids'] as List? ?? const [])
            .map((e) => e.toString())
            .toList(),
      );

  final String id;
  final String periodId;
  final String title;
  final int totalAmount;
  final String method; // equal | per_occupant | per_area | fixed |
                      // specific_units | combined
  final int? fixedAmountPerUnit;
  final List<ComboWeight> comboWeights;
  final bool includeVacant;
  final List<String> unitIds;
}

/// One unit's share of one cost item in the preview (BR-08: exact + rounded
/// surfaced for review).
class PreviewShare {
  PreviewShare({
    required this.unitId,
    required this.exactShare,
    required this.roundedShare,
    this.unitNumber = '',
  });

  factory PreviewShare.fromJson(Map<String, dynamic> j) => PreviewShare(
        unitId: j['unit_id'].toString(),
        unitNumber: j['unit_number']?.toString() ?? '',
        exactShare: j['exact_share']?.toString() ?? '',
        roundedShare: parseMoney(j['rounded_share']),
      );

  final String unitId;
  final String unitNumber;
  final String exactShare; // decimal string from NUMERIC(20,4)
  final int roundedShare;
}

/// One cost item's reviewable breakdown with the reconciliation flag (BR-09).
/// The backend embeds the full CostItem in each preview item, so the editor
/// can be opened in edit mode straight from this screen.
class PreviewItem {
  PreviewItem({
    required this.id,
    required this.title,
    required this.method,
    required this.totalUsed,
    required this.shares,
    required this.reconciled,
    required this.costItem,
  });

  factory PreviewItem.fromJson(Map<String, dynamic> j) => PreviewItem(
        id: j['id'].toString(),
        title: j['title']?.toString() ?? '',
        method: j['method'].toString(),
        totalUsed: parseMoney(j['total_used']),
        shares: (j['shares'] as List? ?? const [])
            .map((e) => PreviewShare.fromJson(Map<String, dynamic>.from(e as Map)))
            .toList(),
        reconciled: j['reconciled'] as bool? ?? false,
        costItem: CostItem.fromJson(j),
      );

  final String id;
  final String title;
  final String method;
  final int totalUsed;
  final List<PreviewShare> shares;
  final bool reconciled;
  final CostItem costItem;
}

class InvoiceItem {
  InvoiceItem({
    required this.id,
    required this.kind,
    required this.title,
    required this.amount,
    this.method,
  });

  factory InvoiceItem.fromJson(Map<String, dynamic> j) => InvoiceItem(
        id: j['id'].toString(),
        kind: j['kind']?.toString() ?? 'charge',
        title: j['title']?.toString() ?? '',
        amount: parseMoney(j['amount']),
        method: j['method']?.toString(),
      );

  final String id;
  final String kind; // charge | late_fee | adjustment
  final String title;
  final int amount;
  final String? method;
}

class InvoiceAdjustment {
  InvoiceAdjustment({
    required this.id,
    required this.kind,
    required this.amount,
    required this.reason,
  });

  factory InvoiceAdjustment.fromJson(Map<String, dynamic> j) =>
      InvoiceAdjustment(
        id: j['id'].toString(),
        kind: j['kind']?.toString() ?? 'debit',
        amount: parseMoney(j['amount']),
        reason: j['reason']?.toString() ?? '',
      );

  final String id;
  final String kind; // debit | credit
  final int amount;
  final String reason;
}

class Invoice {
  Invoice({
    required this.id,
    required this.invoiceNumber,
    required this.unitId,
    required this.baseAmount,
    required this.priorDebt,
    required this.lateFeeAmount,
    required this.creditAmount,
    required this.finalAmount,
    required this.status,
    required this.paidAmount,
    required this.items,
    this.invoiceNumberPresent = true,
    this.unitNumber = '',
    this.periodTitle = '',
    this.issueDate,
    this.dueDate,
    this.adjustments = const [],
  });

  factory Invoice.fromJson(Map<String, dynamic> j) => Invoice(
        id: j['id'].toString(),
        invoiceNumber: j['invoice_number']?.toString() ?? '',
        invoiceNumberPresent: j['invoice_number'] != null,
        unitId: j['unit_id'].toString(),
        unitNumber: j['unit_number']?.toString() ?? '',
        periodTitle: j['period_title']?.toString() ?? '',
        baseAmount: parseMoney(j['base_amount']),
        priorDebt: parseMoney(j['prior_debt']),
        lateFeeAmount: parseMoney(j['late_fee_amount']),
        creditAmount: parseMoney(j['credit_amount']),
        finalAmount: parseMoney(j['final_amount']),
        issueDate: j['issue_date']?.toString(),
        dueDate: j['due_date']?.toString(),
        status: j['status']?.toString() ?? 'unpaid',
        paidAmount: parseMoney(j['paid_amount']),
        items: (j['items'] as List? ?? const [])
            .map((e) => InvoiceItem.fromJson(Map<String, dynamic>.from(e as Map)))
            .toList(),
        adjustments: (j['adjustments'] as List? ?? const [])
            .map((e) =>
                InvoiceAdjustment.fromJson(Map<String, dynamic>.from(e as Map)))
            .toList(),
      );

  final String id;
  final String invoiceNumber;
  final bool invoiceNumberPresent; // drafts carry a DRAFT- placeholder
  final String unitId;
  final String unitNumber;
  final String periodTitle;
  final int baseAmount;
  final int priorDebt;
  final int lateFeeAmount;
  final int creditAmount;
  final int finalAmount;
  final String? issueDate;
  final String? dueDate;
  final String status; // unpaid | partial | paid | expired | cancelled
  final int paidAmount;
  final List<InvoiceItem> items;
  final List<InvoiceAdjustment> adjustments;
}

/// `GET /periods/{id}/preview` response (BR-08): per cost item shares with
/// reconciliation, plus the resulting per-unit draft invoices.
class PeriodPreview {
  PeriodPreview({
    required this.period,
    required this.items,
    required this.invoices,
    required this.reconciled,
  });

  factory PeriodPreview.fromJson(Map<String, dynamic> j) => PeriodPreview(
        period:
            BillingPeriod.fromJson(Map<String, dynamic>.from(j['period'] as Map)),
        items: (j['items'] as List? ?? const [])
            .map((e) => PreviewItem.fromJson(Map<String, dynamic>.from(e as Map)))
            .toList(),
        invoices: (j['invoices'] as List? ?? const [])
            .map((e) => Invoice.fromJson(Map<String, dynamic>.from(e as Map)))
            .toList(),
        reconciled: j['reconciled'] as bool? ?? false,
      );

  final BillingPeriod period;
  final List<PreviewItem> items;
  final List<Invoice> invoices;
  final bool reconciled;
}
