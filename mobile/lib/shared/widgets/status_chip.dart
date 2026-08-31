import 'package:flutter/material.dart';

import '../../core/l10n/app_localizations.dart';
import 'status_labels.dart';

/// Which domain an enum value belongs to — selects the label/color maps.
enum StatusKind { invoice, period, maintenance, payment, expense }

/// Tonal pill with a status dot showing a wire-enum status and its Persian
/// label. Color semantics come from the per-domain maps; unknown values fall
/// back to the neutral surface.
class StatusChip extends StatelessWidget {
  const StatusChip({super.key, required this.kind, required this.value});

  final StatusKind kind;

  /// Locale-neutral enum token from the API (e.g. `partial`, `in_progress`).
  final String value;

  String _label(BuildContext context) {
    // Maintenance status is fully localized via AppLocalizations (fa.arb).
    if (kind == StatusKind.maintenance) {
      final l10n = AppLocalizations.of(context);
      return switch (value) {
        'new' => l10n.maintenanceStatusNew,
        'under_review' => l10n.maintenanceStatusUnderReview,
        'in_progress' => l10n.maintenanceStatusInProgress,
        'done' => l10n.maintenanceStatusDone,
        'closed' => l10n.maintenanceStatusClosed,
        _ => value,
      };
    }
    return switch (kind) {
      StatusKind.invoice => invoiceStatusLabels[value],
      StatusKind.period => periodStatusLabels[value],
      StatusKind.maintenance => maintenanceStatusLabels[value],
      StatusKind.payment => paymentStatusLabels[value],
      StatusKind.expense => expenseApprovalLabels[value],
    } ??
        value;
  }

  Color? get _color => switch (kind) {
        StatusKind.invoice => invoiceStatusColors[value],
        StatusKind.period => null,
        StatusKind.maintenance => maintenanceStatusColors[value],
        StatusKind.payment => paymentStatusColors[value],
        StatusKind.expense => expenseApprovalColors[value],
      };

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    final color = _color ?? scheme.onSurfaceVariant;
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
      decoration: BoxDecoration(
        color: color.withValues(alpha: 0.12),
        borderRadius: BorderRadius.circular(999),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Container(
            width: 6,
            height: 6,
            decoration: BoxDecoration(
              color: color,
              shape: BoxShape.circle,
            ),
          ),
          Text(
            _label(context),
            style: Theme.of(context).textTheme.labelMedium?.copyWith(
                  color: color,
                  fontWeight: FontWeight.w700,
                ),
          ),
        ],
      ),
    );
  }
}
