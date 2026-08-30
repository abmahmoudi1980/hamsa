import 'package:hamsa/core/datetime/jalali.dart';
import 'package:hamsa/core/l10n/app_localizations.dart';

import '../formatters/money_text.dart';

/// Form validation helpers. Error messages come from `fa.arb` so all copy
/// stays in a single string file.
abstract final class Validators {
  static final RegExp _mobile = RegExp(r'^09\d{9}$');

  /// Iranian mobile format `09xxxxxxxxx`; accepts Persian-digit input.
  static String? mobile(AppLocalizations l10n, String? value) {
    final normalized = value == null ? '' : fromPersianDigits(value.trim());
    if (!_mobile.hasMatch(normalized)) return l10n.invalidMobile;
    return null;
  }

  /// Non-empty (after trimming) single-line text.
  static String? requiredText(AppLocalizations l10n, String? value) {
    if (value == null || value.trim().isEmpty) return l10n.requiredField;
    return null;
  }

  /// Positive integer amount in Toman; accepts Persian digits and
  /// thousands separators.
  static String? positiveAmount(AppLocalizations l10n, String? value) {
    final parsed = parseToman(value ?? '');
    if (parsed == null || parsed <= 0) return l10n.invalidAmount;
    return null;
  }
}
