import 'package:flutter/material.dart';

import 'package:hamsa/core/l10n/app_localizations.dart';

import '../../core/datetime/jalali.dart';

const String _thousandsSeparator = '٬'; // U+066C Arabic thousands separator

/// Strips everything but digits from [input], converting Persian
/// (U+06F0–06F9) and Arabic-Indic (U+0660–0669) digits to Latin.
String latinDigitsOnly(String input) {
  final buffer = StringBuffer();
  for (final rune in input.runes) {
    if (rune >= 0x30 && rune <= 0x39) {
      buffer.writeCharCode(rune);
    } else if (rune >= 0x06F0 && rune <= 0x06F9) {
      buffer.writeCharCode(rune - 0x06F0 + 0x30);
    } else if (rune >= 0x0660 && rune <= 0x0669) {
      buffer.writeCharCode(rune - 0x0660 + 0x30);
    }
  }
  return buffer.toString();
}

/// Groups a Latin digit string in threes with the Persian thousands
/// separator, e.g. `2600000` → `2٬600٬000`.
String groupToman(String digits) {
  final buffer = StringBuffer();
  for (var i = 0; i < digits.length; i++) {
    final remaining = digits.length - i;
    buffer.write(digits[i]);
    if (remaining > 1 && (remaining - 1) % 3 == 0) {
      buffer.write(_thousandsSeparator);
    }
  }
  return buffer.toString();
}

/// Parses a user-typed Toman amount: accepts Persian digits and optional
/// separators. Returns null unless the value is a valid integer.
int? parseToman(String input) {
  final normalized =
      latinDigitsOnly(input).replaceAll(_thousandsSeparator, '').replaceAll(',', '');
  return normalized.isEmpty ? null : int.tryParse(normalized);
}

/// Formats an integer Toman amount with thousands separators and Persian
/// digits, e.g. 2600000 → `۲٬۶۰۰٬۰۰۰`.
///
/// The signed number is wrapped in LTR isolates (U+2066…U+2069) so bidi
/// reordering can never detach the minus sign or permute the digit groups —
/// Flutter's bidi mis-nests EN/AN runs around U+066C otherwise (negative
/// balances rendered as `۵٬۷۰۰٬۵-۱۵`). The controls are stripped by
/// [parseToman]/[latinDigitsOnly] on the way back.
String formatToman(int amount) {
  final digits = groupToman(amount.abs().toString());
  return '\u2066${toPersianDigits(amount < 0 ? '-$digits' : digits)}\u2069';
}

/// Amount + «تومان» label rendered with Persian digits everywhere.
class MoneyText extends StatelessWidget {
  const MoneyText({
    super.key,
    required this.amount,
    this.style,
    this.showCurrencySuffix = true,
  });

  /// Integer Toman amount (never fractions — research.md R7).
  final int amount;

  final TextStyle? style;

  final bool showCurrencySuffix;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final text = showCurrencySuffix
        ? '${formatToman(amount)} ${l10n.tomanSuffix}'
        : formatToman(amount);
    return Text(text, style: style);
  }
}
