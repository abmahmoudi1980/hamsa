import 'package:flutter/material.dart';

import 'package:hamsa/core/l10n/app_localizations.dart';

import '../../core/datetime/jalali.dart';

const String _thousandsSeparator = '٬'; // U+066C Arabic thousands separator

/// Formats an integer Toman amount with thousands separators and Persian
/// digits, e.g. 2600000 → `۲٬۶۰۰٬۰۰۰`.
String formatToman(int amount) {
  final negative = amount < 0;
  final digits = amount.abs().toString();
  final buffer = StringBuffer();
  for (var i = 0; i < digits.length; i++) {
    final remaining = digits.length - i;
    buffer.write(digits[i]);
    if (remaining > 1 && (remaining - 1) % 3 == 0) {
      buffer.write(_thousandsSeparator);
    }
  }
  final sign = negative ? '-' : '';
  return toPersianDigits('$sign$buffer');
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
