import 'package:dio/dio.dart';

import 'models/expense.dart';

/// US6 expenses API (T066/T067) — mirrors the backend contracts/api.md
/// "Expenses & Financial Report" section. All routes are manager-only.
class ExpenseRepository {
  ExpenseRepository(this._dio);

  final Dio _dio;

  /// Manager expense list for one building (filter category/approval/date).
  Future<(List<Expense>, int)> buildingExpenses(
    String buildingId, {
    String category = '',
    String approval = '',
    String? from,
    String? to,
    int page = 1,
    int pageSize = 20,
  }) async {
    final res = await _dio.get<Map<String, dynamic>>(
      '/buildings/$buildingId/expenses',
      queryParameters: {
        if (category.isNotEmpty) 'category': category,
        if (approval.isNotEmpty) 'approval': approval,
        if (from != null) 'from': from,
        if (to != null) 'to': to,
        'page': page,
        'page_size': pageSize,
      },
    );
    final items = (res.data?['items'] as List? ?? const [])
        .map((e) => Expense.fromJson(Map<String, dynamic>.from(e as Map)))
        .toList();
    return (items, (res.data?['total'] ?? 0) as int);
  }

  /// Uploads a receipt image via the shared files endpoint (T010) and
  /// returns the file id to send as `receipt_file_id`. The caller passes
  /// the picked file's path/name (XFile from the shared AttachmentPicker).
  Future<String?> uploadReceipt({
    required String path,
    required String name,
  }) async {
    final form = FormData.fromMap({
      'file': await MultipartFile.fromFile(path, filename: name),
    });
    final res = await _dio.post<Map<String, dynamic>>('/files', data: form);
    return res.data?['id'] as String?;
  }

  /// One expense by id (edit form).
  Future<Expense> getExpense(String expenseId) async {
    final res = await _dio.get<Map<String, dynamic>>('/expenses/$expenseId');
    return Expense.fromJson(res.data!);
  }
  Future<Expense> createExpense(
    String buildingId, {
    required String title,
    required String category,
    required int amount,
    required String expenseDate, // ISO YYYY-MM-DD
    String? description,
    String? receiptFileId,
    String approvalStatus = 'pending',
  }) async {
    final res = await _dio.post<Map<String, dynamic>>(
      '/buildings/$buildingId/expenses',
      data: {
        'title': title,
        'category': category,
        'amount': '$amount',
        'expense_date': expenseDate,
        if (description != null && description.isNotEmpty)
          'description': description,
        if (receiptFileId != null) 'receipt_file_id': receiptFileId,
        'approval_status': approvalStatus,
      },
    );
    return Expense.fromJson(res.data!);
  }

  /// Partial update (also the approval workflow: pending → approved/rejected).
  Future<Expense> updateExpense(
    String expenseId, {
    String? title,
    String? category,
    int? amount,
    String? expenseDate,
    String? description,
    String? receiptFileId,
    String? approvalStatus,
  }) async {
    final res = await _dio.put<Map<String, dynamic>>(
      '/expenses/$expenseId',
      data: {
        if (title != null) 'title': title,
        if (category != null) 'category': category,
        if (amount != null) 'amount': '$amount',
        if (expenseDate != null) 'expense_date': expenseDate,
        if (description != null) 'description': description,
        if (receiptFileId != null) 'receipt_file_id': receiptFileId,
        if (approvalStatus != null) 'approval_status': approvalStatus,
      },
    );
    return Expense.fromJson(res.data!);
  }

  /// Soft delete — the ledger row survives server-side (spec §21).
  Future<void> deleteExpense(String expenseId) =>
      _dio.delete<void>('/expenses/$expenseId');

  /// FR-027 financial report for [month] (`YYYY-MM` Gregorian, as stored).
  Future<FinancialReport> financialReport(String buildingId, String month) async {
    final res = await _dio.get<Map<String, dynamic>>(
      '/buildings/$buildingId/financial-report',
      queryParameters: {'month': month},
    );
    return FinancialReport.fromJson(res.data!);
  }
}
