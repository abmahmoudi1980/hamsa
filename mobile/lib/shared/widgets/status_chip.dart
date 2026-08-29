import 'package:flutter/material.dart';

import 'status_labels.dart';

/// Which domain an enum value belongs to — selects the label/color maps.
enum StatusKind { invoice, period, maintenance, payment }

/// Pill showing a wire-enum status with its Persian label and accent color.
class StatusChip extends StatelessWidget {
  const StatusChip({super.key, required this.kind, required this.value});

  final StatusKind kind;

  /// Locale-neutral enum token from the API (e.g. `partial`, `in_progress`).
  final String value;

  String get _label => switch (kind) {
        StatusKind.invoice => invoiceStatusLabels[value],
        StatusKind.period => periodStatusLabels[value],
        StatusKind.maintenance => maintenanceStatusLabels[value],
        StatusKind.payment => paymentStatusLabels[value],
      } ?? value;

  Color? get _color => switch (kind) {
        StatusKind.invoice => invoiceStatusColors[value],
        StatusKind.period => null,
        StatusKind.maintenance => maintenanceStatusColors[value],
        StatusKind.payment => paymentStatusColors[value],
      };

  @override
  Widget build(BuildContext context) {
    final color = _color;
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
      decoration: BoxDecoration(
        color: (color ?? Theme.of(context).colorScheme.primary).withValues(alpha: 0.12),
        borderRadius: BorderRadius.circular(999),
      ),
      child: Text(
        _label,
        style: Theme.of(context).textTheme.labelMedium?.copyWith(
              color: color ?? Theme.of(context).colorScheme.primary,
              fontWeight: FontWeight.w600,
            ),
      ),
    );
  }
}
