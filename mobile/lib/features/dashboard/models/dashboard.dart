/// US9 + US10 dashboard models. Mirrors backend contracts/api.md
/// "Resident Panel P0-08" and "Manager Dashboard P0-09". Enum tokens stay
/// locale-neutral; Persian labels resolve via AppLocalizations in the UI layer.
library;

class LatestInvoice {
  LatestInvoice.fromJson(Map<String, dynamic> json)
      : id = json['id'] as String? ?? '',
        invoiceNumber = json['invoice_number'] as String? ?? '',
        unitNumber = json['unit_number'] as String? ?? '',
        periodTitle = json['period_title'] as String? ?? '',
        finalAmount = _toInt(json['final_amount']),
        dueDate = json['due_date'] as String? ?? '',
        status = json['status'] as String? ?? '';

  final String id;
  final String invoiceNumber;
  final String unitNumber;
  final String periodTitle;
  final int finalAmount;
  final String dueDate;
  final String status;

  bool get hasDueDate => dueDate.isNotEmpty;
}

class LatestRequest {
  LatestRequest.fromJson(Map<String, dynamic> json)
      : id = json['id'] as String? ?? '',
        title = json['title'] as String? ?? '',
        priority = json['priority'] as String? ?? 'normal',
        status = json['status'] as String? ?? 'new',
        createdAt = json['created_at'] as String? ?? '';

  final String id;
  final String title;
  final String priority;
  final String status;
  final String createdAt;
}

class LatestAnnouncement {
  LatestAnnouncement.fromJson(Map<String, dynamic> json)
      : id = json['id'] as String? ?? '',
        title = json['title'] as String? ?? '',
        createdAt = json['created_at'] as String? ?? '',
        isRead = json['is_read'] as bool? ?? false;

  final String id;
  final String title;
  final String createdAt;
  final bool isRead;
}

class HomeSummary {
  HomeSummary.fromJson(Map<String, dynamic> json)
      : payableAmount = _toInt(json['payable_amount']),
        unitCount = _toInt(json['unit_count']),
        latestInvoice = json['latest_invoice'] is Map<String, dynamic>
            ? LatestInvoice.fromJson(
                Map<String, dynamic>.from(json['latest_invoice'] as Map))
            : null,
        openRequestCount = _toInt(json['open_request_count']),
        latestRequest = json['latest_request'] is Map<String, dynamic>
            ? LatestRequest.fromJson(
                Map<String, dynamic>.from(json['latest_request'] as Map))
            : null,
        latestAnnouncements = (json['latest_announcements'] as List? ?? const [])
            .map((e) => LatestAnnouncement.fromJson(
                Map<String, dynamic>.from(e as Map)))
            .toList(),
        unreadAnnouncementCount = _toInt(json['unread_announcement_count']);

  final int payableAmount;
  final int unitCount;
  final LatestInvoice? latestInvoice;
  final int openRequestCount;
  final LatestRequest? latestRequest;
  final List<LatestAnnouncement> latestAnnouncements;
  final int unreadAnnouncementCount;

  bool get hasInvoice => latestInvoice != null;
  bool get hasRequest => latestRequest != null;
}

class ManagerQuickAction {
  ManagerQuickAction.fromJson(Map<String, dynamic> json)
      : key = json['key'] as String? ?? '',
        title = json['title'] as String? ?? '',
        path = json['path'] as String? ?? '';

  final String key;
  final String title;
  final String path;
}

class DashboardAlert {
  DashboardAlert.fromJson(Map<String, dynamic> json)
      : kind = json['kind'] as String? ?? '',
        severity = _toInt(json['severity']),
        unitId = json['unit_id'] as String? ?? '',
        unitNumber = json['unit_number'] as String? ?? '',
        amount = _toInt(json['amount']),
        title = json['title'] as String? ?? '',
        refType = json['ref_type'] as String? ?? '',
        refId = json['ref_id'] as String? ?? '',
        createdAt = json['created_at'] as String? ?? '';

  final String kind;
  final int severity;
  final String unitId;
  final String unitNumber;
  final int amount;
  final String title;
  final String refType;
  final String refId;
  final String createdAt;
}

class BuildingDashboard {
  BuildingDashboard.fromJson(Map<String, dynamic> json)
      : buildingId = json['building_id'] as String? ?? '',
        unitCount = _toInt(json['unit_count']),
        occupiedUnitCount = _toInt(json['occupied_unit_count']),
        debtorUnitCount = _toInt(json['debtor_unit_count']),
        totalDebt = _toInt(json['total_debt']),
        monthIncome = _toInt(json['month_income']),
        monthExpense = _toInt(json['month_expense']),
        openRequests = _toInt(json['open_requests']),
        pendingExpenses = _toInt(json['pending_expenses']),
        month = json['month'] as String? ?? '',
        alerts = (json['alerts'] as List? ?? const [])
            .map((e) =>
                DashboardAlert.fromJson(Map<String, dynamic>.from(e as Map)))
            .toList(),
        quickActions = (json['quick_actions'] as List? ?? const [])
            .map((e) => ManagerQuickAction.fromJson(
                Map<String, dynamic>.from(e as Map)))
            .toList();

  final String buildingId;
  final int unitCount;
  final int occupiedUnitCount;
  final int debtorUnitCount;
  final int totalDebt;
  final int monthIncome;
  final int monthExpense;
  final int openRequests;
  final int pendingExpenses;
  final String month;
  final List<DashboardAlert> alerts;
  final List<ManagerQuickAction> quickActions;
}

int _toInt(dynamic v) {
  if (v == null) return 0;
  if (v is int) return v;
  if (v is num) return v.toInt();
  if (v is String) return int.tryParse(v) ?? 0;
  return 0;
}
