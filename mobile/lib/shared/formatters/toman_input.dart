import 'package:flutter/services.dart';

import '../../core/datetime/jalali.dart' show toPersianDigits;
import 'money_text.dart';

/// Live-formats a Toman amount while typing: accepts Latin/Persian digits,
/// groups them in threes with the Persian thousands separator, and renders
/// Persian digits — e.g. typing `2600000` shows `۲٬۶۰۰٬۰۰۰`.
/// Pair with [parseToman] to read the value back.
class TomanInputFormatter extends TextInputFormatter {
  const TomanInputFormatter();
  @override
  TextEditingValue formatEditUpdate(
    TextEditingValue oldValue,
    TextEditingValue newValue,
  ) {
    final digits = latinDigitsOnly(newValue.text);
    final base = newValue.selection.baseOffset;
    final caretDigitIndex = base < 0
        ? digits.length
        : _digitCount(newValue.text, base.clamp(0, newValue.text.length));

    final buffer = StringBuffer();
    var caretOffset = 0;
    var seen = 0;
    for (var i = 0; i < digits.length; i++) {
      if (seen == caretDigitIndex) caretOffset = buffer.length;
      buffer.write(digits[i]);
      seen++;
      final remaining = digits.length - seen;
      if (remaining > 0 && remaining % 3 == 0) {
        buffer.write('٬');
      }
    }
    if (seen == caretDigitIndex) caretOffset = buffer.length;

    final text = toPersianDigits(buffer.toString());
    return TextEditingValue(
      text: text,
      selection: TextSelection.collapsed(
        offset: caretOffset.clamp(0, text.length),
      ),
    );
  }

  /// Digits among the first [limit] characters of [text].
  static int _digitCount(String text, int limit) {
    var count = 0;
    var end = 0;
    for (final rune in text.runes) {
      if (end >= limit) break;
      end += String.fromCharCode(rune).length;
      if (_isDigit(rune)) count++;
    }
    return count;
  }

  static bool _isDigit(int rune) =>
      (rune >= 0x30 && rune <= 0x39) ||
      (rune >= 0x0660 && rune <= 0x0669) ||
      (rune >= 0x06F0 && rune <= 0x06F9);
}
