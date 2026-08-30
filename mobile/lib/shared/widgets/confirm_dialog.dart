import 'package:flutter/material.dart';

import '../../core/l10n/app_localizations.dart';
import '../../core/theme/app_theme.dart';

/// Standard confirm/cancel dialog: full-width primary confirm button on top,
/// same-size outlined (neutral) cancel button stacked below — direction-safe
/// in RTL. Use [ConfirmDialog.show] to present it and await the result.
class ConfirmDialog extends StatelessWidget {
  const ConfirmDialog({
    super.key,
    required this.title,
    this.message,
    this.content,
    this.confirmLabel,
    this.destructive = false,
  }) : assert(message != null || content != null);

  final String title;

  /// Plain body text; use [content] for interactive bodies.
  final String? message;

  /// Extra widget rendered under [message] (date pickers, text fields, …).
  final Widget? content;

  /// Overrides the confirm label (e.g. `حذف` for destructive actions).
  final String? confirmLabel;

  /// Renders the confirm button in the error color for destructive actions.
  final bool destructive;

  /// Shows the dialog; resolves `true` on confirm, `false`/`null` on cancel.
  static Future<bool?> show(
    BuildContext context, {
    required String title,
    String? message,
    Widget? content,
    String? confirmLabel,
    bool destructive = false,
  }) => showDialog<bool>(
    context: context,
    builder: (_) => ConfirmDialog(
      title: title,
      message: message,
      content: content,
      confirmLabel: confirmLabel,
      destructive: destructive,
    ),
  );

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final scheme = Theme.of(context).colorScheme;
    return AlertDialog(
      title: Text(title),
      content: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          if (message != null) Text(message!),
          if (content != null) ...[
            if (message != null) const SizedBox(height: AppTheme.spaceM),
            content!,
          ],
        ],
      ),
      actionsPadding: const EdgeInsets.fromLTRB(
        AppTheme.spaceXl,
        0,
        AppTheme.spaceXl,
        AppTheme.spaceXl,
      ),
      actions: [
        SizedBox(
          width: double.infinity,
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              FilledButton(
                style: destructive
                    ? FilledButton.styleFrom(
                        backgroundColor: scheme.error,
                        foregroundColor: scheme.onError,
                      )
                    : null,
                onPressed: () => Navigator.pop(context, true),
                child: Text(confirmLabel ?? l10n.confirm),
              ),
              const SizedBox(height: AppTheme.spaceM),
              OutlinedButton(
                style: OutlinedButton.styleFrom(
                  foregroundColor: scheme.onSurfaceVariant,
                ),
                onPressed: () => Navigator.pop(context, false),
                child: Text(l10n.cancel),
              ),
            ],
          ),
        ),
      ],
    );
  }
}
