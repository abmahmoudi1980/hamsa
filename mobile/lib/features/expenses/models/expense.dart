/// US6 expense models (T066/T067) — mirrors the backend contracts/api.md
/// "Expenses & Financial Report" section. Enum tokens stay locale-neutral;
/// Persian labels live in shared/widgets/status_labels.dart.
class Expense {
  Expense.fromJson(Map<String, dynamic> json)
      : id = json['id'] as String,
        buildingId = json['building_id'] as String,
        title = json['title'] as String? ?? '',
        category = json['category'] as String? ?? 'other',
        amount = int.tryParse('${json['amount']}') ?? 0,
        expenseDate = json['expense_date'] as String? ?? '',
        description = json['description'] as String?,
        payerPersonId = json['payer_person_id'] as String?,
        receiptFile = json['receipt_file'] as String?,
        approvalStatus = json['approval_status'] as String? ?? 'pending',
        createdAt = json['created_at'] as String?;

  final String id;
  final String buildingId;
  final String title;
  final String category;
  final int amount;

  /// ISO YYYY-MM-DD (wire format; Jalali is presentation-only, R6).
  final String expenseDate;
  final String? description;
  final String? payerPersonId;
  final String? receiptFile;
  final String approvalStatus;
  final String? createdAt;
}

/// FR-027 financial report aggregation for one building. All amounts Toman.
class FinancialReport {
  FinancialReport.fromJson(Map<String, dynamic> json)
      : month = json['month'] as String? ?? '',
        monthlyIncome = int.tryParse('${json['monthly_income']}') ?? 0,
        monthlyExpense = int.tryParse('${json['monthly_expense']}') ?? 0,
        net = int.tryParse('${json['net']}') ?? 0,
        totalDebt = int.tryParse('${json['total_debt']}') ?? 0,
        totalPayments = int.tryParse('${json['total_payments']}') ?? 0,
        totalExpenses = int.tryParse('${json['total_expenses']}') ?? 0;

  final String month; // YYYY-MM as requested
  final int monthlyIncome;
  final int monthlyExpense;
  final int net;
  final int totalDebt;
  final int totalPayments;
  final int totalExpenses;
}
